// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parse

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	// JerkyDirective is the go:generate directive that marks a struct for jerky generation.
	JerkyDirective = "//go:generate jerky"
)

// Parser parses Go source files to extract jerky-annotated structs.
type Parser struct {
	// fset is the file set for source locations.
	pkg *packages.Package
}

// NewParser creates a new Parser.
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile parses a Go source file and returns all jerky-annotated structs.
func (p *Parser) ParseFile(sourceFile string) ([]ParsedStruct, error) {
	absPath, err := filepath.Abs(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	dir := filepath.Dir(absPath)

	// Load the package containing the source file
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports,
		Dir: dir,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in %s", dir)
	}

	if len(pkgs[0].Errors) > 0 {
		var errs []string
		for _, e := range pkgs[0].Errors {
			errs = append(errs, e.Error())
		}
		return nil, fmt.Errorf("package errors: %s", strings.Join(errs, "; "))
	}

	pkg := pkgs[0]
	p.pkg = pkg

	// Find the specific file
	var targetFile *ast.File
	for i, f := range pkg.GoFiles {
		if f == absPath || filepath.Base(f) == filepath.Base(absPath) {
			targetFile = pkg.Syntax[i]
			break
		}
	}

	if targetFile == nil {
		return nil, fmt.Errorf("file %s not found in package", sourceFile)
	}

	// Find all jerky-annotated structs
	var structs []ParsedStruct

	ast.Inspect(targetFile, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		// Check if this declaration has a jerky directive
		if !hasJerkyDirective(genDecl.Doc) {
			return true
		}

		// Process type specs in this declaration
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			parsed, err := p.parseStruct(typeSpec, structType, absPath, pkg)
			if err != nil {
				// Log error but continue
				continue
			}

			structs = append(structs, parsed)
		}

		return true
	})

	return structs, nil
}

// hasJerkyDirective checks if a comment group contains the jerky directive.
func hasJerkyDirective(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, comment := range doc.List {
		if strings.HasPrefix(strings.TrimSpace(comment.Text), JerkyDirective) {
			return true
		}
	}
	return false
}

// parseStruct parses a single struct into a ParsedStruct.
func (p *Parser) parseStruct(
	typeSpec *ast.TypeSpec,
	structType *ast.StructType,
	sourceFile string,
	pkg *packages.Package,
) (ParsedStruct, error) {
	parsed := ParsedStruct{
		Name:        typeSpec.Name.Name,
		PackagePath: pkg.PkgPath,
		PackageName: pkg.Name,
		SourceFile:  sourceFile,
		Position:    pkg.Fset.Position(typeSpec.Pos()),
	}

	// Parse fields
	fieldNum := 1
	for _, field := range structType.Fields.List {
		// Skip embedded fields for now
		if len(field.Names) == 0 {
			continue
		}

		for _, name := range field.Names {
			// Skip unexported fields
			if !ast.IsExported(name.Name) {
				continue
			}

			goType := p.resolveType(field.Type, pkg)
			tags := parseTags(field.Tag)

			parsed.Fields = append(parsed.Fields, ParsedField{
				Name:        name.Name,
				GoType:      goType,
				Tags:        tags,
				FieldNumber: fieldNum,
				Position:    pkg.Fset.Position(name.Pos()),
			})
			fieldNum++
		}
	}

	return parsed, nil
}

// resolveType resolves an AST expression to a GoType.
func (p *Parser) resolveType(expr ast.Expr, pkg *packages.Package) GoType {
	// Try to get type info from the type checker
	if pkg.TypesInfo != nil {
		if tv, ok := pkg.TypesInfo.Types[expr]; ok {
			return p.typeFromTypesType(tv.Type)
		}
	}

	// Fall back to AST-based resolution
	return p.typeFromAST(expr, pkg)
}

// typeFromTypesType converts a types.Type to a GoType.
func (p *Parser) typeFromTypesType(t types.Type) GoType {
	switch t := t.(type) {
	case *types.Basic:
		return GoType{
			Kind: KindPrimitive,
			Name: t.Name(),
		}

	case *types.Named:
		obj := t.Obj()
		pkgPath := ""
		pkgName := ""
		if obj.Pkg() != nil {
			pkgPath = obj.Pkg().Path()
			pkgName = obj.Pkg().Name()
		}

		gt := GoType{
			Kind:        KindNamed,
			Name:        formatTypeName(pkgName, obj.Name()),
			PackagePath: pkgPath,
			PackageName: pkgName,
		}

		// Check if this is a type alias (e.g., type Key uint32)
		underlying := t.Underlying()
		if _, isBasic := underlying.(*types.Basic); isBasic {
			gt.Underlying = &GoType{
				Kind: KindPrimitive,
				Name: underlying.(*types.Basic).Name(),
			}
		}

		return gt

	case *types.Slice:
		elem := p.typeFromTypesType(t.Elem())
		return GoType{
			Kind: KindSlice,
			Name: "[]" + elem.Name,
			Elem: &elem,
		}

	case *types.Map:
		key := p.typeFromTypesType(t.Key())
		elem := p.typeFromTypesType(t.Elem())
		return GoType{
			Kind: KindMap,
			Name: "map[" + key.Name + "]" + elem.Name,
			Key:  &key,
			Elem: &elem,
		}

	case *types.Pointer:
		elem := p.typeFromTypesType(t.Elem())
		return GoType{
			Kind: KindPointer,
			Name: "*" + elem.Name,
			Elem: &elem,
		}

	case *types.Struct:
		return GoType{
			Kind: KindStruct,
			Name: "struct{}",
		}

	default:
		return GoType{
			Kind: KindPrimitive,
			Name: t.String(),
		}
	}
}

// typeFromAST converts an AST expression to a GoType (fallback when type info unavailable).
func (p *Parser) typeFromAST(expr ast.Expr, pkg *packages.Package) GoType {
	switch t := expr.(type) {
	case *ast.Ident:
		return GoType{
			Kind: KindPrimitive,
			Name: t.Name,
		}

	case *ast.SelectorExpr:
		pkgName := ""
		if ident, ok := t.X.(*ast.Ident); ok {
			pkgName = ident.Name
		}
		return GoType{
			Kind:        KindNamed,
			Name:        formatTypeName(pkgName, t.Sel.Name),
			PackageName: pkgName,
		}

	case *ast.ArrayType:
		elem := p.typeFromAST(t.Elt, pkg)
		if t.Len == nil {
			// Slice
			return GoType{
				Kind: KindSlice,
				Name: "[]" + elem.Name,
				Elem: &elem,
			}
		}
		// Array - treat as slice for proto purposes
		return GoType{
			Kind: KindSlice,
			Name: "[]" + elem.Name,
			Elem: &elem,
		}

	case *ast.MapType:
		key := p.typeFromAST(t.Key, pkg)
		elem := p.typeFromAST(t.Value, pkg)
		return GoType{
			Kind: KindMap,
			Name: "map[" + key.Name + "]" + elem.Name,
			Key:  &key,
			Elem: &elem,
		}

	case *ast.StarExpr:
		elem := p.typeFromAST(t.X, pkg)
		return GoType{
			Kind: KindPointer,
			Name: "*" + elem.Name,
			Elem: &elem,
		}

	default:
		return GoType{
			Kind: KindPrimitive,
			Name: "unknown",
		}
	}
}

// formatTypeName formats a type name with optional package prefix.
func formatTypeName(pkgName, typeName string) string {
	if pkgName == "" {
		return typeName
	}
	return pkgName + "." + typeName
}

// parseTags parses struct field tags.
func parseTags(tag *ast.BasicLit) StructTags {
	if tag == nil {
		return StructTags{}
	}

	// Remove quotes from tag value
	tagValue := strings.Trim(tag.Value, "`")

	st := reflect.StructTag(tagValue)

	return StructTags{
		JSON:    st.Get("json"),
		Msgpack: st.Get("msgpack"),
		Raw:     tagValue,
	}
}
