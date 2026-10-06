# zon

[![Go Reference](https://pkg.go.dev/badge/go.withmatt.com/zon.svg)](https://pkg.go.dev/go.withmatt.com/zon)

Go encoder for [ZON](https://ziglang.org/documentation/0.16.0/std/#std.zon), the Zig
Object Notation, matching the Zig 0.16 standard library.

## Installation

```
go get go.withmatt.com/zon
```

## Usage

```go
e := zon.NewEncoder(w) // standard Zig whitespace: 4-space indent

e.BeginStruct()
e.Name("name")
e.String("example")
e.Name("missing")
e.WriteValue(zon.Union{Tag: "tag", Value: zon.Enum("none")})
e.Name("options")
e.BeginArray(zon.Inline())
e.Bool(true)
e.Bool(false)
e.EndArray()
e.EndStruct()
```

writes

```zig
.{
    .name = "example",
    .missing = .{ .tag = .none },
    .options = .{ true, false },
}
```

A ZON document is exactly one value. Every write is validated against the
current position; invalid writes return a `*zon.Error` with the byte offset
where they occurred and leave the state untouched. The completed document is
flushed to the writer once.

## Design

Modeled on `std.zon.Serializer` from the Zig standard library and the
json/v2 `jsontext` encoder:

- Streaming, imperative `Encoder` over any `io.Writer`. No reflection.
- ZON kinds Go has no native form for are first-class: `Enum` for enum
  literals (also the form of a void-payload union), `BeginUnion`/`EndUnion`
  for tagged unions with payloads, `CodePoint` for `'c'` literals,
  `MultilineString` for the `\\` form, and `BigInt` for the unbounded
  integer literals ZON allows.
- `Value` types (`zon.Union`, `zon.Enum`, `zon.Struct`, ...) compose the same
  kinds when a value needs to travel through Go code first.
- Field names are escaped as Zig identifiers (`while` becomes
  `.@"while"`); strings escape non-printable and non-ASCII bytes as `\xNN`.

Formatting follows `std.zon.Serializer` exactly, including the trailing
comma before a wrapped closing brace. Containers opened by `Begin` methods
wrap one field per line; `Inline()` keeps one line and `Fields(n)` applies
the standard rule (wrap when more than two fields). `Compact()` writes the
minimal form.

## Reflection

`Marshal`, `MarshalWrite`, and `Encoder.WriteAny` walk Go values through the
same primitives, so the two styles compose: stream the skeleton with the
encoder, marshal one heavy field with reflection.

- Struct fields: the first comma-separated component of the tag names
  (`zon:"metrics_listen,omitempty"`), `zon:"-"` skips, otherwise the Go name
  converts to snake_case (`HTTPServer` → `http_server`); anonymous struct
  fields flatten
- The `omitempty` option skips fields holding the zero value of their type,
  so an absent field means the consumer's default — `std.zon.parse` fills
  missing fields from the Zig struct's `= default` declarations
- Nil pointers and interfaces write `null`; maps write struct fields with
  the keys sorted; `[]byte` writes a string literal
- `zon.Enum`, `zon.Union`, `zon.CodePoint`, `zon.Multiline`, `zon.Raw`, and
  `math/big.Int` fields marshal as their ZON kinds
- `chan`, `func`, `complex`, recursive types, and `math/big.Float` return an
  error

## Why no decoder

ZON is untyped: the consumer's type decides what a literal means. `std.zon.parse`
is type-directed — `.foo` resolves as an enum member, a void-payload union tag,
or a struct field only because the target Zig type says so. A Go decoder could
mirror the mapping this package uses for marshaling, but the consumer of ZON
is Zig by construction: tooling close enough to read ZON is close enough to be
written in Zig, where `std.zon.parse` is the reference parser. Emitting ZON has
users in any language; reading it back has users in Zig only. `Raw` splices
pre-encoded ZON in the meantime.

## Verification

`just verify-zig` round-trips encoder output through the Zig 0.16 standard
library itself: `std.zon.parse.fromSliceAlloc` with value assertions, plus
comptime `@import` for the formatting variants. It skips when no zig 0.16 is
on PATH. One known std limitation surfaced there: `std.zon.parse` cannot
compile a `u256`-typed field (its f128 range check cannot represent
`maxInt(u256)`), so wide integer literals are verified through comptime
`@import` instead.

## License

MIT

## Development

```
just test     # gotestsum
just lint     # golangci-lint
just fmt      # gofumpt + golines
```
