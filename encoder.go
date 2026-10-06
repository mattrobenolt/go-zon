package zon

import (
	"fmt"
	"io"
)

// frame is the state of one open container.
type frame struct {
	kind     containerKind
	wrap     bool   // one field per line
	elide    bool   // spaces elided: an array of one declared element
	empty    bool   // no fields written yet
	phase    phase  // what the container expects next
	lastName string // most recent field name, for error messages
}

// containerKind distinguishes the three ZON container forms.
type containerKind uint8

const (
	kindStruct containerKind = iota
	kindArray
	kindUnion
)

func (k containerKind) String() string {
	switch k {
	case kindStruct:
		return "struct"
	case kindArray:
		return "array"
	case kindUnion:
		return "union"
	}
	return "unknown"
}

// phase is what a container expects next.
type phase uint8

const (
	phaseName    phase = iota // struct: a field name or EndStruct
	phaseValue                // array: a value, preceded by a separator
	phasePayload              // struct or union: a value directly, after a
	// field name or ".tag = " prefix with no separator
	phaseEnd // union: EndUnion
)

// Encoder writes a single ZON value to an output stream.
//
// A ZON document is exactly one value, so the encoder validates every write
// against the current position and flushes to the writer when the root value
// completes. After that, every method returns an error.
//
// Output is written with standard Zig whitespace by default: 4-space
// indentation and one field per line for containers opened by the Begin
// methods. [Compact] writes the minimal form instead.
//
// An Encoder is not safe for concurrent use.
type Encoder struct {
	w          io.Writer
	buf        []byte
	flushed    int
	indent     string
	whitespace bool
	level      int
	stack      []frame
	done       bool
}

// NewEncoder returns an Encoder that writes ZON to w.
func NewEncoder(w io.Writer, opts ...Option) *Encoder {
	e := &Encoder{
		w:          w,
		buf:        make([]byte, 0, 256),
		indent:     "    ",
		whitespace: true,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Option configures an [Encoder].
type Option func(*Encoder)

// WithIndent sets the prefix written once per indentation level.
// It has no effect together with [Compact].
func WithIndent(indent string) Option {
	return func(e *Encoder) { e.indent = indent }
}

// Compact disables all whitespace, writing the minimal form,
// e.g. ".{.x=1,.y=2}" instead of ".{\n    .x = 1,\n    .y = 2,\n}".
func Compact() Option {
	return func(e *Encoder) { e.whitespace = false }
}

// ContainerOpt configures how one container is written.
type ContainerOpt struct {
	mode   wrapMode
	fields int
}

type wrapMode uint8

const (
	wrapAlways wrapMode = iota
	wrapNever
	wrapAuto
)

// Inline writes the container on a single line, e.g. ".{ .x = 1, .y = 2 }".
func Inline() ContainerOpt { return ContainerOpt{mode: wrapNever} }

// Fields declares the number of fields in the container and applies the
// standard rule: more than two fields wrap onto one line each, two or fewer
// stay inline, and a one-element array also elides its spaces: ".{1}".
func Fields(n int) ContainerOpt { return ContainerOpt{mode: wrapAuto, fields: n} }

// resolveOpts picks the wrap behavior for a container. With no options the
// container wraps, matching std.zon.Serializer's manual default; with several
// options the last wins.
func resolveOpts(opts []ContainerOpt) ContainerOpt {
	if len(opts) == 0 {
		return ContainerOpt{mode: wrapAlways}
	}
	return opts[len(opts)-1]
}

// BeginStruct starts a struct literal, writing ".{". Each field is written
// with [Encoder.Name] followed by a value. [Encoder.EndStruct] closes it.
func (e *Encoder) BeginStruct(opts ...ContainerOpt) error {
	if err := e.beforeValue("BeginStruct"); err != nil {
		return err
	}
	e.beginContainer(kindStruct, true, resolveOpts(opts))
	return nil
}

// BeginArray starts an array literal, writing ".{". Values follow, then
// [Encoder.EndArray] closes it.
func (e *Encoder) BeginArray(opts ...ContainerOpt) error {
	if err := e.beforeValue("BeginArray"); err != nil {
		return err
	}
	e.beginContainer(kindArray, false, resolveOpts(opts))
	return nil
}

// BeginUnion starts a tagged union literal with a non-void payload, writing
// ".{ .tag = ". The payload value follows, then [Encoder.EndUnion] closes it.
// A union with a void payload has no container form: write the tag with
// [Encoder.Enum] instead.
func (e *Encoder) BeginUnion(tag string) error {
	if tag == "" {
		return e.errorAt("BeginUnion", "name must not be empty")
	}
	if err := e.beforeValue("BeginUnion"); err != nil {
		return err
	}
	e.beginContainer(kindUnion, true, ContainerOpt{mode: wrapNever})
	f := &e.stack[len(e.stack)-1]
	e.writeSeparator(f)
	e.writeFieldName(f, tag)
	return nil
}

// Name writes the field name prefix ".name = " inside a struct. The name is
// escaped as a Zig identifier when it cannot be written bare.
func (e *Encoder) Name(name string) error {
	if name == "" {
		return e.errorAt("Name", "name must not be empty")
	}
	if e.done {
		return e.errorAt("Name", errDone)
	}
	if len(e.stack) == 0 {
		return e.errorAt("Name", "field name outside of a struct")
	}
	f := &e.stack[len(e.stack)-1]
	switch f.kind {
	case kindArray:
		return e.errorAt("Name", "field name inside an array")
	case kindUnion:
		return e.errorAt("Name", "field name inside a union")
	case kindStruct:
	}
	switch f.phase {
	case phaseName:
	case phasePayload:
		return e.errorAt("Name", "expected a value for field ."+f.lastName+", got a field name")
	case phaseValue, phaseEnd:
		// unreachable: arrays and unions reject names above
	}
	f.phase = phasePayload
	e.writeSeparator(f)
	e.writeFieldName(f, name)
	return nil
}

// EndStruct closes the innermost open struct.
func (e *Encoder) EndStruct() error {
	return e.endContainer(kindStruct, "EndStruct")
}

// EndArray closes the innermost open array.
func (e *Encoder) EndArray() error {
	return e.endContainer(kindArray, "EndArray")
}

// EndUnion closes the innermost open union after its payload.
func (e *Encoder) EndUnion() error {
	return e.endContainer(kindUnion, "EndUnion")
}

// beforeValue validates that a value write is legal at the current position
// and writes the field separator for the value. Failed calls leave the state
// untouched.
func (e *Encoder) beforeValue(op string) error {
	if e.done {
		return e.errorAt(op, errDone)
	}
	if len(e.stack) == 0 {
		return nil // the root value
	}
	f := &e.stack[len(e.stack)-1]
	switch f.phase {
	case phaseValue:
		e.writeSeparator(f)
		return nil
	case phasePayload:
		return nil
	case phaseName:
		return e.errorAt(op, "expected a field name, got a value")
	case phaseEnd:
		return e.errorAt(op, "union payload already written; expected EndUnion")
	}
	return nil
}

// afterValue updates the container state after a complete value. At the root
// it finishes the document.
func (e *Encoder) afterValue() error {
	if len(e.stack) == 0 {
		return e.finishRoot()
	}
	e.completeValue()
	return nil
}

// completeValue marks the innermost container as having received its value.
func (e *Encoder) completeValue() {
	f := &e.stack[len(e.stack)-1]
	switch f.kind {
	case kindStruct:
		f.phase = phaseName
	case kindUnion:
		f.phase = phaseEnd
	case kindArray:
		// stays in phaseValue
	}
}

func (e *Encoder) beginContainer(kind containerKind, named bool, opt ContainerOpt) {
	var wrap, elide bool
	switch opt.mode {
	case wrapAlways:
		wrap = true
	case wrapNever:
	case wrapAuto:
		wrap = opt.fields > 2
		elide = !named && opt.fields == 1
	}
	if wrap {
		e.level++
	}
	e.buf = append(e.buf, ".{"...)
	ph := phaseValue
	switch kind {
	case kindStruct:
		ph = phaseName
	case kindArray:
		ph = phaseValue
	case kindUnion:
		ph = phasePayload
	}
	e.stack = append(e.stack, frame{
		kind:  kind,
		wrap:  wrap,
		elide: elide,
		empty: true,
		phase: ph,
	})
}

func (e *Encoder) endContainer(kind containerKind, op string) error {
	if e.done {
		return e.errorAt(op, errDone)
	}
	if len(e.stack) == 0 {
		return e.errorAt(op, "no open container")
	}
	f := &e.stack[len(e.stack)-1]
	if f.kind != kind {
		return e.errorAt(op, op+" does not match open "+f.kind.String())
	}
	switch {
	case kind == kindStruct && f.phase == phasePayload:
		return e.errorAt(op, "field ."+f.lastName+" has no value")
	case kind == kindUnion && f.phase != phaseEnd:
		return e.errorAt(op, "union ."+f.lastName+" requires a payload before EndUnion;"+
			" write the tag with Enum for a void payload")
	}
	if f.wrap {
		e.level--
	}
	if !f.empty {
		if f.wrap {
			if e.whitespace {
				e.buf = append(e.buf, ',')
			}
			e.writeNewline()
			e.writeIndent()
		} else if !f.elide {
			e.writeSpace()
		}
	}
	e.buf = append(e.buf, '}')
	e.stack = e.stack[:len(e.stack)-1]
	if len(e.stack) == 0 {
		return e.finishRoot()
	}
	e.completeValue()
	return nil
}

// writeSeparator writes the separator before a field: a comma when the
// container is not empty, then wrapping whitespace.
func (e *Encoder) writeSeparator(f *frame) {
	if !f.empty {
		e.buf = append(e.buf, ',')
	}
	f.empty = false
	if f.wrap {
		e.writeNewline()
		e.writeIndent()
	} else if !f.elide {
		e.writeSpace()
	}
}

// writeFieldName writes ".name = " with the name escaped as an identifier.
func (e *Encoder) writeFieldName(f *frame, name string) {
	e.buf = append(e.buf, '.')
	e.buf = appendIdent(e.buf, name)
	e.writeSpace()
	e.buf = append(e.buf, '=')
	e.writeSpace()
	f.lastName = name
}

// finishRoot flushes the completed root value to the writer. The encoder
// accepts no further writes.
func (e *Encoder) finishRoot() error {
	e.done = true
	if _, err := e.w.Write(e.buf); err != nil {
		return fmt.Errorf("zon: write: %w", err)
	}
	e.flushed += len(e.buf)
	e.buf = e.buf[:0]
	return nil
}

func (e *Encoder) writeNewline() {
	if e.whitespace {
		e.buf = append(e.buf, '\n')
	}
}

func (e *Encoder) writeIndent() {
	if e.whitespace {
		for i := 0; i < e.level; i++ {
			e.buf = append(e.buf, e.indent...)
		}
	}
}

func (e *Encoder) writeSpace() {
	if e.whitespace {
		e.buf = append(e.buf, ' ')
	}
}

const errDone = "document already complete"

func (e *Encoder) errorAt(op string, reason string) error {
	return &Error{Op: op, Offset: e.flushed + len(e.buf), Reason: reason}
}
