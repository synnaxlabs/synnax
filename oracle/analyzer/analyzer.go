// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package analyzer provides semantic analysis for Oracle schema files.
package analyzer

import (
	"context"
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
)

type analysisCtx struct {
	context.Context
	diag        *diagnostics.Diagnostics
	table       *resolution.Table
	loader      FileLoader
	ast         parser.ISchemaContext
	filePath    string
	namespace   string
	fileDomains map[string]resolution.Domain // file-level domains for inheritance
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
			Context:     ctx,
			diag:        diag,
			table:       table,
			loader:      loader,
			ast:         ast,
			filePath:    filePath,
			namespace:   DeriveNamespace(filePath),
			fileDomains: make(map[string]resolution.Domain),
		}
		analyze(c)
	}
	if diag.HasErrors() {
		return nil, diag
	}
	// Detect recursive types after all types are resolved
	detectRecursiveTypes(table)
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
		Context:     ctx,
		diag:        diag,
		table:       table,
		loader:      loader,
		ast:         ast,
		filePath:    namespace + ".oracle",
		namespace:   namespace,
		fileDomains: make(map[string]resolution.Domain),
	}
	analyze(c)
	if diag.HasErrors() {
		return nil, diag
	}
	// Detect recursive types after all types are resolved
	detectRecursiveTypes(table)
	return table, diag
}

func analyze(c *analysisCtx) {
	if c.ast == nil {
		return
	}

	// Collect file-level domains first (they apply to all definitions)
	for _, fd := range c.ast.AllFileDomain() {
		de := collectFileDomain(fd)
		c.fileDomains[de.Name] = de
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
			Context:     c.Context,
			diag:        c.diag,
			table:       c.table,
			loader:      c.loader,
			ast:         ast,
			filePath:    filePath,
			namespace:   DeriveNamespace(filePath),
			fileDomains: make(map[string]resolution.Domain),
		}
		analyze(ic)
	}
	structs := c.table.StructsInNamespace(c.namespace)
	for i := range structs {
		s := &structs[i]
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
		// Resolve extends parent type (pass s to resolve type param refs like Status<D>)
		if s.Extends != nil {
			resolveType(c, s, s.Extends)
		}
	}

	// Validate extends relationships after all types are resolved
	structs = c.table.StructsInNamespace(c.namespace)
	for i := range structs {
		validateExtends(c, &structs[i])
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
	entry := resolution.Struct{
		AST:           def,
		Name:          name,
		Namespace:     c.namespace,
		FilePath:      c.filePath,
		QualifiedName: qname,
		Fields:        nil,
		Domains:       make(map[string]resolution.Domain),
		TypeParams:    collectTypeParams(def.TypeParams()),
	}

	// Parse extends clause if present
	if def.EXTENDS() != nil {
		entry.Extends = collectTypeRef(def.TypeRef())
	}

	// Start with file-level domains (inherited)
	for k, v := range c.fileDomains {
		entry.Domains[k] = v
	}

	if body := def.StructBody(); body != nil {
		for _, f := range body.AllFieldDef() {
			collectField(c, &entry, f)
		}
		// Collect field omissions (-fieldName syntax)
		for _, fo := range body.AllFieldOmit() {
			entry.OmittedFields = append(entry.OmittedFields, fo.IDENT().GetText())
		}
		// Struct-level domains merge with file-level domains (struct takes precedence)
		for _, d := range body.AllDomain() {
			de := collectDomain(d)
			if existing, ok := entry.Domains[de.Name]; ok {
				// Merge expressions: existing (file-level) + new (struct-level)
				merged := resolution.Domain{
					AST:         de.AST,
					Name:        de.Name,
					Expressions: append(existing.Expressions, de.Expressions...),
				}
				entry.Domains[de.Name] = merged
			} else {
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
	entry := resolution.Struct{
		AST:           def,
		Name:          name,
		Namespace:     c.namespace,
		FilePath:      c.filePath,
		QualifiedName: qname,
		Domains:       make(map[string]resolution.Domain),
		AliasOf:       collectTypeRef(def.TypeRef()),
		TypeParams:    collectTypeParams(def.TypeParams()),
	}

	// Start with file-level domains (inherited)
	for k, v := range c.fileDomains {
		entry.Domains[k] = v
	}

	// Collect domains from alias body if present (merge with file-level)
	if body := def.AliasBody(); body != nil {
		for _, d := range body.AllDomain() {
			de := collectDomain(d)
			if existing, ok := entry.Domains[de.Name]; ok {
				// Merge expressions: existing (file-level) + new (alias-level)
				merged := resolution.Domain{
					AST:         de.AST,
					Name:        de.Name,
					Expressions: append(existing.Expressions, de.Expressions...),
				}
				entry.Domains[de.Name] = merged
			} else {
				entry.Domains[de.Name] = de
			}
		}
	}
	c.table.AddStruct(entry)
}

// collectTypeParams parses generic type parameters from a struct definition.
// Examples: <T>, <T?>, <T extends Foo>, <T? extends Foo>, <T = Bar>, <T? = Bar>
func collectTypeParams(params parser.ITypeParamsContext) []resolution.TypeParam {
	if params == nil {
		return nil
	}
	var result []resolution.TypeParam
	for _, p := range params.AllTypeParam() {
		tp := resolution.TypeParam{Name: p.IDENT().GetText()}
		// Handle optional marker (?), constraint (extends X), and default (= Y)
		// TypeParam rule: IDENT QUESTION? (EXTENDS typeRef)? (EQUALS typeRef)?
		tp.Optional = p.QUESTION() != nil
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
	// Check if this is a map type
	if mapCtx, ok := tr.(*parser.TypeRefMapContext); ok {
		return parseMapTypeRef(mapCtx)
	}

	// Normal type reference
	normalCtx := tr.(*parser.TypeRefNormalContext)
	isOptional, isHardOptional := extractTypeModifiersNormal(normalCtx)
	return &resolution.TypeRef{
		Kind:           resolution.TypeKindUnresolved,
		IsArray:        normalCtx.LBRACKET() != nil,
		IsOptional:     isOptional,
		IsHardOptional: isHardOptional,
		RawType:        extractTypeNormal(normalCtx),
		TypeArgs:       collectTypeArgsNormal(normalCtx.TypeArgs()),
	}
}

// parseMapTypeRef creates a TypeRef for a map type.
func parseMapTypeRef(mapCtx *parser.TypeRefMapContext) *resolution.TypeRef {
	mt := mapCtx.MapType()
	typeRefs := mt.AllTypeRef()

	var keyType, valueType *resolution.TypeRef
	if len(typeRefs) >= 2 {
		keyType = parseTypeRefBasic(typeRefs[0])
		valueType = parseTypeRefBasic(typeRefs[1])
	}

	isOptional, isHardOptional := false, false
	if mods := mapCtx.TypeModifiers(); mods != nil {
		// Count QUESTION tokens: 1 = soft optional (?), 2 = hard optional (??)
		questionCount := len(mods.AllQUESTION())
		if questionCount >= 2 {
			isHardOptional = true
		} else if questionCount >= 1 {
			isOptional = true
		}
	}

	return &resolution.TypeRef{
		Kind:           resolution.TypeKindMap,
		IsOptional:     isOptional,
		IsHardOptional: isHardOptional,
		MapKeyType:     keyType,
		MapValueType:   valueType,
		RawType:        "map",
	}
}

// collectTypeArgsNormal parses type arguments from a normal type reference.
// Example: Status<Foo, Bar> -> [Foo, Bar]
func collectTypeArgsNormal(args parser.ITypeArgsContext) []*resolution.TypeRef {
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
	s *resolution.Struct,
	def parser.IFieldDefContext,
) {
	name := def.IDENT().GetText()
	if _, found := s.Field(name); found {
		c.diag.AddErrorf(def, c.filePath, "duplicate field: %s.%s", s.Name, name)
		return
	}
	tr := def.TypeRef()
	entry := resolution.Field{
		AST:     def,
		Name:    name,
		TypeRef: parseTypeRefBasic(tr),
		Domains: make(map[string]resolution.Domain),
	}

	// Collect inline domains (e.g., @key, @validate required)
	for _, inl := range def.AllInlineDomain() {
		de := collectInlineDomain(inl)
		entry.Domains[de.Name] = de
		if de.Name == "key" {
			s.HasKeyDomain = true
		}
	}

	// Collect domains from field body if present
	if fb := def.FieldBody(); fb != nil {
		for _, d := range fb.AllDomain() {
			de := collectDomain(d)
			entry.Domains[de.Name] = de
			if de.Name == "key" {
				s.HasKeyDomain = true
			}
		}
	}
	s.Fields = append(s.Fields, entry)
}

// collectFileDomain collects a file-level domain declaration.
func collectFileDomain(fd parser.IFileDomainContext) resolution.Domain {
	entry := resolution.Domain{AST: fd, Name: fd.IDENT().GetText()}
	if content := fd.DomainContent(); content != nil {
		collectDomainContent(&entry, content)
	}
	return entry
}

// collectDomain collects a domain definition (@domain syntax).
func collectDomain(def parser.IDomainContext) resolution.Domain {
	entry := resolution.Domain{AST: def, Name: def.IDENT().GetText()}
	if content := def.DomainContent(); content != nil {
		collectDomainContent(&entry, content)
	}
	return entry
}

// collectInlineDomain collects an inline domain on a field.
func collectInlineDomain(def parser.IInlineDomainContext) resolution.Domain {
	entry := resolution.Domain{AST: def, Name: def.IDENT().GetText()}
	if content := def.DomainContent(); content != nil {
		collectDomainContent(&entry, content)
	}
	return entry
}

// collectDomainContent populates a Domain from its content.
// Content can be either a single expression or a block of expressions.
func collectDomainContent(entry *resolution.Domain, content parser.IDomainContentContext) {
	// Check if it's a block: @domain { expr1\n expr2 }
	if block := content.DomainBlock(); block != nil {
		for _, e := range block.AllExpression() {
			expr := resolution.Expression{AST: e, Name: e.IDENT().GetText()}
			for _, v := range e.AllExpressionValue() {
				expr.Values = append(expr.Values, collectValue(v))
			}
			entry.Expressions = append(entry.Expressions, expr)
		}
		return
	}

	// Single expression: @domain expr
	if e := content.Expression(); e != nil {
		expr := resolution.Expression{AST: e, Name: e.IDENT().GetText()}
		for _, v := range e.AllExpressionValue() {
			expr.Values = append(expr.Values, collectValue(v))
		}
		entry.Expressions = append(entry.Expressions, expr)
	}
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
	entry := resolution.Enum{
		AST:           def,
		Name:          name,
		Namespace:     c.namespace,
		FilePath:      c.filePath,
		QualifiedName: qname,
		Domains:       make(map[string]resolution.Domain),
	}

	// Start with file-level domains (inherited)
	for k, v := range c.fileDomains {
		entry.Domains[k] = v
	}

	// Collect enum values and domains from enum body
	if body := def.EnumBody(); body != nil {
		vals := body.AllEnumValue()
		if len(vals) > 0 {
			entry.IsIntEnum = vals[0].INT_LIT() != nil
		}
		for _, v := range vals {
			ev := resolution.EnumEntry{Name: v.IDENT().GetText()}
			if i := v.INT_LIT(); i != nil {
				ev.ExpressionValue.Kind = resolution.ValueKindInt
				ev.ExpressionValue.IntValue, _ = strconv.ParseInt(i.GetText(), 10, 64)
			} else if s := v.STRING_LIT(); s != nil {
				t := s.GetText()
				ev.ExpressionValue.Kind = resolution.ValueKindString
				ev.ExpressionValue.StringValue = t[1 : len(t)-1]
			}
			entry.Values = append(entry.Values, ev)
		}
		// Enum-level domains merge with file-level domains (enum takes precedence)
		for _, d := range body.AllDomain() {
			de := collectDomain(d)
			if existing, ok := entry.Domains[de.Name]; ok {
				// Merge expressions: existing (file-level) + new (enum-level)
				merged := resolution.Domain{
					AST:         de.AST,
					Name:        de.Name,
					Expressions: append(existing.Expressions, de.Expressions...),
				}
				entry.Domains[de.Name] = merged
			} else {
				entry.Domains[de.Name] = de
			}
		}
	}
	c.table.AddEnum(entry)
}

// resolveType resolves a type reference to its concrete type.
// The struct parameter provides context for resolving type parameters within generic structs.
func resolveType(c *analysisCtx, currentStruct *resolution.Struct, t *resolution.TypeRef) {
	// Map types are already resolved at parse time, but we need to resolve key and value types
	if t.Kind == resolution.TypeKindMap {
		if t.MapKeyType != nil {
			resolveType(c, currentStruct, t.MapKeyType)
		}
		if t.MapValueType != nil {
			resolveType(c, currentStruct, t.MapValueType)
		}
		return
	}

	parts := strings.Split(t.RawType, ".")
	ns, name := c.namespace, parts[0]
	if len(parts) == 2 {
		ns, name = parts[0], parts[1]
	}

	// Check if this is a type parameter reference (e.g., field value T in struct Box<T>)
	if currentStruct != nil && len(parts) == 1 {
		if tp, ok := currentStruct.TypeParam(name); ok {
			t.Kind, t.TypeParamRef = resolution.TypeKindTypeParam, &tp
			return
		}
	}

	if resolution.IsPrimitive(name) && len(parts) == 1 {
		t.Kind, t.Primitive = resolution.TypeKindPrimitive, name
		return
	}
	if s, ok := c.table.LookupStruct(ns, name); ok {
		t.Kind, t.StructRef = resolution.TypeKindStruct, &s
		// Recursively resolve type arguments
		for _, arg := range t.TypeArgs {
			resolveType(c, currentStruct, arg)
		}
		return
	}
	if e, ok := c.table.LookupEnum(ns, name); ok {
		t.Kind, t.EnumRef = resolution.TypeKindEnum, &e
		return
	}
	c.diag.AddWarningf(nil, c.filePath, "unresolved type: %s", t.RawType)
}

// extractTypeNormal extracts the type name from a normal type reference context.
func extractTypeNormal(tr *parser.TypeRefNormalContext) string {
	ids := tr.QualifiedIdent().AllIDENT()
	if len(ids) == 2 {
		return ids[0].GetText() + "." + ids[1].GetText()
	}
	return ids[0].GetText()
}

// detectRecursiveTypes marks structs that reference themselves in any field.
// This is called after all types are resolved.
func detectRecursiveTypes(table *resolution.Table) {
	structs := table.AllStructs()
	for i := range structs {
		if isRecursive(&structs[i]) {
			structs[i].IsRecursive = true
		}
	}
}

// isRecursive checks if a struct references itself directly in any field.
func isRecursive(s *resolution.Struct) bool {
	for _, field := range s.Fields {
		if typeRefersTo(field.TypeRef, s) {
			return true
		}
	}
	return false
}

// typeRefersTo checks if a type reference points to the target struct,
// either directly or through type arguments.
func typeRefersTo(t *resolution.TypeRef, target *resolution.Struct) bool {
	if t == nil {
		return false
	}
	switch t.Kind {
	case resolution.TypeKindStruct:
		if t.StructRef == target {
			return true
		}
		// Check type arguments for generic recursive references (e.g., Node<K>[] where K wraps Node)
		for _, arg := range t.TypeArgs {
			if typeRefersTo(arg, target) {
				return true
			}
		}
	case resolution.TypeKindMap:
		// Check map key and value types
		if typeRefersTo(t.MapKeyType, target) || typeRefersTo(t.MapValueType, target) {
			return true
		}
	}
	return false
}

// extractTypeModifiersNormal extracts optional modifiers from a normal type reference context.
// Returns isOptional (?) and isHardOptional (??)
func extractTypeModifiersNormal(tr *parser.TypeRefNormalContext) (isOptional, isHardOptional bool) {
	mods := tr.TypeModifiers()
	if mods == nil {
		return false, false
	}
	// Count QUESTION tokens: 1 = soft optional (?), 2 = hard optional (??)
	questionCount := len(mods.AllQUESTION())
	if questionCount >= 2 {
		return false, true // ?? = hard optional only
	}
	return questionCount >= 1, false // ? = soft optional only
}

// validateExtends validates the extends relationship for a struct.
// It checks for:
// 1. Parent struct exists and is resolved
// 2. No circular inheritance (A extends B extends A)
// 3. Omitted fields exist in the parent
// 4. Self-extension (A extends A)
func validateExtends(c *analysisCtx, s *resolution.Struct) {
	if s.Extends == nil {
		return
	}

	// Check that parent is resolved to a struct
	if s.Extends.Kind != resolution.TypeKindStruct || s.Extends.StructRef == nil {
		c.diag.AddErrorf(s.AST, c.filePath,
			"struct %s extends unresolved or non-struct type: %s",
			s.Name, s.Extends.RawType)
		return
	}

	parent := s.Extends.StructRef

	// Check for self-extension
	if parent == s {
		c.diag.AddErrorf(s.AST, c.filePath, "struct %s cannot extend itself", s.Name)
		return
	}

	// Check for circular inheritance
	if hasCircularInheritance(s, make(map[*resolution.Struct]bool)) {
		c.diag.AddErrorf(s.AST, c.filePath,
			"circular inheritance detected: struct %s", s.Name)
		return
	}

	// Validate omitted fields exist in parent (checking all inherited fields)
	parentFieldNames := make(map[string]bool)
	for _, f := range parent.UnifiedFields() {
		parentFieldNames[f.Name] = true
	}
	for _, omitted := range s.OmittedFields {
		if !parentFieldNames[omitted] {
			c.diag.AddErrorf(s.AST, c.filePath,
				"cannot omit field %q: not found in parent struct %s",
				omitted, parent.Name)
		}
	}

	// Validate type parameter count if parent is generic
	if len(parent.TypeParams) > 0 {
		requiredParams := 0
		for _, tp := range parent.TypeParams {
			if tp.Default == nil && !tp.Optional {
				requiredParams++
			}
		}
		if len(s.Extends.TypeArgs) < requiredParams {
			c.diag.AddErrorf(s.AST, c.filePath,
				"struct %s extends %s but provides %d type arguments (need at least %d)",
				s.Name, parent.Name, len(s.Extends.TypeArgs), requiredParams)
		}
	}
}

// hasCircularInheritance checks if following the extends chain leads back to the start.
func hasCircularInheritance(s *resolution.Struct, visited map[*resolution.Struct]bool) bool {
	if visited[s] {
		return true
	}
	if s.Extends == nil || s.Extends.StructRef == nil {
		return false
	}
	visited[s] = true
	return hasCircularInheritance(s.Extends.StructRef, visited)
}
