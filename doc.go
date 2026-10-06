// Package zon encodes ZON, the Zig Object Notation, as implemented by Zig
// 0.16's standard library.
//
// A ZON document is exactly one value built from literals: structs
// ".{ .name = v }", arrays ".{ v1, v2 }", tagged unions ".{ .tag = payload }",
// enum literals ".name", and the scalars true, false, null, nan, inf, -inf,
// integers, floats, code points 'c', and strings. ZON is untyped: the
// consumer's type decides what a literal means, which is why this package
// encodes but does not decode.
//
// The encoder is streaming and imperative, in the spirit of the json/v2
// jsontext encoder and closely following std.zon.Serializer:
//
//	var sb strings.Builder
//	e := zon.NewEncoder(&sb)
//	e.BeginStruct()
//	e.Name("name")
//	e.String("example")
//	e.Name("version")
//	e.String("0.1.0")
//	e.Name("missing")
//	e.WriteValue(zon.Union{Tag: "tag", Value: zon.Enum("none")})
//	e.Name("options")
//	e.BeginArray(zon.Inline())
//	e.Bool(true)
//	e.Bool(false)
//	e.EndArray()
//	e.EndStruct()
//
// writes
//
//	.{
//	    .name = "example",
//	    .version = "0.1.0",
//	    .missing = .{ .tag = .none },
//	    .options = .{ true, false },
//	}
//
// Every write is validated against the current position and the completed
// root value is flushed to the writer once. ZON kinds that Go has no native
// equivalent for are first-class: [Encoder.Enum] for enum literals and
// void-payload unions, [Encoder.BeginUnion] and [Encoder.EndUnion] for
// tagged unions with payloads, [Encoder.CodePoint], and [Void] as a
// marker. The [Value] types compose the same kinds, and [Encoder.Raw]
// splices pre-encoded ZON.
package zon
