package zon_test

import (
	"errors"
	"strings"
	"testing"

	"go.withmatt.com/zon"
)

func TestInvariants(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(*zon.Encoder) error
		op     string
		reason string
		offset int
	}{
		{
			name: "name outside of a struct",
			fn: func(e *zon.Encoder) error {
				return e.Name("x")
			},
			op:     "Name",
			reason: "field name outside of a struct",
			offset: 0,
		},
		{
			name: "value where a field name is expected",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				return e.Int(1)
			},
			op:     "Int",
			reason: "expected a field name, got a value",
			offset: 2,
		},
		{
			name: "second field name before a value",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.Name("x"); err != nil {
					return err
				}
				return e.Name("y")
			},
			op:     "Name",
			reason: "expected a value for field .x, got a field name",
			offset: 12,
		},
		{
			name: "end without a container",
			fn: func(e *zon.Encoder) error {
				return e.EndStruct()
			},
			op:     "EndStruct",
			reason: "no open container",
			offset: 0,
		},
		{
			name: "end kind mismatch",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginArray(); err != nil {
					return err
				}
				return e.EndStruct()
			},
			op:     "EndStruct",
			reason: "EndStruct does not match open array",
			offset: 2,
		},
		{
			name: "end union mismatch",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginUnion("t"); err != nil {
					return err
				}
				return e.EndArray()
			},
			op:     "EndArray",
			reason: "EndArray does not match open union",
			offset: 8,
		},
		{
			name: "end struct with a nameless value",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.Name("x"); err != nil {
					return err
				}
				return e.EndStruct()
			},
			op:     "EndStruct",
			reason: "field .x has no value",
			offset: 12,
		},
		{
			name: "end union without a payload",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginUnion("t"); err != nil {
					return err
				}
				return e.EndUnion()
			},
			op:     "EndUnion",
			reason: "union .t requires a payload before EndUnion; write the tag with Enum for a void payload",
			offset: 8,
		},
		{
			name: "name inside an array",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginArray(); err != nil {
					return err
				}
				return e.Name("x")
			},
			op:     "Name",
			reason: "field name inside an array",
			offset: 2,
		},
		{
			name: "name after a union payload",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginUnion("t"); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				return e.Name("x")
			},
			op:     "Name",
			reason: "field name inside a union",
			offset: 9,
		},
		{
			name: "value after a union payload",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginUnion("t"); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				return e.Int(2)
			},
			op:     "Int",
			reason: "union payload already written; expected EndUnion",
			offset: 9,
		},
		{
			name: "second root value",
			fn: func(e *zon.Encoder) error {
				if err := e.Int(1); err != nil {
					return err
				}
				return e.Int(2)
			},
			op:     "Int",
			reason: "document already complete",
			offset: 1,
		},
		{
			name: "write after the root closes",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.EndStruct(); err != nil {
					return err
				}
				return e.Int(1)
			},
			op:     "Int",
			reason: "document already complete",
			offset: 3,
		},
		{
			name: "void outside of a union",
			fn: func(e *zon.Encoder) error {
				return e.WriteValue(zon.Void{})
			},
			op:     "WriteValue",
			reason: "void is only valid as the payload of a union",
			offset: 0,
		},
		{
			name: "union without a payload",
			fn: func(e *zon.Encoder) error {
				return e.WriteValue(zon.Union{Tag: "x"})
			},
			op:     "WriteValue",
			reason: "union .x requires a payload; use zon.Void for a void payload",
			offset: 0,
		},
		{
			name: "nil value",
			fn: func(e *zon.Encoder) error {
				return e.WriteValue(nil)
			},
			op:     "WriteValue",
			reason: "nil value",
			offset: 0,
		},
		{
			name: "field without a value",
			fn: func(e *zon.Encoder) error {
				return e.WriteValue(zon.Struct{{Name: "x"}})
			},
			op:     "WriteValue",
			reason: "nil value",
			offset: 8,
		},
		{
			name: "nil big.Int",
			fn: func(e *zon.Encoder) error {
				return e.BigInt(nil)
			},
			op:     "BigInt",
			reason: "nil *big.Int",
			offset: 0,
		},
		{
			name: "surrogate code point",
			fn: func(e *zon.Encoder) error {
				return e.CodePoint(0xD800)
			},
			op:     "CodePoint",
			reason: "invalid code point",
			offset: 0,
		},
		{
			name: "out of range code point",
			fn: func(e *zon.Encoder) error {
				return e.CodePoint(0x110000)
			},
			op:     "CodePoint",
			reason: "invalid code point",
			offset: 0,
		},
		{
			name: "lone carriage return in multiline",
			fn: func(e *zon.Encoder) error {
				return e.MultilineString("a\rb")
			},
			op:     "MultilineString",
			reason: "carriage return not followed by a newline",
			offset: 0,
		},
		{
			name: "empty field name",
			fn: func(e *zon.Encoder) error {
				return e.Name("")
			},
			op:     "Name",
			reason: "name must not be empty",
			offset: 0,
		},
		{
			name: "empty enum name",
			fn: func(e *zon.Encoder) error {
				return e.Enum("")
			},
			op:     "Enum",
			reason: "name must not be empty",
			offset: 0,
		},
		{
			name: "empty union tag",
			fn: func(e *zon.Encoder) error {
				return e.BeginUnion("")
			},
			op:     "BeginUnion",
			reason: "name must not be empty",
			offset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			e := zon.NewEncoder(&sb)
			err := tt.fn(e)
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			var zerr *zon.Error
			if !errors.As(err, &zerr) {
				t.Fatalf("error %v is not a *zon.Error", err)
			}
			if zerr.Op != tt.op {
				t.Errorf("Op = %q, want %q", zerr.Op, tt.op)
			}
			if zerr.Reason != tt.reason {
				t.Errorf("Reason = %q, want %q", zerr.Reason, tt.reason)
			}
			if zerr.Offset != tt.offset {
				t.Errorf("Offset = %d, want %d", zerr.Offset, tt.offset)
			}
		})
	}
}

func TestFailedWritesLeaveStateIntact(t *testing.T) {
	var sb strings.Builder
	e := zon.NewEncoder(&sb)
	if err := e.BeginArray(); err != nil {
		t.Fatalf("BeginArray: %v", err)
	}
	if err := e.Name("x"); err == nil {
		t.Fatal("Name inside an array should fail")
	}
	// The failed write did not consume the array position.
	if err := e.Int(1); err != nil {
		t.Fatalf("Int: %v", err)
	}
	if err := e.EndArray(); err != nil {
		t.Fatalf("EndArray: %v", err)
	}
	want := ".{\n    1,\n}"
	if got := sb.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type recordingWriter struct {
	writes []string
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.writes = append(w.writes, string(p))
	return len(p), nil
}

func TestFlushesOncePerDocument(t *testing.T) {
	w := &recordingWriter{}
	e := zon.NewEncoder(w)
	if err := e.BeginStruct(); err != nil {
		t.Fatalf("BeginStruct: %v", err)
	}
	if got := len(w.writes); got != 0 {
		t.Fatalf("writer saw %d writes before the document completed, want 0", got)
	}
	if err := e.Name("x"); err != nil {
		t.Fatalf("Name: %v", err)
	}
	if err := e.Int(1); err != nil {
		t.Fatalf("Int: %v", err)
	}
	if err := e.EndStruct(); err != nil {
		t.Fatalf("EndStruct: %v", err)
	}
	if got := len(w.writes); got != 1 {
		t.Fatalf("writer saw %d writes, want 1", got)
	}
	if got, want := w.writes[0], ".{\n    .x = 1,\n}"; got != want {
		t.Errorf("write = %q, want %q", got, want)
	}
}

var errWriterBoom = errors.New("boom")

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) { return 0, errWriterBoom }

func TestWriterErrorPropagates(t *testing.T) {
	e := zon.NewEncoder(failingWriter{})
	err := e.Int(1)
	if !errors.Is(err, errWriterBoom) {
		t.Fatalf("error %v does not wrap the writer error", err)
	}
	if !strings.Contains(err.Error(), "zon: write: ") {
		t.Errorf("error message %q does not identify the write", err.Error())
	}
	// The document is complete regardless; further writes fail.
	if err := e.Int(1); err == nil {
		t.Fatal("write after completion should fail")
	}
}
