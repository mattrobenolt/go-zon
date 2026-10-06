package zon

import "testing"

func TestIsValidIdent(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"hello", true},
		{"_underscore", true},
		{"_", true},
		{"i386", true},
		{"type", true},
		{"u32", true},
		{"comptime_float", true},
		{"", false},
		{"3d", false},
		{"a b c", false},
		{"while", false},
		{"test", false},
		{"struct", false},
	}
	for _, tt := range tests {
		if got := isValidIdent(tt.name); got != tt.want {
			t.Errorf("isValidIdent(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestAppendIdent(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"hello", `hello`},
		{"_", `_`},
		{"__", `__`},
		{"i386", `i386`},
		{"type", `type`},
		{"u32", `u32`},
		{"while", `@"while"`},
		{"test", `@"test"`},
		{"enum", `@"enum"`},
		{"4four", `@"4four"`},
		{"", `@""`},
		{"a b c", `@"a b c"`},
		{"11\"23", `@"11\"23"`},
		{"11\x0f23", `@"11\x0f23"`},
	}
	for _, tt := range tests {
		if got := string(appendIdent(nil, tt.name)); got != tt.want {
			t.Errorf("appendIdent(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestAppendStringEscaped(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"plain", "plain"},
		{" \x07 \x11 \" derp '", ` \x07 \x11 \" derp '`},
		{"\n\r\t", `\n\r\t`},
		{"back\\slash", `back\\slash`},
		{"\x00", `\x00`},
		{"\x7f", `\x7f`},
		{"é", `\xc3\xa9`}, // UTF-8 escapes byte by byte, not as a code point
		{"⚡", `\xe2\x9a\xa1`},
		{"~", "~"},
		{"\x1b", `\x1b`},
	}
	for _, tt := range tests {
		if got := string(appendStringEscaped(nil, tt.in)); got != tt.want {
			t.Errorf("appendStringEscaped(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestAppendCharEscaped(t *testing.T) {
	tests := []struct {
		in   rune
		want string
	}{
		{'c', "c"},
		{'~', "~"},
		{'"', `"`},
		{'\n', `\n`},
		{'\r', `\r`},
		{'\t', `\t`},
		{'\\', `\\`},
		{'\'', `\'`},
		{'\x07', `\x07`},
		{'\xff', `\xff`},
		{'⚡', `\u{26a1}`},
		{0x10FFFF, `\u{10ffff}`},
	}
	for _, tt := range tests {
		if got := string(appendCharEscaped(nil, tt.in)); got != tt.want {
			t.Errorf("appendCharEscaped(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsValidCodePoint(t *testing.T) {
	valid := []rune{0, 'a', '~', 0x26a1, 0x10FFFF}
	invalid := []rune{-1, 0xD800, 0xDFFF, 0x110000}
	for _, r := range valid {
		if !isValidCodePoint(r) {
			t.Errorf("isValidCodePoint(%d) = false, want true", r)
		}
	}
	for _, r := range invalid {
		if isValidCodePoint(r) {
			t.Errorf("isValidCodePoint(%d) = true, want false", r)
		}
	}
}
