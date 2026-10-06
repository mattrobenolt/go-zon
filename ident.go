package zon

// keywords is the set of Zig keywords, ported from std.zig.Token.keywords
// (Zig 0.16). A name matching one of these must be written with @"..."
// quoting, since a bare keyword is not a valid identifier.
var keywords = map[string]struct{}{
	"addrspace":   {},
	"align":       {},
	"allowzero":   {},
	"and":         {},
	"anyframe":    {},
	"anytype":     {},
	"asm":         {},
	"break":       {},
	"callconv":    {},
	"catch":       {},
	"comptime":    {},
	"const":       {},
	"continue":    {},
	"defer":       {},
	"else":        {},
	"enum":        {},
	"errdefer":    {},
	"error":       {},
	"export":      {},
	"extern":      {},
	"fn":          {},
	"for":         {},
	"if":          {},
	"inline":      {},
	"linksection": {},
	"noalias":     {},
	"noinline":    {},
	"nosuspend":   {},
	"opaque":      {},
	"or":          {},
	"orelse":      {},
	"packed":      {},
	"pub":         {},
	"resume":      {},
	"return":      {},
	"struct":      {},
	"suspend":     {},
	"switch":      {},
	"test":        {},
	"threadlocal": {},
	"try":         {},
	"union":       {},
	"unreachable": {},
	"var":         {},
	"volatile":    {},
	"while":       {},
}

func isKeyword(name string) bool {
	_, ok := keywords[name]
	return ok
}

// isValidIdent reports whether name can be written bare, i.e. it matches
// [A-Za-z_][A-Za-z0-9_]* and is not a Zig keyword. Primitive type names such
// as "type" and "u32", and a lone "_", stay bare, which mirrors
// std.zig.fmtIdPU as used by the ZON serializer.
func isValidIdent(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		case '0' <= c && c <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return !isKeyword(name)
}

// appendIdent appends name as a Zig identifier, quoting it as @"..." when it
// cannot be written bare. Quotes and non-printable bytes inside the quoted
// form are escaped.
func appendIdent(dst []byte, name string) []byte {
	if isValidIdent(name) {
		return append(dst, name...)
	}
	dst = append(dst, '@', '"')
	dst = appendStringEscaped(dst, name)
	return append(dst, '"')
}
