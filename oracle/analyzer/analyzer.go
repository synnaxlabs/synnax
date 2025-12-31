// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

// Package analyzer provides semantic analysis for Oracle schema files.
package analyzer

import (
	"context"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/resolution"
)

type analysisCtx struct {
	context.Context
	diag      *diagnostics.Diagnostics
	table     *resolution.Table
	loader    FileLoader
	ast       parser.ISchemaContext
	filePath  string
	namespace string
}

func Analyze(
	ctx context.Context,
	files []string,
	loader FileLoader,
) (*resolution.Table, *diagnostics.Diagnostics) {
	diag := &diagnostics.Diagnostics{}
	table := resolution.NewTable()

	for _, file := range files {
		// Mark file as imported to prevent duplicate loading via import statements
		// Import paths don't have the .oracle extension, so we strip it
		importPath := strings.TrimSuffix(file, ".oracle")
		if table.IsImported(importPath) {
			continue
		}
		table.MarkImported(importPath)

		source, filePath, err := loader.Load(file)
		if err != nil {
			diag.AddErrorf(nil, file, "failed to load file: %v", err)
			continue
		}
		ast, parseDiag := parser.Parse(source)
		if parseDiag != nil && parseDiag.HasErrors() {
			for i := range *parseDiag {
				(*parseDiag)[i].File = filePath
			}
			diag.Merge(*parseDiag)
			continue
		}
		c := &analysisCtx{
			Context:   ctx,
			diag:      diag,
			table:     table,
			loader:    loader,
			ast:       ast,
			filePath:  filePath,
			namespace: DeriveNamespace(filePath),
		}
		analyze(c)
	}
	if diag.HasErrors() {
		return nil, diag
	}
	return table, diag
}

func AnalyzeSource(
	ctx context.Context,
	source, namespace string,
	loader FileLoader,
) (*resolution.Table, *diagnostics.Diagnostics) {
	diag := &diagnostics.Diagnostics{}
	table := resolution.NewTable()

	ast, parseDiag := parser.Parse(source)
	if parseDiag != nil && parseDiag.HasErrors() {
		diag.Merge(*parseDiag)
		return nil, diag
	}
	c := &analysisCtx{
		Context:   ctx,
		diag:      diag,
		table:     table,
		loader:    loader,
		ast:       ast,
		filePath:  namespace + ".oracle",
		namespace: namespace,
	}
	analyze(c)
	if diag.HasErrors() {
		return nil, diag
	}
	return table, diag
}

func analyze(c *analysisCtx) {
	if c.ast == nil {
		return
	}
	for _, def := range c.ast.AllDefinition() {
		if s := def.StructDef(); s != nil {
			collectStruct(c, s)
		}
		if e := def.EnumDef(); e != nil {
			collectEnum(c, e)
		}
	}
	for _, imp := range c.ast.AllImportStmt() {
		path := strings.Trim(imp.STRING_LIT().GetText(), `"`)
		if c.table.IsImported(path) {
			continue
		}
		c.table.MarkImported(path)
		source, filePath, err := c.loader.Load(path)
		if err != nil {
			c.diag.AddErrorf(imp, c.filePath, "failed to import %s: %v", path, err)
			continue
		}
		ast, parseDiag := parser.Parse(source)
		if parseDiag != nil && parseDiag.HasErrors() {
			for i := range *parseDiag {
				(*parseDiag)[i].File = filePath
			}
			c.diag.Merge(*parseDiag)
			continue
		}
		ic := &analysisCtx{
			Context:   c.Context,
			diag:      c.diag,
			table:     c.table,
			loader:    c.loader,
			ast:       ast,
			filePath:  filePath,
			namespace: DeriveNamespace(filePath),
		}
		analyze(ic)
	}
	for _, s := range c.table.StructsInNamespace(c.namespace) {
		for _, f := range s.Fields {
			resolveType(c, s, f.TypeRef)
		}
		// Also resolve type parameter constraints and defaults
		for _, tp := range s.TypeParams {
			if tp.Constraint != nil {
				resolveType(c, nil, tp.Constraint)
			}
			if tp.Default != nil {
				resolveType(c, nil, tp.Default)
			}
		}
		// Resolve alias target type
		if s.AliasOf != nil {
			resolveType(c, s, s.AliasOf)
		}
	}
}

func collectStruct(c *analysisCtx, def parser.IStructDefContext) {
	switch d := def.(type) {
	case *parser.StructFullContext:
		collectStructFull(c, d)
	case *parser.StructAliasContext:
		collectStructAlias(c, d)
	}
}

func collectStructFull(c *analysisCtx, def *parser.StructFullContext) {
	name := def.IDENT().GetText()
	qname := c.namespace + "." + name
	if _, exists := c.table.GetStruct(qname); exists {
		c.diag.AddErrorf(def, c.filePath, "duplicate struct definition: %s", qname)
		return
	}
	entry := &resolution.StructEntry{
		AST:           def,
		Name:          name,
		Namespace:     c.namespace,
		FilePath:      c.filePath,
		QualifiedName: qname,
		Fields:        nil,
		Domains:       make(map[string]*resolution.DomainEntry),
		TypeParams:    collectTypeParams(def.TypeParams()),
	}
	if body := def.StructBody(); body != nil {
		for _, f := range body.AllFieldDef() {
			collectField(c, entry, f)
		}
		for _, d := range body.AllDomainDef() {
			if de := collectDomain(d); de != nil {
				entry.Domains[de.Name] = de
			}
		}
	}
	c.table.AddStruct(entry)
}

func collectStructAlias(c *analysisCtx, def *parser.StructAliasContext) {
	name := def.IDENT().GetText()
	qname := c.namespace + "." + name
	if _, exists := c.table.GetStruct(qname); exists {
		c.diag.AddErrorf(def, c.filePath, "duplicate struct definition: %s", qname)
		return
	}
	entry := &resolution.StructEntry{
		AST:           def,
		Name:          name,
		Namespace:     c.namespace,
		FilePath:      c.filePath,
		QualifiedName: qname,
		Domains:       make(map[string]*resolution.DomainEntry),
		TypeParams:    collectTypeParams(def.TypeParams()),
		AliasOf:       collectTypeRef(def.TypeRef()),
	}
	// Collect domains from alias body if present
	if body := def.AliasBody(); body != nil {
		for _, d := range body.AllDomainDef() {
			if de := collectDomain(d); de != nil {
				entry.Domains[de.Name] = de
			}
		}
	}
	c.table.AddStruct(entry)
}

// collectTypeParams parses generic type parameters from a struct definition.
// Examples: <T>, <T, U>, <T extends schema>, <T extends schema = never>
func collectTypeParams(params parser.ITypeParamsContext) []*resolution.TypeParam {
	if params == nil {
		return nil
	}
	var result []*resolution.TypeParam
	for _, p := range params.AllTypeParam() {
		tp := &resolution.TypeParam{Name: p.IDENT().GetText()}
		// Handle constraint (extends X) and default (= Y)
		// TypeParam rule: IDENT (EXTENDS typeRef)? (EQUALS typeRef)?
		typeRefs := p.AllTypeRef()
		hasExtends := p.EXTENDS() != nil
		hasEquals := p.EQUALS() != nil
		if hasExtends && len(typeRefs) > 0 {
			tp.Constraint = parseTypeRefBasic(typeRefs[0])
		}
		if hasEquals {
			// Default is the last typeRef when EQUALS is present
			idx := 0
			if hasExtends {
				idx = 1
			}
			if idx < len(typeRefs) {
				tp.Default = parseTypeRefBasic(typeRefs[idx])
			}
		}
		result = append(result, tp)
	}
	return result
}

// collectTypeRef creates a TypeRef from a parser context.
// Alias for parseTypeRefBasic, used for struct aliases.
func collectTypeRef(tr parser.ITypeRefContext) *resolution.TypeRef {
	return parseTypeRefBasic(tr)
}

// parseTypeRefBasic creates a basic unresolved TypeRef from a parser context.
// Used for type parameter constraints and defaults where we just need the raw type.
func parseTypeRefBasic(tr parser.ITypeRefContext) *resolution.TypeRef {
	isOptional, isNullable := extractTypeModifiers(tr)
	return &resolution.TypeRef{
		Kind:       resolution.TypeKindUnresolved,
		IsArray:    tr.LBRACKET() != nil,
		IsOptional: isOptional,
		IsNullable: isNullable,
		RawType:    extractType(tr),
		TypeArgs:   collectTypeArgs(tr.TypeArgs()),
	}
}

// collectTypeArgs parses type arguments from a type reference.
// Example: Status<Foo, Bar> -> [Foo, Bar]
func collectTypeArgs(args parser.ITypeArgsContext) []*resolution.TypeRef {
	if args == nil {
		return nil
	}
	var result []*resolution.TypeRef
	for _, tr := range args.AllTypeRef() {
		result = append(result, parseTypeRefBasic(tr))
	}
	return result
}

func collectField(
	c *analysisCtx,
	s *resolution.StructEntry,
	def parser.IFieldDefContext,
) {
	name := def.IDENT().GetText()
	if s.Field(name) != nil {
		c.diag.AddErrorf(def, c.filePath, "duplicate field: %s.%s", s.Name, name)
		return
	}
	tr := def.TypeRef()
	isOptional, isNullable := extractTypeModifiers(tr)
	entry := &resolution.FieldEntry{
		AST:  def,
		Name: name,
		TypeRef: &resolution.TypeRef{
			Kind:       resolution.TypeKindUnresolved,
			IsArray:    tr.LBRACKET() != nil,
			IsOptional: isOptional,
			IsNullable: isNullable,
			RawType:    extractType(tr),
			TypeArgs:   collectTypeArgs(tr.TypeArgs()),
		},
		Domains: make(map[string]*resolution.DomainEntry),
	}
	if fb := def.FieldBody(); fb != nil {
		for _, d := range fb.AllDomainDef() {
			if de := collectDomain(d); de != nil {
				entry.Domains[de.Name] = de
				if de.Name == "id" {
					s.HasIDDomain = true
				}
			}
		}
	}
	s.Fields = append(s.Fields, entry)
}

func collectDomain(def parser.IDomainDefContext) *resolution.DomainEntry {
	entry := &resolution.DomainEntry{AST: def, Name: def.IDENT().GetText()}
	if body := def.DomainBody(); body != nil {
		for _, e := range body.AllExpression() {
			expr := &resolution.ExpressionEntry{AST: e, Name: e.IDENT().GetText()}
			for _, v := range e.AllExpressionValue() {
				expr.Values = append(expr.Values, collectValue(v))
			}
			entry.Expressions = append(entry.Expressions, expr)
		}
	}
	return entry
}

func collectValue(v parser.IExpressionValueContext) resolution.ExpressionValue {
	if s := v.STRING_LIT(); s != nil {
		t := s.GetText()
		return resolution.ExpressionValue{
			Kind:        resolution.ValueKindString,
			StringValue: t[1 : len(t)-1],
		}
	}
	if i := v.INT_LIT(); i != nil {
		n, _ := strconv.ParseInt(i.GetText(), 10, 64)
		return resolution.ExpressionValue{Kind: resolution.ValueKindInt, IntValue: n}
	}
	if f := v.FLOAT_LIT(); f != nil {
		n, _ := strconv.ParseFloat(f.GetText(), 64)
		return resolution.ExpressionValue{Kind: resolution.ValueKindFloat, FloatValue: n}
	}
	if b := v.BOOL_LIT(); b != nil {
		return resolution.ExpressionValue{
			Kind:      resolution.ValueKindBool,
			BoolValue: b.GetText() == "true",
		}
	}
	if q := v.QualifiedIdent(); q != nil {
		ids := q.AllIDENT()
		if len(ids) == 2 {
			return resolution.ExpressionValue{
				Kind:       resolution.ValueKindIdent,
				IdentValue: ids[0].GetText() + "." + ids[1].GetText(),
			}
		}
		return resolution.ExpressionValue{
			Kind:       resolution.ValueKindIdent,
			IdentValue: ids[0].GetText(),
		}
	}
	return resolution.ExpressionValue{Kind: resolution.ValueKindIdent}
}

func collectEnum(c *analysisCtx, def parser.IEnumDefContext) {
	name := def.IDENT().GetText()
	qname := c.namespace + "." + name
	if _, exists := c.table.GetEnum(qname); exists {
		c.diag.AddErrorf(def, c.filePath, "duplicate enum definition: %s", qname)
		return
	}
	entry := &resolution.EnumEntry{
		AST:           def,
		Name:          name,
		Namespace:     c.namespace,
		FilePath:      c.filePath,
		QualifiedName: qname,
		ValuesByName:  make(map[string]*resolution.EnumValue),
		Domains:       make(map[string]*resolution.DomainEntry),
	}
	// Collect enum values and domains from enum body
	if body := def.EnumBody(); body != nil {
		vals := body.AllEnumValue()
		if len(vals) > 0 {
			entry.IsIntEnum = vals[0].INT_LIT() != nil
		}
		for _, v := range vals {
			ev := &resolution.EnumValue{Name: v.IDENT().GetText()}
			if i := v.INT_LIT(); i != nil {
				ev.IntValue, _ = strconv.ParseInt(i.GetText(), 10, 64)
			} else if s := v.STRING_LIT(); s != nil {
				t := s.GetText()
				ev.StringValue = t[1 : len(t)-1]
			}
			entry.Values = append(entry.Values, ev)
			entry.ValuesByName[ev.Name] = ev
		}
		for _, d := range body.AllDomainDef() {
			if de := collectDomain(d); de != nil {
				entry.Domains[de.Name] = de
			}
		}
	}
	c.table.AddEnum(entry)
}

// resolveType resolves a type reference to its concrete type.
// The struct parameter provides context for resolving type parameters within generic structs.
func resolveType(c *analysisCtx, currentStruct *resolution.StructEntry, t *resolution.TypeRef) {
	parts := strings.Split(t.RawType, ".")
	ns, name := c.namespace, parts[0]
	if len(parts) == 2 {
		ns, name = parts[0], parts[1]
	}

	// Check if this is a type parameter reference (e.g., field value T in struct Box<T>)
	if currentStruct != nil && len(parts) == 1 {
		if tp := currentStruct.TypeParam(name); tp != nil {
			t.Kind, t.TypeParamRef = resolution.TypeKindTypeParam, tp
			return
		}
	}

	if resolution.IsPrimitive(name) && len(parts) == 1 {
		t.Kind, t.Primitive = resolution.TypeKindPrimitive, name
		return
	}
	if s, ok := c.table.LookupStruct(ns, name); ok {
		t.Kind, t.StructRef = resolution.TypeKindStruct, s
		// Recursively resolve type arguments
		for _, arg := range t.TypeArgs {
			resolveType(c, currentStruct, arg)
		}
		return
	}
	if e, ok := c.table.LookupEnum(ns, name); ok {
		t.Kind, t.EnumRef = resolution.TypeKindEnum, e
		return
	}
	c.diag.AddWarningf(nil, c.filePath, "unresolved type: %s", t.RawType)
}

func extractType(tr parser.ITypeRefContext) string {
	ids := tr.QualifiedIdent().AllIDENT()
	if len(ids) == 2 {
		return ids[0].GetText() + "." + ids[1].GetText()
	}
	return ids[0].GetText()
}

func extractTypeModifiers(tr parser.ITypeRefContext) (isOptional, isNullable bool) {
	mods := tr.TypeModifiers()
	if mods == nil {
		return false, false
	}
	return mods.QUESTION() != nil, mods.BANG() != nil
}
