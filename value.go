package zon

import (
	"fmt"
	"math/big"
)

// Value is a ZON value that [Encoder.WriteValue] can write.
//
// The set of implementations is closed. Besides the primitives, the types
// exist for the ZON kinds Go has no native equivalent for: enums, tagged
// unions, and void. Values compose, so a [Union] payload is itself a Value.
type Value interface {
	isValue()
}

type (
	// Bool is a true or false literal.
	Bool bool
	// Int is a signed integer literal.
	Int int64
	// Uint is an unsigned integer literal.
	Uint uint64
	// Float is a floating-point literal.
	Float float64
	// String is a double-quoted string literal.
	String string
	// Multiline is a multiline string literal, the \\ form.
	Multiline string
	// CodePoint is a Unicode code point literal, e.g. 'a'.
	CodePoint rune
	// Null is the literal null, the ZON form of an absent optional.
	Null struct{}
	// Enum is an enum literal, ".name". It is also the form of a tagged
	// union with a void payload.
	Enum string
	// Array is an array literal, ".{ v1, v2 }".
	Array []Value
	// Struct is a struct literal, ".{ .name = v }".
	Struct []Field
	// Raw is pre-encoded ZON, written verbatim.
	Raw []byte
)

// Void is the payload of a tagged union whose variant has no value. It is
// only valid as [Union].Value, where it makes the union serialize as a bare
// enum literal; written anywhere else it is an error.
type Void struct{}

// Union is a tagged union. With a Void payload it writes as the enum literal
// ".tag"; with any other payload it writes as ".{ .tag = payload }".
type Union struct {
	Tag   string
	Value Value
}

// Field is one named field of a [Struct].
type Field struct {
	Name  string
	Value Value
}

// BigInt is an arbitrary-precision integer literal. It embeds
// [math/big.Int], inheriting its methods.
type BigInt struct {
	big.Int
}

func (Bool) isValue()      {}
func (Int) isValue()       {}
func (Uint) isValue()      {}
func (Float) isValue()     {}
func (String) isValue()    {}
func (Multiline) isValue() {}
func (CodePoint) isValue() {}
func (Null) isValue()      {}
func (Enum) isValue()      {}
func (Void) isValue()      {}
func (Union) isValue()     {}
func (Array) isValue()     {}
func (Struct) isValue()    {}
func (BigInt) isValue()    {}
func (Raw) isValue()       {}

// WriteValue writes v at the current position.
//
// An [Array] or [Struct] uses the standard wrap rule: more than two fields
// wrap onto one line each. Streaming with the Begin methods defaults to
// wrapping; use [Fields] or [Inline] to control it there.
func (e *Encoder) WriteValue(v Value) error {
	switch v := v.(type) {
	case nil:
		return e.errorAt("WriteValue", "nil value")
	case Bool:
		return e.Bool(bool(v))
	case Int:
		return e.Int(int64(v))
	case Uint:
		return e.Uint(uint64(v))
	case Float:
		return e.Float(float64(v))
	case String:
		return e.String(string(v))
	case Multiline:
		return e.MultilineString(string(v))
	case CodePoint:
		return e.CodePoint(rune(v))
	case Null:
		return e.Null()
	case Enum:
		return e.Enum(string(v))
	case Void:
		return e.errorAt("WriteValue", "void is only valid as the payload of a union")
	case Union:
		return e.writeUnion(v)
	case Array:
		return e.writeArray(v)
	case Struct:
		return e.writeStruct(v)
	case BigInt:
		return e.BigInt(&v.Int)
	case Raw:
		return e.Raw(v)
	default:
		return e.errorAt("WriteValue", fmt.Sprintf("unrecognized value type %T", v))
	}
}

func (e *Encoder) writeUnion(u Union) error {
	if u.Value == nil {
		return e.errorAt(
			"WriteValue",
			"union ."+u.Tag+" requires a payload; use zon.Void for a void payload",
		)
	}
	if _, isVoid := u.Value.(Void); isVoid {
		return e.Enum(u.Tag)
	}
	if err := e.BeginUnion(u.Tag); err != nil {
		return err
	}
	if err := e.WriteValue(u.Value); err != nil {
		return err
	}
	return e.EndUnion()
}

func (e *Encoder) writeArray(a Array) error {
	if err := e.BeginArray(Fields(len(a))); err != nil {
		return err
	}
	for _, elem := range a {
		if err := e.WriteValue(elem); err != nil {
			return err
		}
	}
	return e.EndArray()
}

func (e *Encoder) writeStruct(s Struct) error {
	if err := e.BeginStruct(Fields(len(s))); err != nil {
		return err
	}
	for _, f := range s {
		if err := e.Name(f.Name); err != nil {
			return err
		}
		if err := e.WriteValue(f.Value); err != nil {
			return err
		}
	}
	return e.EndStruct()
}
