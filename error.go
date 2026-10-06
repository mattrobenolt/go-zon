package zon

import "fmt"

// Error reports an invalid operation on an [Encoder].
type Error struct {
	// Offset is the byte offset in the output where the invalid write occurred.
	Offset int
	// Op is the operation that failed, e.g. "Name".
	Op string
	// Reason describes the violated invariant.
	Reason string
}

func (e *Error) Error() string {
	return fmt.Sprintf("zon: %s at offset %d: %s", e.Op, e.Offset, e.Reason)
}
