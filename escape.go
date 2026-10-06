package zon

import "strconv"

const hexDigits = "0123456789abcdef"

func isPrintableASCII(c byte) bool {
	return 0x20 <= c && c <= 0x7e
}

// appendStringEscaped appends the contents of a double-quoted Zig string
// literal. Printable ASCII is written raw; every other byte, including
// non-ASCII UTF-8, is escaped as \xNN. Mirrors std.zig.stringEscape.
func appendStringEscaped(dst []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '"':
			dst = append(dst, '\\', '"')
		default:
			if isPrintableASCII(c) {
				dst = append(dst, c)
			} else {
				dst = append(dst, '\\', 'x', hexDigits[c>>4], hexDigits[c&0xf])
			}
		}
	}
	return dst
}

// isValidCodePoint reports whether r is a valid Unicode code point:
// within range and not a surrogate.
func isValidCodePoint(r rune) bool {
	return 0 <= r && r <= 0x10FFFF && (r < 0xD800 || r > 0xDFFF)
}

// appendCharEscaped appends the contents of a single-quoted Zig character
// literal. Bytes at most 0xFF are escaped as \xNN; larger code points as
// \u{...} with unpadded lowercase hex. Mirrors std.zig.charEscape.
func appendCharEscaped(dst []byte, r rune) []byte {
	switch r {
	case '\n':
		return append(dst, '\\', 'n')
	case '\r':
		return append(dst, '\\', 'r')
	case '\t':
		return append(dst, '\\', 't')
	case '\\':
		return append(dst, '\\', '\\')
	case '\'':
		return append(dst, '\\', '\'')
	}
	if 0x20 <= r && r <= 0x7e {
		return append(dst, byte(r))
	}
	if r <= 0xff {
		return append(dst, '\\', 'x', hexDigits[r>>4], hexDigits[r&0xf])
	}
	dst = append(dst, '\\', 'u', '{')
	dst = strconv.AppendInt(dst, int64(r), 16)
	return append(dst, '}')
}
