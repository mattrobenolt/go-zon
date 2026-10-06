package zon_test

import (
	"math"
	"math/big"
	"strings"
	"testing"

	"go.withmatt.com/zon"
)

func encode(t *testing.T, opts []zon.Option, fn func(*zon.Encoder) error) string {
	t.Helper()
	var sb strings.Builder
	e := zon.NewEncoder(&sb, opts...)
	if err := fn(e); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return sb.String()
}

func TestGolden(t *testing.T) {
	tests := []struct {
		name string
		opts []zon.Option
		fn   func(*zon.Encoder) error
		want string
	}{
		{
			name: "empty struct",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: `.{}`,
		},
		{
			name: "empty struct with end",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: `.{}`,
		},
		{
			name: "empty array",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginArray(); err != nil {
					return err
				}
				return e.EndArray()
			},
			want: `.{}`,
		},
		{
			name: "single field struct wraps by default",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.Name("x"); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: `.{
    .x = 1,
}`,
		},
		{
			name: "two fields inline with Fields",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(zon.Fields(2)); err != nil {
					return err
				}
				if err := e.Name("x"); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				if err := e.Name("y"); err != nil {
					return err
				}
				if err := e.Int(2); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: `.{ .x = 1, .y = 2 }`,
		},
		{
			name: "three fields wrap with Fields",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(zon.Fields(3)); err != nil {
					return err
				}
				for _, n := range []string{"x", "y", "z"} {
					if err := e.Name(n); err != nil {
						return err
					}
					if err := e.Int(1); err != nil {
						return err
					}
				}
				return e.EndStruct()
			},
			want: `.{
    .x = 1,
    .y = 1,
    .z = 1,
}`,
		},
		{
			name: "Inline stays on one line",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(zon.Inline()); err != nil {
					return err
				}
				for _, n := range []string{"x", "y", "z"} {
					if err := e.Name(n); err != nil {
						return err
					}
					if err := e.Int(1); err != nil {
						return err
					}
				}
				return e.EndStruct()
			},
			want: `.{ .x = 1, .y = 1, .z = 1 }`,
		},
		{
			name: "array of one elides spaces",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginArray(zon.Fields(1)); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				return e.EndArray()
			},
			want: `.{1}`,
		},
		{
			name: "array of two",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginArray(zon.Fields(2)); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				if err := e.Int(2); err != nil {
					return err
				}
				return e.EndArray()
			},
			want: `.{ 1, 2 }`,
		},
		{
			name: "array of three wraps",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginArray(zon.Fields(3)); err != nil {
					return err
				}
				for _, i := range []int64{1, 2, 3} {
					if err := e.Int(i); err != nil {
						return err
					}
				}
				return e.EndArray()
			},
			want: `.{
    1,
    2,
    3,
}`,
		},
		{
			name: "compact",
			opts: []zon.Option{zon.Compact()},
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.Name("x"); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				if err := e.Name("y"); err != nil {
					return err
				}
				if err := e.String("s"); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: `.{.x=1,.y="s"}`,
		},
		{
			name: "compact nested",
			opts: []zon.Option{zon.Compact()},
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.Name("a"); err != nil {
					return err
				}
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.Name("b"); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				if err := e.EndStruct(); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: `.{.a=.{.b=1}}`,
		},
		{
			name: "nested wrapped indentation",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.Name("inner"); err != nil {
					return err
				}
				if err := e.BeginStruct(zon.Fields(3)); err != nil {
					return err
				}
				for _, n := range []string{"a", "b", "c"} {
					if err := e.Name(n); err != nil {
						return err
					}
					if err := e.Int(1); err != nil {
						return err
					}
				}
				if err := e.EndStruct(); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: `.{
    .inner = .{
        .a = 1,
        .b = 1,
        .c = 1,
    },
}`,
		},
		{
			name: "custom indent",
			opts: []zon.Option{zon.WithIndent("\t")},
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.Name("x"); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				if err := e.Name("y"); err != nil {
					return err
				}
				if err := e.Int(2); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: ".{\n\t.x = 1,\n\t.y = 2,\n}",
		},
		{
			name: "root int",
			fn: func(e *zon.Encoder) error {
				return e.Int(-5)
			},
			want: `-5`,
		},
		{
			name: "root int64 min",
			fn: func(e *zon.Encoder) error {
				return e.Int(math.MinInt64)
			},
			want: `-9223372036854775808`,
		},
		{
			name: "root uint64 max",
			fn: func(e *zon.Encoder) error {
				return e.Uint(math.MaxUint64)
			},
			want: `18446744073709551615`,
		},
		{
			name: "root bigint",
			fn: func(e *zon.Encoder) error {
				return e.BigInt(new(big.Int).Lsh(big.NewInt(1), 130))
			},
			want: `1361129467683753853853498429727072845824`,
		},
		{
			name: "floats",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginArray(zon.Inline()); err != nil {
					return err
				}
				if err := e.Float(1.5); err != nil {
					return err
				}
				if err := e.Float(0.1); err != nil {
					return err
				}
				if err := e.Float(1e300); err != nil {
					return err
				}
				if err := e.Float(5); err != nil {
					return err
				}
				if err := e.Float(math.NaN()); err != nil {
					return err
				}
				if err := e.Float(math.Inf(1)); err != nil {
					return err
				}
				if err := e.Float(math.Inf(-1)); err != nil {
					return err
				}
				if err := e.Float(math.Copysign(0, -1)); err != nil {
					return err
				}
				return e.EndArray()
			},
			want: `.{ 1.5, 0.1, 1e+300, 5, nan, inf, -inf, -0.0 }`,
		},
		{
			name: "root scalars",
			fn: func(e *zon.Encoder) error {
				return e.Bool(true)
			},
			want: `true`,
		},
		{
			name: "root null",
			fn: func(e *zon.Encoder) error {
				return e.Null()
			},
			want: `null`,
		},
		{
			name: "root enum",
			fn: func(e *zon.Encoder) error {
				return e.Enum("debug")
			},
			want: `.debug`,
		},
		{
			name: "root keyword enum",
			fn: func(e *zon.Encoder) error {
				return e.Enum("while")
			},
			want: `.@"while"`,
		},
		{
			name: "keyword and primitive field names",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(zon.Fields(3)); err != nil {
					return err
				}
				if err := e.Name("while"); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				if err := e.Name("type"); err != nil {
					return err
				}
				if err := e.Int(2); err != nil {
					return err
				}
				if err := e.Name("4four"); err != nil {
					return err
				}
				if err := e.Int(3); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: `.{
    .@"while" = 1,
    .type = 2,
    .@"4four" = 3,
}`,
		},
		{
			name: "string escaping",
			fn: func(e *zon.Encoder) error {
				return e.String(" \x07 \x11 \" derp é '")
			},
			want: `" \x07 \x11 \" derp \xc3\xa9 '"`,
		},
		{
			name: "code points",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginArray(zon.Inline()); err != nil {
					return err
				}
				if err := e.CodePoint('a'); err != nil {
					return err
				}
				if err := e.CodePoint('\x07'); err != nil {
					return err
				}
				if err := e.CodePoint('"'); err != nil {
					return err
				}
				if err := e.CodePoint('⚡'); err != nil {
					return err
				}
				return e.EndArray()
			},
			want: `.{ 'a', '\x07', '"', '\u{26a1}' }`,
		},
		{
			name: "multiline field",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.Name("content"); err != nil {
					return err
				}
				if err := e.MultilineString("hello\nworld"); err != nil {
					return err
				}
				if err := e.Name("x"); err != nil {
					return err
				}
				if err := e.Int(1); err != nil {
					return err
				}
				return e.EndStruct()
			},
			// Note: the trailing space after "content = " and the lonely
			// comma line are std.zon.Serializer's exact behavior.
			want: ".{\n    .content = \n    \\\\hello\n    \\\\world\n    ,\n    .x = 1,\n}",
		},
		{
			name: "multiline at root",
			fn: func(e *zon.Encoder) error {
				return e.MultilineString("hello\nworld")
			},
			want: `\\hello
\\world`,
		},
		{
			name: "multiline with CRLF",
			fn: func(e *zon.Encoder) error {
				return e.MultilineString("a\r\nb")
			},
			want: `\\a
\\b`,
		},
		{
			name: "union imperative",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginUnion("io"); err != nil {
					return err
				}
				if err := e.String("EOF"); err != nil {
					return err
				}
				return e.EndUnion()
			},
			want: `.{ .io = "EOF" }`,
		},
		{
			name: "unions inside struct",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(); err != nil {
					return err
				}
				if err := e.Name("err"); err != nil {
					return err
				}
				if err := e.BeginUnion("io"); err != nil {
					return err
				}
				if err := e.String("EOF"); err != nil {
					return err
				}
				if err := e.EndUnion(); err != nil {
					return err
				}
				if err := e.Name("retry"); err != nil {
					return err
				}
				if err := e.Enum("backoff"); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: `.{
    .err = .{ .io = "EOF" },
    .retry = .backoff,
}`,
		},
		{
			name: "value types compose",
			fn: func(e *zon.Encoder) error {
				return e.WriteValue(zon.Struct{
					{Name: "name", Value: zon.String("example")},
					{Name: "mode", Value: zon.Enum("debug")},
					{Name: "err", Value: zon.Union{Tag: "io", Value: zon.String("EOF")}},
					{Name: "retry", Value: zon.Union{Tag: "backoff", Value: zon.Void{}}},
					{Name: "pos", Value: zon.Array{zon.Int(3), zon.Int(4)}},
				})
			},
			want: `.{
    .name = "example",
    .mode = .debug,
    .err = .{ .io = "EOF" },
    .retry = .backoff,
    .pos = .{ 3, 4 },
}`,
		},
		{
			name: "value types apply wrap rule",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginArray(zon.Fields(2)); err != nil {
					return err
				}
				if err := e.WriteValue(zon.Array{zon.Int(1)}); err != nil {
					return err
				}
				if err := e.WriteValue(zon.Struct{
					{Name: "x", Value: zon.Int(1)},
					{Name: "y", Value: zon.Int(2)},
				}); err != nil {
					return err
				}
				return e.EndArray()
			},
			want: `.{ .{1}, .{ .x = 1, .y = 2 } }`,
		},
		{
			name: "raw",
			fn: func(e *zon.Encoder) error {
				if err := e.BeginStruct(zon.Fields(1)); err != nil {
					return err
				}
				if err := e.Name("raw"); err != nil {
					return err
				}
				if err := e.Raw([]byte(`.{ .inner = true }`)); err != nil {
					return err
				}
				return e.EndStruct()
			},
			want: `.{ .raw = .{ .inner = true } }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encode(t, tt.opts, tt.fn); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
