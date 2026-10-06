package zon

import (
	"bytes"
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
// a "zon:\"-\"" tag skips the field, a "zon:\"name\"" tag sets the field name,
// and otherwise the Go name converts to snake_case ("HTTPServer" becomes
// "http_server"). Anonymous struct fields, tagged or not, are flattened into
// the parent like the encoding/json packages do.
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
	fields := marshalFields(v.Type())
	if err := e.BeginStruct(Fields(len(fields))); err != nil {
		return err
	}
	for _, f := range fields {
		fv := fieldByIndex(v, f.index)
		if !fv.IsValid() {
			continue // a nil anonymous pointer: nothing to flatten
		}
		if err := e.Name(f.name); err != nil {
			return err
		}
		if err := e.writeAny(fv, depth+1); err != nil {
			return err
		}
	}
	return e.EndStruct()
}

type marshalField struct {
	name  string
	index []int
}

// fieldCache caches resolved field lists per struct type.
var fieldCache sync.Map // reflect.Type -> []*marshalField

func marshalFields(t reflect.Type) []*marshalField {
	if cached, ok := fieldCache.Load(t); ok {
		return cached.([]*marshalField)
	}
	fields := structFields(t, nil)
	fieldCache.Store(t, fields)
	return fields
}

// structFields resolves the ZON fields of a struct: exported fields in
// declaration order, anonymous structs flattened, names from the zon tag or
// snake_case conversion.
func structFields(t reflect.Type, prefix []int) []*marshalField {
	var fields []*marshalField
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		name, _ := f.Tag.Lookup("zon")
		if name == "-" {
			continue
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if f.Anonymous && ft.Kind() == reflect.Struct && name == "" {
			fields = append(fields, structFields(ft, appendIndex(prefix, f.Index))...)
			continue
		}
		if name == "" {
			name = snakeCase(f.Name)
		}
		fields = append(fields, &marshalField{name: name, index: appendIndex(prefix, f.Index)})
	}
	return fields
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
