package gen

import (
	"fmt"
)

// Struct struct.
type Struct struct {
	Type      string
	StoreName string
	Fields    []Field
	PkIndex   int
}

// GoString returns struct information as Go code string.
func (s *Struct) GoString() string {
	res := "gen.Struct{\n"
	res += fmt.Sprintf("\tType: %q,\n", s.Type)
	res += fmt.Sprintf("\tSQLName: %q,\n", s.StoreName)
	res += "\tFields: []gen.Field{\n"
	for _, f := range s.Fields {
		res += fmt.Sprintf("\t\t%s,\n", f.GoString())
	}
	res += "\t},\n"
	res += fmt.Sprintf("\tPKFieldIndex: %d,\n", s.PkIndex)
	res += "}"
	return res
}

// HasPK returns true if the struct has a PK.
func (s *Struct) HasPK() bool {
	return s.PkIndex >= 0
}

// PKField returns a primary key field, panics for views.
func (s *Struct) PKField() Field {
	if !s.HasPK() {
		panic("skygen: no pk field")
	}
	return s.Fields[s.PkIndex]
}
