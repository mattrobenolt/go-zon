package zon_test

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"go.withmatt.com/zon"
)

func marshal(t *testing.T, v any, opts ...zon.Option) string {
	t.Helper()
	var sb strings.Builder
	if err := zon.MarshalWrite(&sb, v, opts...); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return sb.String()
}

type Embedded struct {
	Deep string `zon:"deep"`
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name string
		v    any
		opts []zon.Option
		want string
	}{
		{
			name: "nil",
			v:    nil,
			want: `null`,
		},
		{
			name: "tagged fields",
			v: struct {
				Name    string `zon:"name"`
				Version string `zon:"version"`
			}{Name: "example", Version: "0.1.0"},
			want: `.{ .name = "example", .version = "0.1.0" }`,
		},
		{
			name: "snake case names",
			v: struct {
				HTTPServer string
				URLPath    string
				ID         string
				X509       string
				A          string
				UserID     string
			}{HTTPServer: "s", URLPath: "p", ID: "i", X509: "x", A: "a", UserID: "u"},
			want: ".{\n    .http_server = \"s\",\n    .url_path = \"p\",\n    .id = \"i\",\n" +
				"    .x509 = \"x\",\n    .a = \"a\",\n    .user_id = \"u\",\n}",
		},
		{
			name: "keyword, skipped, and unexported fields",
			v: struct {
				While  int `zon:"while"`
				Public int
				hidden string
				Skip   int `zon:"-"`
			}{While: 1, Public: 2, hidden: "x", Skip: 3},
			want: `.{ .@"while" = 1, .public = 2 }`,
		},
		{
			name: "pointers and interfaces",
			v: struct {
				Opt   *int32 `zon:"opt"`
				Empty *int32 `zon:"empty"`
				Any   any    `zon:"any"`
				Nil   any    `zon:"nil"`
			}{Opt: new(int32), Empty: nil, Any: "hi", Nil: nil},
			want: ".{\n    .opt = 0,\n    .empty = null,\n    .any = \"hi\",\n    .nil = null,\n}",
		},
		{
			name: "slices arrays and bytes",
			v: struct {
				List []int32  `zon:"list"`
				Nil  []int32  `zon:"nil"`
				Fix  [3]int32 `zon:"fix"`
				Data []byte   `zon:"data"`
				NilB []byte   `zon:"nilb"`
				FixB [3]byte  `zon:"fixb"`
			}{List: []int32{1, 2}, Nil: nil, Fix: [3]int32{1, 2, 3}, Data: []byte("hi"), NilB: nil, FixB: [3]byte{1, 2, 3}},
			want: ".{\n    .list = .{ 1, 2 },\n    .nil = .{},\n    .fix = .{\n        1,\n" +
				"        2,\n        3,\n    },\n    .data = \"hi\",\n    .nilb = \"\",\n" +
				"    .fixb = .{\n        1,\n        2,\n        3,\n    },\n}",
		},
		{
			name: "map keys sorted",
			v: map[string]int{
				"zeta":  1,
				"alpha": 2,
				"mid":   3,
			},
			want: ".{\n    .alpha = 2,\n    .mid = 3,\n    .zeta = 1,\n}",
		},
		{
			name: "map inline",
			v: map[string]int{
				"beta":  1,
				"alpha": 2,
			},
			want: `.{ .alpha = 2, .beta = 1 }`,
		},
		{
			name: "value types",
			v: struct {
				Mode     zon.Enum      `zon:"mode"`
				Retry    zon.Union     `zon:"retry"`
				Err      zon.Union     `zon:"err"`
				CP       zon.CodePoint `zon:"cp"`
				Big      zon.BigInt    `zon:"big"`
				Body     zon.Multiline `zon:"body"`
				Fragment zon.Raw       `zon:"fragment"`
				Nothing  zon.Null      `zon:"nothing"`
			}{
				Mode:     zon.Enum("debug"),
				Retry:    zon.Union{Tag: "backoff", Value: zon.Void{}},
				Err:      zon.Union{Tag: "io", Value: zon.String("EOF")},
				CP:       '⚡',
				Body:     "line",
				Fragment: zon.Raw(".{ .raw = true }"),
			},
			want: ".{\n    .mode = .debug,\n    .retry = .backoff,\n    .err = .{ .io = \"EOF\" },\n" +
				"    .cp = '\\u{26a1}',\n    .big = 0,\n    .body = \n    \\\\line\n    ,\n" +
				"    .fragment = .{ .raw = true },\n    .nothing = null,\n}",
		},
		{
			name: "math big",
			v: struct {
				Big   *big.Int `zon:"big"`
				Empty *big.Int `zon:"empty"`
			}{Big: new(big.Int).Lsh(big.NewInt(1), 130)},
			want: ".{ .big = 1361129467683753853853498429727072845824, .empty = null }",
		},
		{
			name: "embedded flattening",
			v: struct {
				Embedded
				Named Embedded `zon:"named"`
			}{},
			want: `.{ .deep = "", .named = .{ .deep = "" } }`,
		},
		{
			name: "embedded pointer flattening",
			v: struct {
				*Embedded
				Top string `zon:"top"`
			}{Top: "t"},
			want: `.{ .top = "t" }`,
		},
		{
			name: "compact",
			v: struct {
				X int32 `zon:"x"`
				Y int32 `zon:"y"`
			}{X: 1, Y: 2},
			opts: []zon.Option{zon.Compact()},
			want: `.{.x=1,.y=2}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := marshal(t, tt.v, tt.opts...); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestMarshalBytes(t *testing.T) {
	got, err := zon.Marshal(map[string]int{"one": 1})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if want := ".{ .one = 1 }"; string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWriteAnyInEncoder(t *testing.T) {
	// Reflection composes with the streaming API at any value position.
	var sb strings.Builder
	e := zon.NewEncoder(&sb)
	if err := e.BeginStruct(zon.Fields(1)); err != nil {
		t.Fatalf("BeginStruct: %v", err)
	}
	if err := e.Name("deps"); err != nil {
		t.Fatalf("Name: %v", err)
	}
	if err := e.WriteAny(map[string]int{"a": 1, "b": 2}); err != nil {
		t.Fatalf("WriteAny: %v", err)
	}
	if err := e.EndStruct(); err != nil {
		t.Fatalf("EndStruct: %v", err)
	}
	want := ".{ .deps = .{ .a = 1, .b = 2 } }"
	if got := sb.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMarshalErrors(t *testing.T) {
	tests := []struct {
		name   string
		v      any
		reason string
	}{
		{
			name:   "chan",
			v:      struct{ C chan int }{},
			reason: "unsupported type chan int",
		},
		{
			name:   "func",
			v:      struct{ F func() }{},
			reason: "unsupported type func()",
		},
		{
			name:   "complex",
			v:      struct{ Z complex128 }{},
			reason: "unsupported type complex128",
		},
		{
			name:   "big float",
			v:      struct{ F big.Float }{},
			reason: "unsupported type math/big.Float; convert to float64 or use zon.Float",
		},
		{
			name:   "map key not string",
			v:      map[int]int{1: 1},
			reason: "map key type must be string, got int",
		},
		{
			name:   "recursive type",
			v:      cyclicValue(),
			reason: "marshal depth exceeded 1024 levels; recursive types are not supported",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := zon.Marshal(tt.v)
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			var zerr *zon.Error
			if !errors.As(err, &zerr) {
				t.Fatalf("error %v is not a *zon.Error", err)
			}
			if zerr.Reason != tt.reason {
				t.Errorf("Reason = %q, want %q", zerr.Reason, tt.reason)
			}
		})
	}
}

type node struct {
	Next *node
}

func cyclicValue() any {
	n := node{}
	n.Next = &n
	return &n
}
