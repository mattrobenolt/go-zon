package zon

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"slices"
	"strings"
	"sync"
)

// maxMarshalDepth bounds reflection recursion. Recursive Go types would
// otherwise recurse until the stack overflows; Zig's own serializer refuses
// them at compile time, so marshaling one is an error here.
const maxMarshalDepth = 1024

var (
	valueType  = reflect.TypeOf((*Value)(nil)).Elem()
	bigIntType = reflect.TypeOf(big.Int{})
	bigFloat   = reflect.TypeOf(big.Float{})
)

// Marshal writes v as a ZON document and returns the bytes.
//
// See [Encoder.WriteAny] for how Go values map to ZON.
func Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := MarshalWrite(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalWrite writes v as a single ZON document to w.
//
// See [Encoder.WriteAny] for how Go values map to ZON.
func MarshalWrite(w io.Writer, v any, opts ...Option) error {
	return NewEncoder(w, opts...).WriteAny(v)
}

// WriteAny writes any Go value at the current position, using reflection.
//
// Values implementing [Value] write as themselves, so [Enum], [Union],
// [CodePoint], [Multiline], [Raw], and friends work directly as struct field
// types. A math/big.Int writes as an unbounded integer literal.
//
// Otherwise the Go value maps to ZON as follows:
//
//   - nil pointers and interfaces write null; non-nil ones write their value
//   - bool, integers, floats, and strings write the matching literals
//   - structs write struct literals; see below for field names
//   - maps write struct literals, one field per key, keys sorted
//   - slices and arrays write array literals; a []byte writes a string literal
//
// Struct fields resolve in declaration order: unexported fields are skipped,
// a "zon:\"-\"" tag skips the field, and otherwise the first comma-separated
// tag component sets the field name ("zon:\"name,omitempty\"") with the rest
// as options, falling back to the snake_case Go name. "omitempty" skips
// fields holding the zero value of their type, so an absent field means the
// consumer's default. Anonymous struct fields, tagged or not, are flattened
// into the parent like the encoding/json packages do.
//
// Recursive types, chan, func, complex, and math/big.Float values are
// unsupported and return an error.
func (e *Encoder) WriteAny(v any) error {
	return e.writeAny(reflect.ValueOf(v), 0)
}

func (e *Encoder) writeAny(v reflect.Value, depth int) error {
	if depth > maxMarshalDepth {
		return e.errorAt(
			"WriteAny",
			"marshal depth exceeded 1024 levels; recursive types are not supported",
		)
	}
	if !v.IsValid() {
		return e.Null() // a nil interface
	}
	switch v.Type() {
	case bigIntType:
		b := v.Interface().(big.Int)
		return e.BigInt(&b)
	case bigFloat:
		return e.errorAt(
			"WriteAny",
			"unsupported type math/big.Float; convert to float64 or use zon.Float",
		)
	}
	if v.Type().Implements(valueType) {
		return e.WriteValue(v.Interface().(Value))
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return e.Null()
		}
		return e.writeAny(v.Elem(), depth)
	case reflect.Bool:
		return e.Bool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return e.Int(v.Int())
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		return e.Uint(v.Uint())
	case reflect.Float32, reflect.Float64:
		return e.Float(v.Float())
	case reflect.String:
		return e.String(v.String())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return e.String(string(v.Bytes()))
		}
		return e.writeList(v, depth)
	case reflect.Array:
		return e.writeList(v, depth)
	case reflect.Map:
		return e.writeMap(v, depth)
	case reflect.Struct:
		return e.writeGoStruct(v, depth)
	default:
		return e.errorAt("WriteAny", "unsupported type "+v.Type().String())
	}
}

func (e *Encoder) writeList(v reflect.Value, depth int) error {
	if err := e.BeginArray(Fields(v.Len())); err != nil {
		return err
	}
	for i := 0; i < v.Len(); i++ {
		if err := e.writeAny(v.Index(i), depth+1); err != nil {
			return err
		}
	}
	return e.EndArray()
}

func (e *Encoder) writeMap(v reflect.Value, depth int) error {
	if v.Type().Key().Kind() != reflect.String {
		return e.errorAt("WriteAny", "map key type must be string, got "+v.Type().Key().String())
	}
	keys := v.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int {
		return strings.Compare(a.String(), b.String())
	})
	if err := e.BeginStruct(Fields(len(keys))); err != nil {
		return err
	}
	for _, k := range keys {
		if err := e.Name(k.String()); err != nil {
			return err
		}
		if err := e.writeAny(v.MapIndex(k), depth+1); err != nil {
			return err
		}
	}
	return e.EndStruct()
}

func (e *Encoder) writeGoStruct(v reflect.Value, depth int) error {
	mt := marshalTypeOf(v.Type())
	if mt.err != nil {
		return e.errorAt("WriteAny", mt.err.Error())
	}
	// Resolve the writable fields first: omitempty drops zero values, so the
	// written count decides the wrap rule.
	writable := make([]resolvedField, 0, len(mt.fields))
	for _, f := range mt.fields {
		fv := fieldByIndex(v, f.index)
		if !fv.IsValid() || (f.omitEmpty && fv.IsZero()) {
			continue
		}
		writable = append(writable, resolvedField{name: f.name, val: fv})
	}
	if err := e.BeginStruct(Fields(len(writable))); err != nil {
		return err
	}
	for _, w := range writable {
		if err := e.Name(w.name); err != nil {
			return err
		}
		if err := e.writeAny(w.val, depth+1); err != nil {
			return err
		}
	}
	return e.EndStruct()
}

type resolvedField struct {
	name string
	val  reflect.Value
}

type marshalField struct {
	name      string
	index     []int
	omitEmpty bool
}

// marshalType is the resolved fields of one struct type.
type marshalType struct {
	fields []*marshalField
	err    error // invalid zon tag, discovered while resolving
}

// fieldCache caches resolved field lists per struct type.
var fieldCache sync.Map // reflect.Type -> *marshalType

func marshalTypeOf(t reflect.Type) *marshalType {
	if cached, ok := fieldCache.Load(t); ok {
		return cached.(*marshalType)
	}
	mt := new(marshalType)
	mt.fields, mt.err = structFields(t, nil)
	fieldCache.Store(t, mt)
	return mt
}

// structFields resolves the ZON fields of a struct: exported fields in
// declaration order, anonymous structs flattened, names from the zon tag or
// snake_case conversion. The first comma-separated tag component names the
// field; the rest are options: "omitempty" skips zero values.
func structFields(t reflect.Type, prefix []int) ([]*marshalField, error) {
	var fields []*marshalField
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		name, opts := cutOptions(f.Tag.Get("zon"))
		if name == "-" {
			continue
		}
		omitEmpty := false
		for _, opt := range opts {
			switch opt {
			case "omitempty":
				omitEmpty = true
			case "":
				// tolerate empty components, like a trailing comma
			default:
				return nil, fmt.Errorf("unknown option %q in zon tag on field %s of type %s",
					opt, f.Name, t.String())
			}
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if f.Anonymous && ft.Kind() == reflect.Struct && name == "" {
			embedded, err := structFields(ft, appendIndex(prefix, f.Index))
			if err != nil {
				return nil, err
			}
			fields = append(fields, embedded...)
			continue
		}
		if name == "" {
			name = snakeCase(f.Name)
		}
		fields = append(fields, &marshalField{
			name:      name,
			index:     appendIndex(prefix, f.Index),
			omitEmpty: omitEmpty,
		})
	}
	return fields, nil
}

// cutOptions splits a zon tag into its name component and option components:
// "metrics_listen,omitempty" is the name "metrics_listen" with one option.
func cutOptions(tag string) (name string, opts []string) {
	name, rest, _ := strings.Cut(tag, ",")
	if rest == "" {
		return name, nil
	}
	return name, strings.Split(rest, ",")
}

// fieldByIndex walks to a field, dereferencing pointers on the way. It
// returns the zero Value when it crosses a nil pointer.
func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

func appendIndex(prefix []int, index []int) []int {
	next := make([]int, 0, len(prefix)+len(index))
	next = append(next, prefix...)
	return append(next, index...)
}

// snakeCase converts a Go name to the Zig field naming convention:
// "HTTPServer" becomes "http_server", "URLPath" becomes "url_path",
// "ID" becomes "id", and "X509" becomes "x509".
func snakeCase(name string) string {
	var sb strings.Builder
	sb.Grow(len(name) + 4)
	for i := 0; i < len(name); i++ {
		c := name[i]
		if isUpperASCII(c) {
			if i > 0 && (isLowerASCII(name[i-1]) || isDigitASCII(name[i-1]) ||
				(isUpperASCII(name[i-1]) && i+1 < len(name) && isLowerASCII(name[i+1]))) {
				sb.WriteByte('_')
			}
			c += 'a' - 'A'
		}
		sb.WriteByte(c)
	}
	return sb.String()
}

func isLowerASCII(c byte) bool { return 'a' <= c && c <= 'z' }
func isUpperASCII(c byte) bool { return 'A' <= c && c <= 'Z' }
func isDigitASCII(c byte) bool { return '0' <= c && c <= '9' }
