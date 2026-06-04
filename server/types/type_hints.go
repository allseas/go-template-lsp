// Package types implements utils for types and the type tree
package types

import (
	"bufio"
	"bytes"
	"fmt"
	"go/types"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/tools/go/packages"
)

// TypeHint represents a `gotype:` type hint found in a template file.
type TypeHint struct {
	Line int
	Type string
}

// ParseTypeHints find the first match of the type hint
func ParseTypeHints(f io.Reader) []TypeHint {
	// Regex to capture a gotype hint inside a Go template comment.
	// Supports optional trimming dashes and whitespace around delimiters, e.g.:
	// {{/*gotype: Type*/}}, {{- /* gotype: pkg.Type */ -}}, {{/*gotype: path/to/pkg.Type*/}} etc.
	// Notes:
	// - Allow package paths with "/" before the final ".Type" segment.
	// - Still capture the entire token so we can later reduce to the final type name.
	re := regexp.MustCompile(`gotype:\s*([A-Za-z_][A-Za-z0-9_/.-]*)`)
	var hints []TypeHint

	scanner := bufio.NewScanner(f)
	lineNo := 0
	gotypeBytes := []byte("gotype:")
	for scanner.Scan() {
		lineNo++
		line := scanner.Bytes()
		if !bytes.Contains(line, gotypeBytes) {
			continue
		}
		matches := re.FindAllSubmatch(line, -1)
		for _, m := range matches {
			if len(m) >= 2 {
				hints = append(hints, TypeHint{Line: lineNo, Type: string(m[1])})
			}
		}
	}
	return hints
}

// TypeField is a resolved field from a struct type.
type TypeField struct {
	Name     string
	TypeName string
	Type     types.Type // actual type object
	Embedded bool
}

// MethodType is the struct for the functions in the model.
type MethodType struct {
	Func       *types.Func
	Name       string
	ReturnName string
	ReturnType types.Type
	Params     []ParamType
}

// ParamType is needed to extract parameter types of a function
type ParamType struct {
	Name     string
	Type     types.Type
	TypeName string
}

// LoadTypeFromHint loads the Go package identified by the hint and returns a
// Tree with DotType and Pkg set.
func LoadTypeFromHint(hint, workspaceRoot string) (*Tree, error) {
	importPath, typeName := splitTypeHint(hint)

	log.Debug().
		Str("hint", hint).
		Str("importPath", importPath).
		Str("typeName", typeName).
		Str("workspaceRoot", workspaceRoot).
		Msg("LoadTypeFromHint: attempting to load type")

	// possibly add packages.NeedTypesInfo | packages.NeedImports |  packages.NeedName | packages.NeedFiles | packages.NeedSyntax later (some used in code_gen)
	dir := workspaceRoot
	if _, err := os.Stat(dir); err != nil {
		log.Warn().
			Str("dir", dir).
			Msg("LoadTypeFromHint: workspace root does not exist on disk, using process cwd")
		if cwd, cwdErr := os.Getwd(); cwdErr == nil {
			dir = cwd
		}
	}
	cfg := &packages.Config{
		Mode: packages.NeedTypes,
		Dir:  dir,
	}

	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		log.Error().
			Err(err).
			Str("importPath", importPath).
			Str("dir", workspaceRoot).
			Msg("LoadTypeFromHint: packages.Load failed")
		return nil, fmt.Errorf("packages.Load(%q): %w", importPath, err)
	}
	if len(pkgs) == 0 {
		log.Error().Str("importPath", importPath).Msg("LoadTypeFromHint: no packages found")
		return nil, fmt.Errorf("no packages found for import path %q", importPath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		log.Error().
			Str("importPath", importPath).
			Str("error", pkg.Errors[0].Msg).
			Msg("LoadTypeFromHint: package has errors")
		return nil, fmt.Errorf("package %q has errors: %v", importPath, pkg.Errors[0])
	}

	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		log.Error().
			Str("typeName", typeName).
			Str("importPath", importPath).
			Msg("LoadTypeFromHint: type not found in package scope")
		return nil, fmt.Errorf("type %q not found in package %q", typeName, importPath)
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		log.Error().Str("typeName", typeName).Msg("LoadTypeFromHint: type is not a named type")
		return nil, fmt.Errorf("%q is not a named type in package %q", typeName, importPath)
	}

	log.Debug().
		Str("typeName", typeName).
		Str("importPath", importPath).
		Int("numFields", named.Underlying().(*types.Struct).NumFields()).
		Int("numMethods", named.NumMethods()).
		Msg("LoadTypeFromHint: type loaded successfully")

	tree := &Tree{DotType: named, Pkg: pkg.Types}
	return tree, nil
}

// splitTypeHint splits a raw gotype hint into (importPath, typeName).
func splitTypeHint(hint string) (importPath, typeName string) {
	idx := strings.LastIndex(hint, ".")
	if idx == -1 {
		return ".", hint
	}
	return hint[:idx], hint[idx+1:]
}

// NamedMethods extracts the methods from the model
func NamedMethods(named *types.Named) []MethodType {
	var methods []MethodType
	for i := range named.NumMethods() {
		fn := named.Method(i)
		if !fn.Exported() {
			continue
		}

		sig := fn.Signature()
		results := sig.Results()

		if results.Len() == 0 || results.Len() > 2 {
			continue
		}

		var params []ParamType
		// if the generics are used in the functions, then sig.TypeParams should be extracted
		sigParams := sig.Params()
		for j := range sigParams.Len() {
			p := sigParams.At(j)
			params = append(params, ParamType{
				Name:     p.Name(),
				Type:     p.Type(),
				TypeName: types.TypeString(p.Type(), nil),
			})
		}

		ret := results.At(0)
		methods = append(methods, MethodType{
			Func:       fn,
			Name:       fn.Name(),
			ReturnType: ret.Type(),
			ReturnName: types.TypeString(ret.Type(), nil),
			Params:     params,
		})
	}
	return methods
}

// StructFields returns the exported fields of the struct
func StructFields(named *types.Named) []TypeField {
	// Underlying returns structs fields and types
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil
	}

	fields := make([]TypeField, 0, st.NumFields())
	for i := range st.NumFields() {
		f := st.Field(i)
		// we can't access unexported fields
		if !f.Exported() {
			continue
		}
		fields = append(fields, TypeField{
			Name:     f.Name(),
			TypeName: types.TypeString(f.Type(), nil),
			Type:     f.Type(),
			Embedded: f.Embedded(),
		})
	}
	return fields
}
