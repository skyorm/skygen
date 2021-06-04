package gen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strings"
)

// File extracts structs information from file.
func File(path string) ([]Struct, error) {
	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var res []Struct
	for _, decl := range fileNode.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			doc := ts.Doc
			if doc == nil && len(gd.Specs) == 1 {
				doc = gd.Doc
			}
			if doc == nil {
				continue
			}
			sm := commentRegexp.FindStringSubmatch(commentText(doc))
			if len(sm) < 2 {
				continue
			}
			table := sm[1]
			str, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			if str.Incomplete {
				continue
			}
			s, err := parseStructTypeSpec(ts, str)
			if err != nil {
				return nil, err
			}
			s.StoreName = table
			res = append(res, *s)
		}
	}

	return res, nil
}

var commentRegexp = regexp.MustCompile("sky:([0-9A-Za-z_]+)")

func fileGoType(x ast.Expr) string {
	switch t := x.(type) {
	case *ast.StarExpr:
		return "*" + fileGoType(t.X)
	case *ast.SelectorExpr:
		return fileGoType(t.X) + "." + t.Sel.String()
	case *ast.Ident:
		s := t.String()
		if s == "byte" {
			return "uint8"
		}
		return s
	case *ast.ArrayType:
		return "[" + fileGoType(t.Len) + "]" + fileGoType(t.Elt)
	case *ast.BasicLit:
		return t.Value
	case nil:
		return ""
	default:
		panic(fmt.Sprintf("skygen: fileGoType: unhandled '%s' (%#v). Please report this bug.", x, x))
	}
}

func commentText(g *ast.CommentGroup) string {
	v := make([]string, len(g.List))
	for i, c := range g.List {
		v[i] = c.Text
	}
	return strings.Join(v, " ")
}

func parseStructTypeSpec(ts *ast.TypeSpec, str *ast.StructType) (*Struct, error) {
	res := &Struct{
		Type:    ts.Name.Name,
		PkIndex: -1,
	}
	var n int
	for _, f := range str.Fields.List {
		if f.Tag == nil {
			continue
		}
		tag := f.Tag.Value
		if len(tag) < 3 {
			continue
		}
		tag = reflect.StructTag(tag[1 : len(tag)-1]).Get("sky") // strip quotes
		if tag == "" || tag == "-" {
			continue
		}
		if len(f.Names) == 0 {
			return nil, fmt.Errorf(
				"skygen: %s has anonymous field %s with 'skygen:' tag, it is not allowed",
				res.Type, f.Type)
		}
		if len(f.Names) != 1 {
			panic(fmt.Sprintf("skygen: %d names: %#v. Please report this bug.", len(f.Names), f.Names))
		}
		name := f.Names[0]
		if !name.IsExported() {
			return nil, fmt.Errorf(
				"skygen: %s has non-exported field %s with 'skygen:' tag, it is not allowed",
				res.Type, name.Name)
		}
		column, isPK := parseFieldTag(tag)
		if column == "" {
			return nil, fmt.Errorf(
				"skygen: %s has field %s with invalid 'skygen:' tag value, it is not allowed",
				res.Type, name.Name)
		}
		typ := fileGoType(f.Type)
		if isPK {
			if strings.HasPrefix(typ, "*") {
				return nil, fmt.Errorf(
					"skygen: %s has pointer field %s with with 'pk' label in 'skygen:' tag, it is not allowed",
					res.Type, name.Name)
			}
			if strings.HasPrefix(typ, "[") {
				return nil, fmt.Errorf(
					"skygen: %s has slice field %s with with 'pk' label in 'skygen:' tag, it is not allowed",
					res.Type, name.Name)
			}
			if res.PkIndex >= 0 {
				return nil, fmt.Errorf(
					"skygen: %s has field %s with with duplicate 'pk' label in 'skygen:' tag "+
						"(first used by %s), it is not allowed", res.Type, name.Name, res.Fields[res.PkIndex].Name)
			}
		}
		res.Fields = append(res.Fields, Field{
			Name:   name.Name,
			Type:   typ,
			Column: column,
			IsPK:   isPK,
		})
		if isPK {
			res.PkIndex = n
		}
		n++
	}
	if err := checkFields(res); err != nil {
		return nil, err
	}
	return res, nil
}

func parseFieldTag(t string) (name string, isPk bool) {
	p := strings.Split(t, ",")
	if len(p) == 0 || len(p) > 2 {
		return
	}
	if len(p) == 2 {
		switch p[1] {
		case "pk":
			isPk = true
		default:
			return
		}
	}
	name = p[0]
	return
}

func checkFields(s *Struct) error {
	if len(s.Fields) == 0 {
		return fmt.Errorf("skygen: %s has no fields with 'skygen:' tag, it is not allowed", s.Type)
	}
	dupes := make(map[string]string)
	for _, f := range s.Fields {
		if f2, ok := dupes[f.Column]; ok {
			return fmt.Errorf(
				"skygen: %s has field %s with 'skygen:' tag with duplicate column name %s (used by %s), "+
					"it is not allowed", s.Type, f.Name, f.Column, f2)
		}
		dupes[f.Column] = f.Name
	}
	return nil
}
