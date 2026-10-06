package zon_test

import (
	"fmt"
	"log"
	"strings"

	"go.withmatt.com/zon"
)

// Example builds a document with the streaming encoder. The ZON kinds Go has
// no native form for appear as values: an enum literal, a tagged union with a
// payload, and a union with a void payload written as a bare tag.
func Example() {
	var sb strings.Builder
	e := zon.NewEncoder(&sb)

	e.BeginStruct()
	e.Name("listen")
	e.String("127.0.0.1:9103")
	e.Name("mode")
	e.Enum("debug")
	e.Name("tls")
	e.BeginUnion("file")
	e.BeginStruct(zon.Fields(1))
	e.Name("cert")
	e.String("server.pem")
	e.EndStruct()
	e.EndUnion()
	e.Name("retry")
	e.Enum("backoff")
	e.Name("options")
	e.BeginArray(zon.Inline())
	e.Bool(true)
	e.Bool(false)
	e.EndArray()
	e.EndStruct()

	fmt.Print(sb.String())
	// Output:
	// .{
	//     .listen = "127.0.0.1:9103",
	//     .mode = .debug,
	//     .tls = .{ .file = .{ .cert = "server.pem" } },
	//     .retry = .backoff,
	//     .options = .{ true, false },
	// }
}

func ExampleMarshal() {
	type tls struct {
		Cert string `zon:"cert,omitempty"`
		Key  string `zon:"key,omitempty"`
	}
	type server struct {
		Listen string            `zon:"listen"`
		TLS    *tls              `zon:"tls,omitempty"`
		Extra  map[string]string `zon:"extra"`
	}

	b, err := zon.Marshal(server{
		Listen: "127.0.0.1:9103",
		TLS:    &tls{Cert: "server.pem"}, // key stays empty: omitted
		Extra:  map[string]string{"env": "prod"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(b))
	// Output:
	// .{
	//     .listen = "127.0.0.1:9103",
	//     .tls = .{ .cert = "server.pem" },
	//     .extra = .{ .env = "prod" },
	// }
}

// The two styles compose: stream the document by hand, marshal one field with
// reflection.
func ExampleEncoder_WriteAny() {
	var sb strings.Builder
	e := zon.NewEncoder(&sb)

	e.BeginStruct(zon.Fields(1))
	e.Name("deps")
	e.WriteAny(map[string]string{"web": "1.0.0", "api": "0.4.2"})
	e.EndStruct()

	fmt.Print(sb.String())
	// Output: .{ .deps = .{ .api = "0.4.2", .web = "1.0.0" } }
}
