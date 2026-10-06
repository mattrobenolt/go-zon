package zon

import (
	"math"
	"math/big"
	"strconv"
)

// Bool writes the literal true or false.
func (e *Encoder) Bool(b bool) error {
	if err := e.beforeValue("Bool"); err != nil {
		return err
	}
	e.buf = strconv.AppendBool(e.buf, b)
	return e.afterValue()
}

// Int writes a signed integer literal in decimal.
func (e *Encoder) Int(i int64) error {
	if err := e.beforeValue("Int"); err != nil {
		return err
	}
	e.buf = strconv.AppendInt(e.buf, i, 10)
	return e.afterValue()
}

// Uint writes an unsigned integer literal in decimal.
func (e *Encoder) Uint(u uint64) error {
	if err := e.beforeValue("Uint"); err != nil {
		return err
	}
	e.buf = strconv.AppendUint(e.buf, u, 10)
	return e.afterValue()
}

// BigInt writes an arbitrary-precision integer literal in decimal. ZON
// integer literals are unbounded, so this covers widths Go has no type for.
func (e *Encoder) BigInt(i *big.Int) error {
	if i == nil {
		return e.errorAt("BigInt", "nil *big.Int")
	}
	if err := e.beforeValue("BigInt"); err != nil {
		return err
	}
	e.buf = i.Append(e.buf, 10)
	return e.afterValue()
}

// Float writes a floating-point literal. NaN and infinities are written as
// the ZON keywords nan, inf, and -inf, negative zero as -0.0, and everything
// else as the shortest decimal form that round-trips.
func (e *Encoder) Float(f float64) error {
	if err := e.beforeValue("Float"); err != nil {
		return err
	}
	switch {
	case math.IsNaN(f):
		e.buf = append(e.buf, "nan"...)
	case math.IsInf(f, 1):
		e.buf = append(e.buf, "inf"...)
	case math.IsInf(f, -1):
		e.buf = append(e.buf, "-inf"...)
	case f == 0 && math.Signbit(f):
		e.buf = append(e.buf, "-0.0"...)
	default:
		e.buf = strconv.AppendFloat(e.buf, f, 'g', -1, 64)
	}
	return e.afterValue()
}

// Null writes the literal null.
func (e *Encoder) Null() error {
	if err := e.beforeValue("Null"); err != nil {
		return err
	}
	e.buf = append(e.buf, "null"...)
	return e.afterValue()
}

// String writes a double-quoted string literal. Non-printable and non-ASCII
// bytes are escaped as \xNN, so any string round-trips.
func (e *Encoder) String(s string) error {
	if err := e.beforeValue("String"); err != nil {
		return err
	}
	e.buf = append(e.buf, '"')
	e.buf = appendStringEscaped(e.buf, s)
	e.buf = append(e.buf, '"')
	return e.afterValue()
}

// MultilineString writes a Zig multiline string literal, the \\ form, which
// keeps embedded newlines readable without escapes. A carriage return must
// be followed by a newline; the pair is normalized to a single newline.
// At the document root no leading or trailing newline is written.
func (e *Encoder) MultilineString(s string) error {
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' && (i+1 == len(s) || s[i+1] != '\n') {
			return e.errorAt("MultilineString", "carriage return not followed by a newline")
		}
	}
	if err := e.beforeValue("MultilineString"); err != nil {
		return err
	}
	if len(e.stack) > 0 {
		e.writeNewline()
		e.writeIndent()
	}
	e.buf = append(e.buf, '\\', '\\')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\r' {
			continue
		}
		e.buf = append(e.buf, c)
		if c == '\n' {
			e.writeIndent()
			e.buf = append(e.buf, '\\', '\\')
		}
	}
	if len(e.stack) > 0 {
		e.buf = append(e.buf, '\n')
		e.writeIndent()
	}
	return e.afterValue()
}

// CodePoint writes a Unicode code point literal, e.g. 'a', '\x07', or
// '\u{26a1}'. This is ZON's narrowest character form; Go has no equivalent
// literal.
func (e *Encoder) CodePoint(r rune) error {
	if !isValidCodePoint(r) {
		return e.errorAt("CodePoint", "invalid code point")
	}
	if err := e.beforeValue("CodePoint"); err != nil {
		return err
	}
	e.buf = append(e.buf, '\'')
	e.buf = appendCharEscaped(e.buf, r)
	e.buf = append(e.buf, '\'')
	return e.afterValue()
}

// Enum writes an enum literal, ".name", with the name escaped as a Zig
// identifier when it cannot be written bare. A tagged union with a void
// payload serializes to exactly this form, so this method writes both.
func (e *Encoder) Enum(name string) error {
	if name == "" {
		return e.errorAt("Enum", "name must not be empty")
	}
	if err := e.beforeValue("Enum"); err != nil {
		return err
	}
	e.buf = append(e.buf, '.')
	e.buf = appendIdent(e.buf, name)
	return e.afterValue()
}

// Raw writes pre-encoded ZON verbatim at the current position. The caller
// guarantees that b is valid ZON there.
func (e *Encoder) Raw(b []byte) error {
	if err := e.beforeValue("Raw"); err != nil {
		return err
	}
	e.buf = append(e.buf, b...)
	return e.afterValue()
}
