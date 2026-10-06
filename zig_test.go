package zon_test

import (
	"bytes"
	"io"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.withmatt.com/zon"
)

// check.zig, in the repository root, holds the Zig-side assertions and is
// copied next to the generated .zon files: @embedFile resolves relative to
// the importing file's directory, so it cannot stay in the repository root.

type marshalDep struct {
	URL  string `zon:"url"`
	Hash string `zon:"hash"`
}

type MarshalEmbedded struct {
	Inner string `zon:"embedded_inner"`
}

// marshalDoc mirrors the Marshal struct in check.zig field for field.
type marshalDoc struct {
	Name    string                 `zon:"name"`
	Version string                 `zon:"version"`
	Mode    zon.Enum               `zon:"mode"`
	While   int                    `zon:"while"`
	UserID  int                    // no tag: snake_case conversion
	Opt     *uint32                `zon:"opt"`
	OptNull *uint32                `zon:"optnull"`
	Count   int64                  `zon:"count"`
	Ratio   float64                `zon:"ratio"`
	Flag    bool                   `zon:"flag"`
	Big     *big.Int               `zon:"big"`
	Tags    []string               `zon:"tags"`
	Matrix  [2][2]int32            `zon:"matrix"`
	Deps    map[string]*marshalDep `zon:"deps"`
	Retry   zon.Union              `zon:"retry"`
	Err     zon.Union              `zon:"err"`
	Content zon.Multiline          `zon:"content"`
	CP      zon.CodePoint          `zon:"cp"`
	Skip    string                 `zon:"-"`
	hidden  string
	MarshalEmbedded
}

// TestZigParse round-trips encoder output through the Zig 0.16 standard
// library: it generates ZON with this package, parses it with
// std.zon.parse.fromSliceAlloc and comptime @import, and asserts the values.
// It skips when no zig 0.16 is on PATH; run it with `just verify-zig` inside
// the dev shell.
func TestZigParse(t *testing.T) {
	zig, err := exec.LookPath("zig")
	if err != nil {
		t.Skip("zig not found on PATH")
	}
	out, err := exec.CommandContext(t.Context(), zig, "version").Output()
	if err != nil {
		t.Skipf("zig version: %v", err)
	}
	if v := strings.TrimSpace(string(out)); !strings.HasPrefix(v, "0.16.") {
		t.Skipf("zig %s is not 0.16.x, this library targets Zig 0.16 ZON", v)
	}

	dir := t.TempDir()

	// The kitchen sink exercises every literal through the runtime parser,
	// including nan and inf, which comptime imports cannot type.
	sink := func(e *zon.Encoder) error {
		if err := e.BeginStruct(); err != nil {
			return err
		}
		if err := e.Name("nan"); err != nil {
			return err
		}
		if err := e.Float(math.NaN()); err != nil {
			return err
		}
		if err := e.Name("inf"); err != nil {
			return err
		}
		if err := e.Float(math.Inf(1)); err != nil {
			return err
		}
		if err := e.Name("ninf"); err != nil {
			return err
		}
		if err := e.Float(math.Inf(-1)); err != nil {
			return err
		}
		if err := e.Name("negzero"); err != nil {
			return err
		}
		if err := e.Float(math.Copysign(0, -1)); err != nil {
			return err
		}
		if err := e.Name("exp"); err != nil {
			return err
		}
		if err := e.Float(1e300); err != nil {
			return err
		}
		if err := e.Name("whole"); err != nil {
			return err
		}
		if err := e.Float(5); err != nil {
			return err
		}
		if err := e.Name("tiny"); err != nil {
			return err
		}
		if err := e.Float(0.1); err != nil {
			return err
		}
		if err := e.Name("min"); err != nil {
			return err
		}
		if err := e.Int(math.MinInt64); err != nil {
			return err
		}
		if err := e.Name("max"); err != nil {
			return err
		}
		if err := e.Int(math.MaxInt64); err != nil {
			return err
		}
		if err := e.Name("umax"); err != nil {
			return err
		}
		if err := e.Uint(math.MaxUint64); err != nil {
			return err
		}
		if err := e.Name("b"); err != nil {
			return err
		}
		if err := e.Bool(true); err != nil {
			return err
		}
		if err := e.Name("opt"); err != nil {
			return err
		}
		if err := e.Uint(7); err != nil {
			return err
		}
		if err := e.Name("optnull"); err != nil {
			return err
		}
		if err := e.Null(); err != nil {
			return err
		}
		if err := e.Name("s"); err != nil {
			return err
		}
		if err := e.String("é \\ hi \x07 \" derp '"); err != nil {
			return err
		}
		if err := e.Name("ml"); err != nil {
			return err
		}
		if err := e.MultilineString("a\r\nb"); err != nil {
			return err
		}
		if err := e.Name("cp"); err != nil {
			return err
		}
		if err := e.CodePoint('⚡'); err != nil {
			return err
		}
		if err := e.Name("arr1"); err != nil {
			return err
		}
		if err := e.BeginArray(zon.Fields(1)); err != nil {
			return err
		}
		if err := e.Int(1); err != nil {
			return err
		}
		if err := e.EndArray(); err != nil {
			return err
		}
		if err := e.Name("arr"); err != nil {
			return err
		}
		if err := e.BeginArray(zon.Fields(3)); err != nil {
			return err
		}
		for _, i := range []int64{1, 2, 3} {
			if err := e.Int(i); err != nil {
				return err
			}
		}
		if err := e.EndArray(); err != nil {
			return err
		}
		if err := e.Name("mode"); err != nil {
			return err
		}
		if err := e.Enum("debug"); err != nil {
			return err
		}
		if err := e.Name("kw"); err != nil {
			return err
		}
		if err := e.Enum("while"); err != nil {
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
		if err := e.Name("type"); err != nil {
			return err
		}
		if err := e.BeginStruct(zon.Fields(1)); err != nil {
			return err
		}
		if err := e.Name("x"); err != nil {
			return err
		}
		if err := e.Int(1); err != nil {
			return err
		}
		if err := e.EndStruct(); err != nil {
			return err
		}
		return e.EndStruct()
	}

	// marshalDoc mirrors the Marshal struct in check.zig field for field.
	opt := uint32(7)
	doc := marshalDoc{
		Name:    "example",
		Version: "0.1.0",
		Mode:    zon.Enum("debug"),
		While:   5,
		UserID:  42, // no tag: snake_case conversion, "user_id"
		Opt:     &opt,
		Count:   1000000007,
		Ratio:   0.1,
		Flag:    true,
		Big:     new(big.Int).Lsh(big.NewInt(1), 100),
		Tags:    []string{"a", "b", "c"},
		Matrix:  [2][2]int32{{1, 2}, {3, 4}},
		Deps: map[string]*marshalDep{
			"beta":  {URL: "https://u2", Hash: "h2"},
			"alpha": {URL: "https://u1", Hash: "h1"},
		},
		Retry:           zon.Union{Tag: "backoff", Value: zon.Void{}},
		Err:             zon.Union{Tag: "io", Value: zon.String("EOF")},
		Content:         zon.Multiline("first\nsecond"),
		CP:              zon.CodePoint('⚡'),
		hidden:          "x",
		MarshalEmbedded: MarshalEmbedded{Inner: "deep"},
	}
	docs := map[string]struct {
		opts []zon.Option
		fn   func(*zon.Encoder) error
	}{
		"sink.zon": {fn: sink},
		"marshal.zon": {fn: func(e *zon.Encoder) error {
			return e.WriteAny(doc)
		}},
		"big.zon": {fn: func(e *zon.Encoder) error {
			if err := e.BeginStruct(); err != nil {
				return err
			}
			if err := e.Name("big"); err != nil {
				return err
			}
			if err := e.BigInt(new(big.Int).Lsh(big.NewInt(1), 255)); err != nil {
				return err
			}
			return e.EndStruct()
		}},
		"fmt_wrapped.zon": {fn: func(e *zon.Encoder) error {
			if err := e.BeginStruct(); err != nil {
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
		}},
		"fmt_compact.zon": {opts: []zon.Option{zon.Compact()}, fn: func(e *zon.Encoder) error {
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
		}},
		"fmt_deep.zon": {fn: func(e *zon.Encoder) error {
			if err := e.BeginStruct(); err != nil {
				return err
			}
			if err := e.Name("one"); err != nil {
				return err
			}
			if err := e.BeginStruct(zon.Fields(3)); err != nil {
				return err
			}
			for _, n := range []string{"a", "b", "c"} {
				if err := e.Name(n); err != nil {
					return err
				}
				if err := e.String("v"); err != nil {
					return err
				}
			}
			if err := e.EndStruct(); err != nil {
				return err
			}
			return e.EndStruct()
		}},
		"ml_root.zon": {fn: func(e *zon.Encoder) error {
			return e.MultilineString("one\ntwo")
		}},
	}

	for name, doc := range docs {
		var b bytes.Buffer
		e := zon.NewEncoder(&b, doc.opts...)
		if err := doc.fn(e); err != nil {
			t.Fatalf("generate %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b.Bytes(), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	src, err := os.Open("check.zig")
	if err != nil {
		t.Fatalf("open check.zig: %v", err)
	}
	defer src.Close()
	dst, err := os.Create(filepath.Join(dir, "check.zig"))
	if err != nil {
		t.Fatalf("create check.zig: %v", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("copy check.zig: %v", err)
	}
	if err := dst.Close(); err != nil {
		t.Fatalf("close check.zig: %v", err)
	}

	cmd := exec.CommandContext(t.Context(), zig, "test", "check.zig")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("zig test failed, encoder output does not parse:\n%s", out)
	}
}
