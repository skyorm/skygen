package gen

import (
	"fmt"
)

// Field struct.
type Field struct {
	Name   string
	Type   string
	Column string
	IsPK   bool
}

// GoString implements fmt.GoStringer interface..
func (fi *Field) GoString() string {
	return fmt.Sprintf("{Name: %q, Type: %q, Column: %q}", fi.Name, fi.Type, fi.Column)
}
