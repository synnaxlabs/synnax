// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer

import (
	"context"
	"maps"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

type analysisCtx struct {
	context.Context
	fileDomains map[string]resolution.Domain
	diag        *diagnostics.Diagnostics
	table       *resolution.Table
	loader      FileLoader
	ast         parser.ISchemaContext
	filePath    string
	namespace   string
}

func Analyze(
	ctx context.Context,
	files []string,
	loader FileLoader,
) (*resolution.Table, *diagnostics.Diagnostics) {
	diag := &diagnostics.Diagnostics{}
	table := resolution.NewTable()

	for _, file := range files {
		importPath := strings.TrimSuffix(file, ".oracle")
		if table.IsImported(importPath) {
			continue
		}
		table.MarkImported(importPath)

		source, filePath, err := loader.Load(file)
		if err != nil {
			d := diagnostics.Errorf(nil, "failed to load file: %v", err)
			d.File = file
			diag.Add(d)
			continue
		}
		ast, parseDiag := parser.Parse(source)
		if parseDiag != nil && !parseDiag.Ok() {
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
	if !diag.Ok() {
		return nil, diag
	}
	detectRecursiveTypes(table)
	validateDomainOmits(table, diag)
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
	if parseDiag != nil && !parseDiag.Ok() {
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
	if !diag.Ok() {
		return nil, diag
	}
	detectRecursiveTypes(table)
	validateDomainOmits(table, diag)
	return table, diag
}

// usedQualifiers collects the package qualifiers the file references: every
// identifier immediately followed by a dot, in any position (type references,
// extends clauses, domain expressions).
func usedQualifiers(ast parser.ISchemaContext) set.Set[string] {
	used := make(set.Set[string])
	stream, ok := ast.GetParser().GetTokenStream().(*antlr.CommonTokenStream)
	if !ok {
		return used
	}
	var defaultChan []antlr.Token
	for _, tok := range stream.GetAllTokens() {
		if tok.GetChannel() == antlr.TokenDefaultChannel {
			defaultChan = append(defaultChan, tok)
		}
	}
	for i := 0; i+1 < len(defaultChan); i++ {
		if defaultChan[i].GetTokenType() == parser.OracleLexerIDENT &&
			defaultChan[i+1].GetTokenType() == parser.OracleLexerDOT {
			used.Add(defaultChan[i].GetText())
		}
	}
	return used
}

func analyze(c *analysisCtx) {
	if c.ast == nil {
		return
	}

	for _, fd := range c.ast.AllFileDomain() {
		de := collectFileDomain(fd)
		if existing, ok := c.fileDomains[de.Name]; ok {
			c.fileDomains[de.Name] = de.Merge(existing)
		} else {
			c.fileDomains[de.Name] = de
		}
	}

	for _, def := range c.ast.AllDefinition() {
		if s := def.StructDef(); s != nil {
			collectStruct(c, s)
		}
		if e := def.EnumDef(); e != nil {
			collectEnum(c, e)
		}
		if td := def.TypeDefDef(); td != nil {
			collectTypeDef(c, td)
		}
		if u := def.UnionDef(); u != nil {
			collectUnion(c, u)
		}
	}

	used := usedQualifiers(c.ast)
	for _, imp := range c.ast.AllImportStmt() {
		path := strings.Trim(imp.STRING_LIT().GetText(), `"`)
		if !used.Contains(filepath.Base(path)) {
			d := diagnostics.Errorf(imp, "unused import %q", path)
			d.File = c.filePath
			c.diag.Add(d)
		}
		if c.table.IsImported(path) {
			continue
		}
		c.table.MarkImported(path)
		source, filePath, err := c.loader.Load(path)
		if err != nil {
			d := diagnostics.Errorf(imp, "failed to import %s: %v", path, err)
			d.File = c.filePath
			c.diag.Add(d)
			continue
		}
		ast, parseDiag := parser.Parse(source)
		if parseDiag != nil && !parseDiag.Ok() {
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

	types := c.table.TypesInNamespace(c.namespace)
	for i := range types {
		typ := &types[i]
		resolveTypeRefs(c, typ)
	}
	validateDeadOutputs(c, types)
	validateFileVersion(c)
	validateVersionArgs(c, types)
	for _, typ := range types {
		for i, t := range c.table.Types {
			if t.QualifiedName == typ.QualifiedName {
				c.table.Types[i] = typ
				break
			}
		}
	}

	finalizeEnumExtensions(c)
	finalizeUnionExtensions(c)

	for _, typ := range c.table.TypesInNamespace(c.namespace) {
		validateExtends(c, typ)
		validateActionExtends(c, typ)
		validateTypeParams(c, typ)
		validateUnion(c, typ)
		validateFieldOverrides(c, typ)
	}

	desugarPartialOverrides(c)
	finalizeActionExtends(c)
	checkOptionalDefaultInvariant(c)
	checkDefaultInvariant(c)
	synthesizeCreateTypes(c)
}

// desugarPartialOverrides rewrites each struct's typeless override fields into
// complete fields, resolving the inherited type, optionality, (when omitted)
// default, and domains from the parent. The field's own domains win on a name
// conflict, matching mergeOverrideField. After this pass a typeless override
// (key?) is indistinguishable from a full restatement (key Key?), so the code
// generators make the same embed-vs-flatten decision and emit identical output.
// Domain removal (-@domain) is intentionally left for the generators to handle,
// since it has no full-restatement equivalent and cannot be expressed through
// embedding.
func desugarPartialOverrides(c *analysisCtx) {
	for i := range c.table.Types {
		typ := c.table.Types[i]
		if typ.Namespace != c.namespace {
			continue
		}
		form, ok := typ.Form.(resolution.StructForm)
		if !ok || len(form.Extends) == 0 {
			continue
		}
		if hasCircularInheritance(typ, c.table, make(set.Set[string])) {
			continue
		}
		parents := resolvedParentFields(c, form)
		fields := make([]resolution.Field, len(form.Fields))
		copy(fields, form.Fields)
		changed := false
		for j := range fields {
			f := &fields[j]
			if f.HasType() {
				continue
			}
			pf, ok := parents[f.Name]
			if !ok {
				continue
			}
			f.Type = pf.Type
			if !f.Optional {
				f.Optional = pf.Optional
			}
			if f.Default == nil {
				f.Default = pf.Default
			}
			if len(pf.Domains) > 0 {
				domains := make(map[string]resolution.Domain, len(pf.Domains)+len(f.Domains))
				maps.Copy(domains, pf.Domains)
				for k, v := range f.Domains {
					if existing, ok := domains[k]; ok {
						domains[k] = v.Merge(existing)
					} else {
						domains[k] = v
					}
				}
				f.Domains = domains
			}
			changed = true
		}
		if !changed {
			continue
		}
		form.Fields = fields
		typ.Form = form
		c.table.Types[i] = typ
	}
}

// resolvedParentFields collects the fields a struct inherits, keyed by name, with
// generic type parameters substituted. The first parent wins on a name conflict,
// matching UnifiedFields.
func resolvedParentFields(c *analysisCtx, form resolution.StructForm) map[string]resolution.Field {
	out := make(map[string]resolution.Field)
	for _, ext := range form.Extends {
		parent, ok := ext.Resolve(c.table)
		if !ok {
			continue
		}
		parentForm, ok := parent.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		typeArgMap := make(map[string]resolution.TypeRef)
		for i, tp := range parentForm.TypeParams {
			if i < len(ext.TypeArgs) {
				typeArgMap[tp.Name] = ext.TypeArgs[i]
			}
		}
		for _, pf := range resolution.UnifiedFields(parent, c.table) {
			if _, exists := out[pf.Name]; exists {
				continue
			}
			pf.Type = resolution.SubstituteTypeRef(pf.Type, typeArgMap)
			out[pf.Name] = pf
		}
	}
	return out
}

// validateFieldOverrides reports partial-override syntax that cannot resolve: a
// field that omits its type, or removes a domain with `-@domain`, must name a
// field inherited from a parent struct to inherit from. Outside an extends
// chain, or naming no parent field, there is nothing to inherit.
func validateFieldOverrides(c *analysisCtx, typ resolution.Type) {
	form, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return
	}
	// Resolving parent fields walks the extends chain; a cycle (already reported
	// by validateExtends) would recur without bound, so skip it here.
	if hasCircularInheritance(typ, c.table, make(set.Set[string])) {
		return
	}
	parentFields := make(set.Set[string])
	for _, extendsRef := range form.Extends {
		parent, ok := extendsRef.Resolve(c.table)
		if !ok {
			continue
		}
		for _, f := range resolution.UnifiedFields(parent, c.table) {
			parentFields.Add(f.Name)
		}
	}
	for _, f := range form.Fields {
		if parentFields.Contains(f.Name) {
			continue
		}
		if !f.HasType() {
			d := diagnostics.Errorf(nil,
				"field %q in struct %s declares no type and overrides no parent field",
				f.Name, typ.Name)
			d.File = c.filePath
			c.diag.Add(d)
		}
		if len(f.OmittedDomains) > 0 {
			d := diagnostics.Errorf(nil,
				"field %q in struct %s removes a domain with -@ but overrides no parent field",
				f.Name, typ.Name)
			d.File = c.filePath
			c.diag.Add(d)
		}
	}
}

// finalizeEnumExtensions expands each extending enum's members to the union of
// the enums it extends plus its own declared members, reporting unresolved
// parents, non-enum parents, mismatched value kinds, and conflicting member
// values. It mutates the table in place so downstream plugins read a fully
// populated member list just like a plain enum's.
func finalizeEnumExtensions(c *analysisCtx) {
	for _, typ := range c.table.TypesInNamespace(c.namespace) {
		form, ok := typ.Form.(resolution.EnumForm)
		if !ok || len(form.Extends) == 0 {
			continue
		}
		merged, isInt, ok := effectiveEnumValues(c, typ, set.New[string]())
		if !ok {
			continue
		}
		form.Values = merged
		form.IsIntEnum = isInt
		typ.Form = form
		for i, t := range c.table.Types {
			if t.QualifiedName == typ.QualifiedName {
				c.table.Types[i] = typ
				break
			}
		}
	}
}

// addTypeDiag records an error diagnostic attributed to the file that
// declares typ.
func addTypeDiag(c *analysisCtx, typ resolution.Type, format string, args ...any) {
	d := diagnostics.Errorf(nil, format, args...)
	d.File = typ.FilePath
	c.diag.Add(d)
}

// effectiveEnumValues returns an enum's fully-resolved members: the union of the
// members of every enum it extends (recursively), followed by its own declared
// members. visited guards against cycles. The returned bool is the int/string
// kind; the final bool is false when the enum or a parent is malformed (a
// diagnostic is added in that case).
func effectiveEnumValues(
	c *analysisCtx, typ resolution.Type, visited set.Set[string],
) ([]resolution.EnumValue, bool, bool) {
	form, ok := typ.Form.(resolution.EnumForm)
	if !ok {
		return nil, false, false
	}
	if visited.Contains(typ.QualifiedName) {
		addTypeDiag(c, typ, "enum %s has a cyclic extends chain", typ.QualifiedName)
		return nil, false, false
	}
	visited.Add(typ.QualifiedName)

	var (
		merged  []resolution.EnumValue
		byName  = make(map[string]resolution.EnumValue)
		isInt   = form.IsIntEnum
		kindSet bool
	)
	add := func(parentName string, v resolution.EnumValue) bool {
		if existing, dup := byName[v.Name]; dup {
			if existing.Value != v.Value {
				addTypeDiag(c, typ, "enum %s inherits member %q with conflicting values from %s",
					typ.QualifiedName, v.Name, parentName)
				return false
			}
			return true
		}
		byName[v.Name] = v
		merged = append(merged, v)
		return true
	}

	for i := range form.Extends {
		parent, ok := c.table.Get(form.Extends[i].Name)
		if !ok {
			addTypeDiag(c, typ, "enum %s extends unknown enum %q", typ.QualifiedName, form.Extends[i].Name)
			return nil, false, false
		}
		if _, ok := parent.Form.(resolution.EnumForm); !ok {
			addTypeDiag(c, typ, "enum %s extends %s, which is not an enum", typ.QualifiedName, parent.QualifiedName)
			return nil, false, false
		}
		// Each branch gets its own copy so visited tracks the ancestor path,
		// not every node seen. A shared set would falsely flag a diamond
		// (two parents extending a common ancestor) as a cyclic chain.
		pVals, pInt, ok := effectiveEnumValues(c, parent, visited.Copy())
		if !ok {
			return nil, false, false
		}
		if kindSet && pInt != isInt {
			addTypeDiag(c, typ, "enum %s extends enums of mixed integer and string kinds", typ.QualifiedName)
			return nil, false, false
		}
		isInt, kindSet = pInt, true
		for _, v := range pVals {
			if !add(parent.QualifiedName, v) {
				return nil, false, false
			}
		}
	}
	if kindSet && len(form.Values) > 0 && form.IsIntEnum != isInt {
		addTypeDiag(c, typ, "enum %s adds members of a different kind than the enums it extends",
			typ.QualifiedName)
		return nil, false, false
	}
	for _, v := range form.Values {
		if !add(typ.QualifiedName, v) {
			return nil, false, false
		}
	}
	return merged, isInt, true
}

// finalizeUnionExtensions resolves each union whose extends targets are other
// unions into a flat variant list: the variants of every extended union
// (recursively), followed by the union's own declared variants. Extends
// targets that are structs keep the shared-base-field semantics and are left
// untouched; mixing the two in one declaration is an error. The table is
// mutated in place so downstream plugins read a plain, fully populated union.
func finalizeUnionExtensions(c *analysisCtx) {
	for _, typ := range c.table.TypesInNamespace(c.namespace) {
		form, ok := typ.Form.(resolution.UnionForm)
		if !ok || len(form.Extends) == 0 {
			continue
		}
		if !hasUnionBases(form, c.table) {
			continue
		}
		merged, ok := effectiveUnionVariants(c, typ, set.New[string]())
		if !ok {
			continue
		}
		form.Variants = merged
		form.Extends = nil
		typ.Form = form
		for i, t := range c.table.Types {
			if t.QualifiedName == typ.QualifiedName {
				c.table.Types[i] = typ
				break
			}
		}
	}
}

// hasUnionBases reports whether any of a union's extends targets resolves to
// another union.
func hasUnionBases(form resolution.UnionForm, table *resolution.Table) bool {
	for _, ref := range form.Extends {
		if base, ok := ref.Resolve(table); ok {
			if _, isUnion := base.Form.(resolution.UnionForm); isUnion {
				return true
			}
		}
	}
	return false
}

// effectiveUnionVariants returns a union's fully-resolved variant list: the
// variants of every union it extends (recursively), followed by its own
// declared variants. visited guards against cycles. The bool is false when the
// union or a parent is malformed (a diagnostic is added in that case).
func effectiveUnionVariants(
	c *analysisCtx, typ resolution.Type, visited set.Set[string],
) ([]resolution.UnionVariant, bool) {
	form, ok := typ.Form.(resolution.UnionForm)
	if !ok {
		return nil, false
	}
	if visited.Contains(typ.QualifiedName) {
		addTypeDiag(c, typ, "union %s has a cyclic extends chain", typ.QualifiedName)
		return nil, false
	}
	visited.Add(typ.QualifiedName)

	var merged []resolution.UnionVariant
	byName := make(map[string]resolution.UnionVariant)
	add := func(parentName string, v resolution.UnionVariant) bool {
		if existing, dup := byName[v.Name]; dup {
			if existing.Type.Name != v.Type.Name {
				addTypeDiag(c, typ,
					"union %s inherits variant %q with conflicting payload types from %s (%s vs %s)",
					typ.QualifiedName, v.Name, parentName, existing.Type.Name, v.Type.Name)
				return false
			}
			return true
		}
		byName[v.Name] = v
		merged = append(merged, v)
		return true
	}

	for i := range form.Extends {
		parent, ok := form.Extends[i].Resolve(c.table)
		if !ok {
			addTypeDiag(c, typ, "union %s extends unresolved type: %s",
				typ.QualifiedName, form.Extends[i].Name)
			return nil, false
		}
		pform, isUnion := parent.Form.(resolution.UnionForm)
		if !isUnion {
			addTypeDiag(c, typ,
				"union %s cannot mix struct and union bases in extends; %s is not a union",
				typ.QualifiedName, parent.QualifiedName)
			return nil, false
		}
		if pform.Discriminator != form.Discriminator {
			addTypeDiag(c, typ,
				"union %s extends union %s with a different discriminator (%q vs %q)",
				typ.QualifiedName, parent.QualifiedName, pform.Discriminator, form.Discriminator)
			return nil, false
		}
		pVars := pform.Variants
		if len(pform.Extends) > 0 {
			if !hasUnionBases(pform, c.table) {
				addTypeDiag(c, typ,
					"union %s cannot extend union %s, which extends base structs",
					typ.QualifiedName, parent.QualifiedName)
				return nil, false
			}
			// Each branch gets its own copy so visited tracks the ancestor
			// path, not every node seen, matching effectiveEnumValues.
			if pVars, ok = effectiveUnionVariants(c, parent, visited.Copy()); !ok {
				return nil, false
			}
		}
		for _, v := range pVars {
			if !add(parent.QualifiedName, v) {
				return nil, false
			}
		}
	}
	for _, v := range form.Variants {
		if !add(typ.QualifiedName, v) {
			return nil, false
		}
	}
	return merged, true
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
	if _, exists := c.table.Get(qname); exists {
		d := diagnostics.Errorf(def, "duplicate type definition: %s", qname)
		d.File = c.filePath
		c.diag.Add(d)
		return
	}

	form := resolution.StructForm{
		TypeParams: collectTypeParams(def.TypeParams()),
	}

	if def.EXTENDS() != nil {
		if typeRefList := def.TypeRefList(); typeRefList != nil {
			for _, tr := range typeRefList.AllTypeRef() {
				ext := collectTypeRef(tr, nil)
				form.Extends = append(form.Extends, ext)
			}
		}
	}

	domains := make(map[string]resolution.Domain)
	maps.Copy(domains, c.fileDomains)

	if body := def.StructBody(); body != nil {
		for _, f := range body.AllFieldDef() {
			field := collectField(c, f, form.TypeParams, &form.HasKeyDomain)
			form.Fields = append(form.Fields, field)
		}
		for _, fo := range body.AllFieldOmit() {
			form.OmittedFields = append(form.OmittedFields, fo.IDENT().GetText())
		}
		for _, a := range body.AllActionDef() {
			action := collectAction(c, a, form.TypeParams)
			form.Actions = append(form.Actions, action)
		}
		for _, d := range body.AllDomain() {
			de := collectDomain(d)
			if existing, ok := domains[de.Name]; ok {
				domains[de.Name] = de.Merge(existing)
			} else {
				domains[de.Name] = de
			}
		}
	}

	lo.Must0(c.table.Add(resolution.Type{
		Name:          name,
		Namespace:     c.namespace,
		QualifiedName: qname,
		FilePath:      c.filePath,
		Form:          form,
		Domains:       domains,
		AST:           def,
	}))
}

// collectAction translates a parsed action block into a resolution.Action,
// reusing collectField for payload fields so action-field semantics match
// struct-field semantics (optionality, validation, type refs).
func collectAction(
	c *analysisCtx,
	def parser.IActionDefContext,
	typeParams []resolution.TypeParam,
) resolution.Action {
	action := resolution.Action{
		Name:    def.IDENT().GetText(),
		Domains: make(map[string]resolution.Domain),
		AST:     def,
	}
	if def.EXTENDS() != nil {
		if typeRefList := def.TypeRefList(); typeRefList != nil {
			for _, tr := range typeRefList.AllTypeRef() {
				action.Extends = append(action.Extends, collectTypeRef(tr, typeParams))
			}
		}
	}
	body := def.ActionBody()
	if body == nil {
		return action
	}
	hasKeyDomain := false
	for _, f := range body.AllFieldDef() {
		field := collectField(c, f, typeParams, &hasKeyDomain)
		if !field.HasType() {
			d := diagnostics.Errorf(f, "action %s field %q must declare a type", action.Name, field.Name)
			d.File = c.filePath
			c.diag.Add(d)
		}
		action.Fields = append(action.Fields, field)
	}
	for _, d := range body.AllDomain() {
		de := collectDomain(d)
		if existing, ok := action.Domains[de.Name]; ok {
			action.Domains[de.Name] = de.Merge(existing)
		} else {
			action.Domains[de.Name] = de
		}
	}
	return action
}

func collectStructAlias(c *analysisCtx, def *parser.StructAliasContext) {
	name := def.IDENT().GetText()
	qname := c.namespace + "." + name

	if _, exists := c.table.Get(qname); exists {
		d := diagnostics.Errorf(def, "duplicate type definition: %s", qname)
		d.File = c.filePath
		c.diag.Add(d)
		return
	}

	typeParams := collectTypeParams(def.TypeParams())
	tr := def.TypeRef()
	target := collectTypeRef(tr, typeParams)

	if normalCtx, ok := tr.(*parser.TypeRefNormalContext); ok {
		if arrMod := normalCtx.ArrayModifier(); arrMod != nil {
			var arraySize *int64
			if intLit := arrMod.INT_LIT(); intLit != nil {
				size, _ := strconv.ParseInt(intLit.GetText(), 10, 64)
				arraySize = &size
			}
			target = resolution.TypeRef{
				Name:      "Array",
				TypeArgs:  []resolution.TypeRef{target},
				ArraySize: arraySize,
			}
		}
	}

	domains := make(map[string]resolution.Domain)
	maps.Copy(domains, c.fileDomains)
	if body := def.AliasBody(); body != nil {
		for _, d := range body.AllDomain() {
			de := collectDomain(d)
			if existing, ok := domains[de.Name]; ok {
				domains[de.Name] = de.Merge(existing)
			} else {
				domains[de.Name] = de
			}
		}
	}

	lo.Must0(c.table.Add(resolution.Type{
		Name:          name,
		Namespace:     c.namespace,
		QualifiedName: qname,
		FilePath:      c.filePath,
		Form: resolution.AliasForm{
			Target:     target,
			TypeParams: typeParams,
		},
		Domains: domains,
		AST:     def,
	}))
}

func collectTypeParams(params parser.ITypeParamsContext) []resolution.TypeParam {
	if params == nil {
		return nil
	}
	var result []resolution.TypeParam
	for _, p := range params.AllTypeParam() {
		tp := resolution.TypeParam{Name: p.IDENT().GetText()}
		tp.Optional = p.QUESTION() != nil
		typeRefs := p.AllTypeRef()
		hasExtends := p.EXTENDS() != nil
		hasEquals := p.EQUALS() != nil
		if hasExtends && len(typeRefs) > 0 {
			ref := collectTypeRef(typeRefs[0], nil)
			tp.Constraint = &ref
		}
		if hasEquals {
			idx := 0
			if hasExtends {
				idx = 1
			}
			if idx < len(typeRefs) {
				ref := collectTypeRef(typeRefs[idx], nil)
				tp.Default = &ref
			}
		}
		result = append(result, tp)
	}
	return result
}

func collectTypeRef(tr parser.ITypeRefContext, typeParams []resolution.TypeParam) resolution.TypeRef {
	if mapCtx, ok := tr.(*parser.TypeRefMapContext); ok {
		return collectMapTypeRef(mapCtx, typeParams)
	}

	normalCtx := tr.(*parser.TypeRefNormalContext)
	rawType := extractTypeNormal(normalCtx)

	ref := resolution.TypeRef{Name: rawType}

	for i := range typeParams {
		if typeParams[i].Name == rawType {
			ref.TypeParam = &typeParams[i]
			ref.Name = ""
			break
		}
	}

	if args := normalCtx.TypeArgs(); args != nil {
		for _, arg := range args.AllTypeRef() {
			ref.TypeArgs = append(ref.TypeArgs, collectTypeRef(arg, typeParams))
		}
	}

	return ref
}

func collectMapTypeRef(mapCtx *parser.TypeRefMapContext, typeParams []resolution.TypeParam) resolution.TypeRef {
	mt := mapCtx.MapType()
	typeRefs := mt.AllTypeRef()

	ref := resolution.TypeRef{Name: "Map"}
	if len(typeRefs) >= 2 {
		keyRef := collectTypeRef(typeRefs[0], typeParams)
		valueRef := collectTypeRef(typeRefs[1], typeParams)
		ref.TypeArgs = []resolution.TypeRef{keyRef, valueRef}
	}
	return ref
}

func collectField(c *analysisCtx, def parser.IFieldDefContext, typeParams []resolution.TypeParam, hasKeyDomain *bool) resolution.Field {
	field := resolution.Field{
		Name:    def.IDENT().GetText(),
		Domains: make(map[string]resolution.Domain),
		AST:     def,
	}

	// A field may omit its type to partially override an inherited field, in
	// which case the type and optionality are resolved from the parent later.
	if tr := def.TypeRef(); tr != nil {
		normalCtx, isNormal := tr.(*parser.TypeRefNormalContext)
		mapCtx, isMap := tr.(*parser.TypeRefMapContext)
		isArray := false
		var arraySize *int64

		if isNormal {
			field.Optional = extractTypeModifiersNormal(normalCtx)
			if arrMod := normalCtx.ArrayModifier(); arrMod != nil {
				isArray = true
				if intLit := arrMod.INT_LIT(); intLit != nil {
					size, _ := strconv.ParseInt(intLit.GetText(), 10, 64)
					arraySize = &size
				}
			}
		} else if isMap {
			field.Optional = extractTypeModifiersMap(mapCtx)
		}

		typeRef := collectTypeRef(tr, typeParams)
		if isArray {
			typeRef = resolution.TypeRef{
				Name:      "Array",
				TypeArgs:  []resolution.TypeRef{typeRef},
				ArraySize: arraySize,
			}
		}
		field.Type = typeRef
	} else if mods := def.TypeModifiers(); mods != nil {
		// Typeless override that overrides only nullability (key?); the type is
		// inherited from the parent during override resolution.
		field.Optional = modifiersFrom(mods)
	}

	if fd := def.FieldDefault(); fd != nil {
		dv := collectFieldDefault(fd)
		field.Default = &dv
	}

	for _, inl := range def.AllInlineDomain() {
		de := collectInlineDomain(inl)
		field.Domains[de.Name] = de
		if de.Name == "key" {
			*hasKeyDomain = true
		}
	}
	for _, om := range def.AllDomainOmit() {
		field.OmittedDomains = append(field.OmittedDomains, om.IDENT().GetText())
	}

	if fb := def.FieldBody(); fb != nil {
		for _, d := range fb.AllDomain() {
			de := collectDomain(d)
			field.Domains[de.Name] = de
			if de.Name == "key" {
				*hasKeyDomain = true
			}
		}
		for _, om := range fb.AllDomainOmit() {
			field.OmittedDomains = append(field.OmittedDomains, om.IDENT().GetText())
		}
	}
	return field
}

func collectFileDomain(fd parser.IFileDomainContext) resolution.Domain {
	entry := resolution.Domain{AST: fd, Name: fd.IDENT().GetText()}
	if content := fd.DomainContent(); content != nil {
		collectDomainContent(&entry, content)
	}
	return entry
}

func collectDomain(def parser.IDomainContext) resolution.Domain {
	entry := resolution.Domain{AST: def, Name: def.IDENT().GetText()}
	if content := def.DomainContent(); content != nil {
		collectDomainContent(&entry, content)
	}
	return entry
}

func collectInlineDomain(def parser.IInlineDomainContext) resolution.Domain {
	entry := resolution.Domain{AST: def, Name: def.IDENT().GetText()}
	if content := def.DomainContent(); content != nil {
		collectDomainContent(&entry, content)
	}
	return entry
}

func collectDomainContent(entry *resolution.Domain, content parser.IDomainContentContext) {
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
	if e := content.Expression(); e != nil {
		expr := resolution.Expression{AST: e, Name: e.IDENT().GetText()}
		for _, v := range e.AllExpressionValue() {
			expr.Values = append(expr.Values, collectValue(v))
		}
		entry.Expressions = append(entry.Expressions, expr)
	}
}

func collectFieldDefault(fd parser.IFieldDefaultContext) resolution.ExpressionValue {
	if ev := fd.ExpressionValue(); ev != nil {
		return collectValue(ev)
	}
	if arr := fd.ArrayDefault(); arr != nil {
		return collectArrayDefault(arr)
	}
	if st := fd.StructDefault(); st != nil {
		return collectStructDefault(st)
	}
	return resolution.ExpressionValue{}
}

// collectDefaultValue collects a default value in a nested position (an array
// element or a struct field), recursing into arrays and structs.
func collectDefaultValue(dv parser.IDefaultValueContext) resolution.ExpressionValue {
	if ev := dv.ExpressionValue(); ev != nil {
		return collectValue(ev)
	}
	if arr := dv.ArrayDefault(); arr != nil {
		return collectArrayDefault(arr)
	}
	if st := dv.StructDefault(); st != nil {
		return collectStructDefault(st)
	}
	return resolution.ExpressionValue{}
}

func collectArrayDefault(arr parser.IArrayDefaultContext) resolution.ExpressionValue {
	out := resolution.ExpressionValue{Kind: resolution.ValueKindArray}
	for _, el := range arr.AllDefaultValue() {
		out.Elements = append(out.Elements, collectDefaultValue(el))
	}
	return out
}

func collectStructDefault(st parser.IStructDefaultContext) resolution.ExpressionValue {
	out := resolution.ExpressionValue{Kind: resolution.ValueKindStruct}
	for _, f := range st.AllStructFieldDefault() {
		out.Fields = append(out.Fields, resolution.StructFieldValue{
			Name:  f.IDENT().GetText(),
			Value: collectDefaultValue(f.DefaultValue()),
		})
	}
	return out
}

func collectValue(v parser.IExpressionValueContext) resolution.ExpressionValue {
	if ts := v.TRIPLE_STRING_LIT(); ts != nil {
		t := ts.GetText()
		content := t[3 : len(t)-3]
		dedented := dedent(content)
		return resolution.ExpressionValue{
			Kind:        resolution.ValueKindString,
			StringValue: dedented,
		}
	}
	if s := v.STRING_LIT(); s != nil {
		t := s.GetText()
		unquoted, err := strconv.Unquote(t)
		if err != nil {
			unquoted = t[1 : len(t)-1]
		}
		return resolution.ExpressionValue{
			Kind:        resolution.ValueKindString,
			StringValue: unquoted,
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

func collectUnion(c *analysisCtx, def parser.IUnionDefContext) {
	idents := def.AllIDENT()
	if len(idents) < 3 {
		return
	}
	name := idents[0].GetText()
	if idents[1].GetText() != "on" {
		d := diagnostics.Errorf(def,
			"union %s: expected 'on' before the discriminator field, got %q",
			name, idents[1].GetText())
		d.File = c.filePath
		c.diag.Add(d)
		return
	}
	discriminator := idents[2].GetText()
	qname := c.namespace + "." + name
	if _, exists := c.table.Get(qname); exists {
		d := diagnostics.Errorf(def, "duplicate type definition: %s", qname)
		d.File = c.filePath
		c.diag.Add(d)
		return
	}

	form := resolution.UnionForm{Discriminator: discriminator}

	if def.EXTENDS() != nil {
		if typeRefList := def.TypeRefList(); typeRefList != nil {
			for _, tr := range typeRefList.AllTypeRef() {
				form.Extends = append(form.Extends, collectTypeRef(tr, nil))
			}
		}
	}

	domains := make(map[string]resolution.Domain)
	maps.Copy(domains, c.fileDomains)

	if body := def.UnionBody(); body != nil {
		for _, v := range body.AllUnionVariant() {
			form.Variants = append(form.Variants, collectUnionVariant(c, name, v))
		}
		for _, d := range body.AllDomain() {
			de := collectDomain(d)
			if existing, ok := domains[de.Name]; ok {
				domains[de.Name] = de.Merge(existing)
			} else {
				domains[de.Name] = de
			}
		}
	}

	lo.Must0(c.table.Add(resolution.Type{
		Name:          name,
		Namespace:     c.namespace,
		QualifiedName: qname,
		FilePath:      c.filePath,
		Form:          form,
		Domains:       domains,
		AST:           def,
	}))
}

func collectUnionVariant(
	c *analysisCtx, unionName string, def parser.IUnionVariantContext,
) resolution.UnionVariant {
	switch v := def.(type) {
	case *parser.NamedVariantContext:
		variant := resolution.UnionVariant{
			Name:    v.VariantName().GetText(),
			Type:    collectTypeRef(v.TypeRef(), nil),
			Domains: make(map[string]resolution.Domain),
			AST:     def,
		}
		if body := v.UnionVariantBody(); body != nil {
			for _, d := range body.AllDomain() {
				de := collectDomain(d)
				variant.Domains[de.Name] = de
			}
		}
		return variant
	case *parser.InlineVariantContext:
		return collectInlineVariant(c, unionName, v)
	default:
		return resolution.UnionVariant{}
	}
}

// collectInlineVariant desugars an inline variant body into a Synthetic
// struct type registered in the table, so the variant resolves and validates
// like a named payload while generators flatten its fields into the variant
// member instead of emitting a standalone type.
func collectInlineVariant(
	c *analysisCtx, unionName string, def *parser.InlineVariantContext,
) resolution.UnionVariant {
	variantName := def.VariantName().GetText()
	variant := resolution.UnionVariant{
		Name:    variantName,
		Domains: make(map[string]resolution.Domain),
		AST:     def,
		Inline:  true,
	}

	form := resolution.StructForm{}
	if def.EXTENDS() != nil {
		if typeRefList := def.TypeRefList(); typeRefList != nil {
			for _, tr := range typeRefList.AllTypeRef() {
				form.Extends = append(form.Extends, collectTypeRef(tr, nil))
			}
		}
	}
	if body := def.StructBody(); body != nil {
		for _, f := range body.AllFieldDef() {
			form.Fields = append(form.Fields, collectField(c, f, nil, &form.HasKeyDomain))
		}
		for _, fo := range body.AllFieldOmit() {
			d := diagnostics.Errorf(fo,
				"union %s variant %q: field omissions are not supported in inline variant bodies",
				unionName, variantName)
			d.File = c.filePath
			c.diag.Add(d)
		}
		for _, a := range body.AllActionDef() {
			d := diagnostics.Errorf(a,
				"union %s variant %q: actions are not supported in inline variant bodies",
				unionName, variantName)
			d.File = c.filePath
			c.diag.Add(d)
		}
		for _, d := range body.AllDomain() {
			de := collectDomain(d)
			variant.Domains[de.Name] = de
		}
	}

	name := unionName + pascalIdent(variantName) + "Payload"
	qname := c.namespace + "." + name
	if _, exists := c.table.Get(qname); exists {
		d := diagnostics.Errorf(def,
			"union %s variant %q: inline payload name %s collides with an existing type",
			unionName, variantName, qname)
		d.File = c.filePath
		c.diag.Add(d)
		return variant
	}
	domains := make(map[string]resolution.Domain)
	maps.Copy(domains, c.fileDomains)
	lo.Must0(c.table.Add(resolution.Type{
		Name:          name,
		Namespace:     c.namespace,
		QualifiedName: qname,
		FilePath:      c.filePath,
		Form:          form,
		Domains:       domains,
		AST:           def,
		Synthetic:     true,
	}))
	variant.Type = resolution.TypeRef{Name: name}
	return variant
}

// pascalIdent converts a snake_case variant name to PascalCase for the
// synthesized payload type name.
func pascalIdent(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

func collectEnum(c *analysisCtx, def parser.IEnumDefContext) {
	name := def.IDENT().GetText()
	qname := c.namespace + "." + name
	if _, exists := c.table.Get(qname); exists {
		d := diagnostics.Errorf(def, "duplicate type definition: %s", qname)
		d.File = c.filePath
		c.diag.Add(d)
		return
	}

	form := resolution.EnumForm{}
	if def.EXTENDS() != nil {
		if typeRefList := def.TypeRefList(); typeRefList != nil {
			for _, tr := range typeRefList.AllTypeRef() {
				form.Extends = append(form.Extends, collectTypeRef(tr, nil))
			}
		}
	}
	domains := make(map[string]resolution.Domain)
	maps.Copy(domains, c.fileDomains)

	if body := def.EnumBody(); body != nil {
		vals := body.AllEnumValue()
		if len(vals) > 0 {
			form.IsIntEnum = vals[0].INT_LIT() != nil
		}
		for _, v := range vals {
			ev := resolution.EnumValue{
				Name:    v.IDENT().GetText(),
				Domains: make(map[string]resolution.Domain),
			}
			if i := v.INT_LIT(); i != nil {
				n, _ := strconv.ParseInt(i.GetText(), 10, 64)
				ev.Value = n
			} else if s := v.STRING_LIT(); s != nil {
				t := s.GetText()
				unquoted, err := strconv.Unquote(t)
				if err != nil {
					unquoted = t[1 : len(t)-1]
				}
				ev.Value = unquoted
			}
			if body := v.EnumValueBody(); body != nil {
				for _, d := range body.AllDomain() {
					de := collectDomain(d)
					ev.Domains[de.Name] = de
				}
			}
			form.Values = append(form.Values, ev)
		}
		for _, d := range body.AllDomain() {
			de := collectDomain(d)
			if existing, ok := domains[de.Name]; ok {
				domains[de.Name] = de.Merge(existing)
			} else {
				domains[de.Name] = de
			}
		}
	}

	lo.Must0(c.table.Add(resolution.Type{
		Name:          name,
		Namespace:     c.namespace,
		QualifiedName: qname,
		FilePath:      c.filePath,
		Form:          form,
		Domains:       domains,
		AST:           def,
	}))
}

func collectTypeDef(c *analysisCtx, def parser.ITypeDefDefContext) {
	name := def.IDENT().GetText()
	qname := c.namespace + "." + name
	if _, exists := c.table.Get(qname); exists {
		d := diagnostics.Errorf(def, "duplicate type definition: %s", qname)
		d.File = c.filePath
		c.diag.Add(d)
		return
	}

	typeParams := collectTypeParams(def.TypeParams())
	tr := def.TypeRef()
	base := collectTypeRef(tr, typeParams)

	if normalCtx, ok := tr.(*parser.TypeRefNormalContext); ok {
		if arrMod := normalCtx.ArrayModifier(); arrMod != nil {
			var arraySize *int64
			if intLit := arrMod.INT_LIT(); intLit != nil {
				size, _ := strconv.ParseInt(intLit.GetText(), 10, 64)
				arraySize = &size
			}
			base = resolution.TypeRef{
				Name:      "Array",
				TypeArgs:  []resolution.TypeRef{base},
				ArraySize: arraySize,
			}
		}
	}

	domains := make(map[string]resolution.Domain)
	maps.Copy(domains, c.fileDomains)

	if body := def.TypeDefBody(); body != nil {
		for _, d := range body.AllDomain() {
			de := collectDomain(d)
			if existing, ok := domains[de.Name]; ok {
				domains[de.Name] = de.Merge(existing)
			} else {
				domains[de.Name] = de
			}
		}
	}

	lo.Must0(c.table.Add(resolution.Type{
		Name:          name,
		Namespace:     c.namespace,
		QualifiedName: qname,
		FilePath:      c.filePath,
		Form:          resolution.DistinctForm{Base: base, TypeParams: typeParams},
		Domains:       domains,
		AST:           def,
	}))
}

func resolveTypeRefs(c *analysisCtx, typ *resolution.Type) {
	switch form := typ.Form.(type) {
	case resolution.StructForm:
		for i := range form.Fields {
			resolveTypeRef(c, typ, &form.Fields[i].Type)
		}
		for i := range form.Actions {
			for j := range form.Actions[i].Fields {
				resolveTypeRef(c, typ, &form.Actions[i].Fields[j].Type)
			}
			for j := range form.Actions[i].Extends {
				resolveTypeRef(c, typ, &form.Actions[i].Extends[j])
			}
		}
		for i := range form.TypeParams {
			if form.TypeParams[i].Constraint != nil {
				resolveTypeRef(c, typ, form.TypeParams[i].Constraint)
			}
			if form.TypeParams[i].Default != nil {
				resolveTypeRef(c, typ, form.TypeParams[i].Default)
			}
		}
		for i := range form.Extends {
			resolveTypeRef(c, typ, &form.Extends[i])
		}
		typ.Form = form
	case resolution.AliasForm:
		resolveTypeRef(c, typ, &form.Target)
		typ.Form = form
	case resolution.DistinctForm:
		resolveTypeRef(c, typ, &form.Base)
		for i := range form.TypeParams {
			if form.TypeParams[i].Constraint != nil {
				resolveTypeRef(c, typ, form.TypeParams[i].Constraint)
			}
			if form.TypeParams[i].Default != nil {
				resolveTypeRef(c, typ, form.TypeParams[i].Default)
			}
		}
		typ.Form = form
	case resolution.UnionForm:
		for i := range form.Extends {
			resolveTypeRef(c, typ, &form.Extends[i])
		}
		for i := range form.Variants {
			resolveTypeRef(c, typ, &form.Variants[i].Type)
		}
		typ.Form = form
	case resolution.EnumForm:
		if len(form.Extends) > 0 {
			for i := range form.Extends {
				resolveTypeRef(c, typ, &form.Extends[i])
			}
			typ.Form = form
		}
	}
}

func resolveTypeRef(c *analysisCtx, currentType *resolution.Type, ref *resolution.TypeRef) {
	if ref.TypeParam != nil {
		return
	}

	parts := strings.Split(ref.Name, ".")
	ns, name := c.namespace, parts[0]
	if len(parts) == 2 {
		ns, name = parts[0], parts[1]
	}

	if currentType != nil && len(parts) == 1 {
		switch form := currentType.Form.(type) {
		case resolution.StructForm:
			for i := range form.TypeParams {
				if form.TypeParams[i].Name == name {
					ref.TypeParam = &form.TypeParams[i]
					ref.Name = ""
					return
				}
			}
		case resolution.AliasForm:
			for i := range form.TypeParams {
				if form.TypeParams[i].Name == name {
					ref.TypeParam = &form.TypeParams[i]
					ref.Name = ""
					return
				}
			}
		case resolution.DistinctForm:
			for i := range form.TypeParams {
				if form.TypeParams[i].Name == name {
					ref.TypeParam = &form.TypeParams[i]
					ref.Name = ""
					return
				}
			}
		}
	}

	if (resolution.IsPrimitive(name) || resolution.IsConstraint(name)) && len(parts) == 1 {
		ref.Name = name
	} else if typ, ok := c.table.Lookup(ns, name); ok {
		ref.Name = typ.QualifiedName
	} else {
		d := diagnostics.Warningf(nil, "unresolved type: %s", ref.Name)
		d.File = c.filePath
		c.diag.Add(d)
	}

	for i := range ref.TypeArgs {
		resolveTypeRef(c, currentType, &ref.TypeArgs[i])
	}
}

func extractTypeNormal(tr *parser.TypeRefNormalContext) string {
	ids := tr.QualifiedIdent().AllIDENT()
	if len(ids) == 2 {
		return ids[0].GetText() + "." + ids[1].GetText()
	}
	return ids[0].GetText()
}

// modifiersFrom reports whether an optionality modifier context marks the field
// optional. A trailing `?` makes the field optional; `??` is a grammar error
// and is rejected by the parser before this function is reached.
func modifiersFrom(mods parser.ITypeModifiersContext) (optional bool) {
	if mods == nil {
		return false
	}
	return mods.QUESTION() != nil
}

func extractTypeModifiersNormal(tr *parser.TypeRefNormalContext) (optional bool) {
	return modifiersFrom(tr.TypeModifiers())
}

func extractTypeModifiersMap(tr *parser.TypeRefMapContext) (optional bool) {
	return modifiersFrom(tr.TypeModifiers())
}

func detectRecursiveTypes(table *resolution.Table) {
	for i := range table.Types {
		if form, ok := table.Types[i].Form.(resolution.StructForm); ok {
			if isRecursive(&table.Types[i], table) {
				form.IsRecursive = true
				table.Types[i].Form = form
			}
		}
	}
}

func isRecursive(typ *resolution.Type, table *resolution.Table) bool {
	form, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return false
	}
	for _, field := range form.Fields {
		if resolution.RefersTo(field.Type, typ.QualifiedName, table) {
			return true
		}
	}
	return false
}

func validateExtends(c *analysisCtx, typ resolution.Type) {
	form, ok := typ.Form.(resolution.StructForm)
	if !ok || len(form.Extends) == 0 {
		return
	}

	for i, extendsRef := range form.Extends {
		parent, ok := extendsRef.Resolve(c.table)
		if !ok {
			d := diagnostics.Errorf(nil,
				"struct %s extends unresolved type at position %d: %s",
				typ.Name, i+1, extendsRef.Name)
			d.File = c.filePath
			c.diag.Add(d)
			continue
		}
		parentForm, ok := parent.Form.(resolution.StructForm)
		if !ok {
			d := diagnostics.Errorf(nil,
				"struct %s extends non-struct type at position %d: %s",
				typ.Name, i+1, parent.Name)
			d.File = c.filePath
			c.diag.Add(d)
			continue
		}
		if parent.QualifiedName == typ.QualifiedName {
			d := diagnostics.Errorf(nil, "struct %s cannot extend itself", typ.Name)
			d.File = c.filePath
			c.diag.Add(d)
			continue
		}
		if len(parentForm.TypeParams) > 0 {
			requiredParams := 0
			for _, tp := range parentForm.TypeParams {
				if tp.Default == nil && !tp.Optional {
					requiredParams++
				}
			}
			if len(extendsRef.TypeArgs) < requiredParams {
				d := diagnostics.Errorf(nil,
					"struct %s extends %s but provides %d type arguments (need at least %d)",
					typ.Name, parent.Name, len(extendsRef.TypeArgs), requiredParams)
				d.File = c.filePath
				c.diag.Add(d)
			}
		}
	}

	if hasCircularInheritance(typ, c.table, make(set.Set[string])) {
		d := diagnostics.Errorf(nil, "circular inheritance detected: struct %s", typ.Name)
		d.File = c.filePath
		c.diag.Add(d)
		return
	}

	allParentFields := make(set.Set[string])
	for _, extendsRef := range form.Extends {
		parent, ok := extendsRef.Resolve(c.table)
		if !ok {
			continue
		}
		for _, f := range resolution.UnifiedFields(parent, c.table) {
			allParentFields.Add(f.Name)
		}
	}
	for _, omitted := range form.OmittedFields {
		if !allParentFields.Contains(omitted) {
			d := diagnostics.Errorf(nil,
				"cannot omit field %q: not found in any parent struct",
				omitted)
			d.File = c.filePath
			c.diag.Add(d)
		}
	}
}

// validateActionExtends checks that every action's extends clause names a
// struct type, mirroring validateExtends. A generic parent must be supplied
// with enough type arguments to bind its required parameters.
func validateActionExtends(c *analysisCtx, typ resolution.Type) {
	form, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return
	}
	for _, action := range form.Actions {
		for i, ext := range action.Extends {
			parent, ok := ext.Resolve(c.table)
			if !ok {
				d := diagnostics.Errorf(nil,
					"action %s extends unresolved type at position %d: %s",
					action.Name, i+1, ext.Name)
				d.File = c.filePath
				c.diag.Add(d)
				continue
			}
			parentForm, ok := parent.Form.(resolution.StructForm)
			if !ok {
				d := diagnostics.Errorf(nil,
					"action %s extends non-struct type at position %d: %s",
					action.Name, i+1, parent.Name)
				d.File = c.filePath
				c.diag.Add(d)
				continue
			}
			requiredParams := 0
			for _, tp := range parentForm.TypeParams {
				if tp.Default == nil && !tp.Optional {
					requiredParams++
				}
			}
			if len(ext.TypeArgs) < requiredParams {
				d := diagnostics.Errorf(nil,
					"action %s extends %s but provides %d type arguments (need at least %d)",
					action.Name, parent.Name, len(ext.TypeArgs), requiredParams)
				d.File = c.filePath
				c.diag.Add(d)
			}
		}
	}
}

// finalizeActionExtends flattens each action's extends clause, prepending the
// inherited struct fields to the action's own fields. After this pass
// Action.Fields holds the unified list, so code generators consume it unchanged.
func finalizeActionExtends(c *analysisCtx) {
	for i := range c.table.Types {
		typ := c.table.Types[i]
		if typ.Namespace != c.namespace {
			continue
		}
		form, ok := typ.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		changed := false
		for ai := range form.Actions {
			if len(form.Actions[ai].Extends) == 0 {
				continue
			}
			form.Actions[ai].Fields = unifyActionFields(c, form.Actions[ai])
			changed = true
		}
		if changed {
			typ.Form = form
			c.table.Types[i] = typ
		}
	}
}

// unifyActionFields returns the action's payload fields with inherited struct
// fields prepended. Parents are walked left-to-right (first wins on a name
// conflict between parents) and an action's own field overrides any inherited
// field of the same name, keeping its declared position.
func unifyActionFields(c *analysisCtx, action resolution.Action) []resolution.Field {
	own := make(set.Set[string])
	for _, f := range action.Fields {
		own.Add(f.Name)
	}
	seen := make(set.Set[string])
	var inherited []resolution.Field
	for _, ext := range action.Extends {
		parent, ok := ext.Resolve(c.table)
		if !ok {
			continue
		}
		parentForm, ok := parent.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		typeArgMap := make(map[string]resolution.TypeRef)
		for j, tp := range parentForm.TypeParams {
			if j < len(ext.TypeArgs) {
				typeArgMap[tp.Name] = ext.TypeArgs[j]
			}
		}
		for _, pf := range resolution.UnifiedFields(parent, c.table) {
			if own.Contains(pf.Name) || seen.Contains(pf.Name) {
				continue
			}
			seen.Add(pf.Name)
			sub := pf
			sub.Type = resolution.SubstituteTypeRef(pf.Type, typeArgMap)
			inherited = append(inherited, sub)
		}
	}
	if len(inherited) == 0 {
		return action.Fields
	}
	return append(inherited, action.Fields...)
}

func typeParamsOf(form resolution.TypeForm) []resolution.TypeParam {
	switch f := form.(type) {
	case resolution.StructForm:
		return f.TypeParams
	case resolution.AliasForm:
		return f.TypeParams
	case resolution.DistinctForm:
		return f.TypeParams
	}
	return nil
}

func validateTypeParams(c *analysisCtx, typ resolution.Type) {
	for _, tp := range typeParamsOf(typ.Form) {
		if tp.Constraint == nil {
			continue
		}
		if tp.Constraint.Name != "numeric" {
			continue
		}
		if tp.Default == nil {
			d := diagnostics.Errorf(nil,
				"type parameter %s of %s constrained by 'numeric' requires a default (e.g. = float64); the constraint cannot be expressed concretely in Go, C++, Python, or Proto",
				tp.Name, typ.Name)
			d.File = c.filePath
			c.diag.Add(d)
			continue
		}
		if !resolution.IsNumberPrimitive(tp.Default.Name) {
			d := diagnostics.Errorf(nil,
				"type parameter %s of %s constrained by 'numeric' has non-numeric default %q; default must be a number primitive (int*, uint*, float32, float64)",
				tp.Name, typ.Name, tp.Default.Name)
			d.File = c.filePath
			c.diag.Add(d)
		}
	}
}

// validateUnion checks the structural constraints on a discriminated union:
//
//   - At least one variant is declared.
//   - Variant names (the JSON discriminator string values) are unique.
//   - Each Extends target resolves to a struct type.
//   - Each variant references a struct type.
//   - Neither the base structs nor the variant structs redeclare the
//     discriminator field; the union declaration owns it exclusively.
//
// Generic variant/base types are permitted but not yet type-checked: an open
// question for a follow-up that introduces unions parameterized by a type
// argument applied to every variant.
func validateUnion(c *analysisCtx, typ resolution.Type) {
	form, ok := typ.Form.(resolution.UnionForm)
	if !ok {
		return
	}

	if len(form.Variants) == 0 {
		d := diagnostics.Errorf(nil, "union %s has no variants", typ.Name)
		d.File = c.filePath
		c.diag.Add(d)
		return
	}

	seenValues := set.New[string]()
	for _, variant := range form.Variants {
		if variant.Name == form.Discriminator {
			d := diagnostics.Errorf(nil,
				"union %s declares a variant value %q that collides with the discriminator field name; the generated per-variant and discriminator type names would clash",
				typ.Name, variant.Name)
			d.File = c.filePath
			c.diag.Add(d)
		}
		if seenValues.Contains(variant.Name) {
			d := diagnostics.Errorf(nil,
				"union %s declares duplicate variant value %q",
				typ.Name, variant.Name)
			d.File = c.filePath
			c.diag.Add(d)
		}
		seenValues.Add(variant.Name)
	}

	baseFields := set.New[string]()
	for i, baseRef := range form.Extends {
		base, ok := baseRef.Resolve(c.table)
		if !ok {
			d := diagnostics.Errorf(nil,
				"union %s extends unresolved type at position %d: %s",
				typ.Name, i+1, baseRef.Name)
			d.File = c.filePath
			c.diag.Add(d)
			continue
		}
		if _, isStruct := base.Form.(resolution.StructForm); !isStruct {
			d := diagnostics.Errorf(nil,
				"union %s extends non-struct type at position %d: %s",
				typ.Name, i+1, base.Name)
			d.File = c.filePath
			c.diag.Add(d)
			continue
		}
		for _, f := range resolution.UnifiedFields(base, c.table) {
			baseFields.Add(f.Name)
		}
	}

	if baseFields.Contains(form.Discriminator) {
		d := diagnostics.Errorf(nil,
			"union %s base struct declares the discriminator field %q, which is owned by the union",
			typ.Name, form.Discriminator)
		d.File = c.filePath
		c.diag.Add(d)
	}

	if baseFields.Contains("variant") {
		d := diagnostics.Errorf(nil,
			"union %s base struct declares a field named \"variant\", which is reserved for the union's generated protobuf oneof",
			typ.Name)
		d.File = c.filePath
		c.diag.Add(d)
	}

	for _, variant := range form.Variants {
		variantType, ok := variant.Type.Resolve(c.table)
		if !ok {
			d := diagnostics.Errorf(nil,
				"union %s variant %q references unresolved type: %s",
				typ.Name, variant.Name, variant.Type.Name)
			d.File = c.filePath
			c.diag.Add(d)
			continue
		}
		if _, isStruct := variantType.Form.(resolution.StructForm); !isStruct {
			d := diagnostics.Errorf(nil,
				"union %s variant %q must reference a struct type, got: %s",
				typ.Name, variant.Name, variantType.QualifiedName)
			d.File = c.filePath
			c.diag.Add(d)
			continue
		}
		for _, f := range resolution.UnifiedFields(variantType, c.table) {
			if f.Name == form.Discriminator {
				d := diagnostics.Errorf(nil,
					"union %s variant %q (%s) declares the discriminator field %q, which is owned by the union",
					typ.Name, variant.Name, variantType.QualifiedName, form.Discriminator)
				d.File = c.filePath
				c.diag.Add(d)
				break
			}
		}
	}
}

func hasCircularInheritance(typ resolution.Type, table *resolution.Table, visited set.Set[string]) bool {
	if visited.Contains(typ.QualifiedName) {
		return true
	}
	form, ok := typ.Form.(resolution.StructForm)
	if !ok || len(form.Extends) == 0 {
		return false
	}
	visited.Add(typ.QualifiedName)
	for _, extendsRef := range form.Extends {
		parent, ok := extendsRef.Resolve(table)
		if !ok {
			continue
		}
		if hasCircularInheritance(parent, table, visited) {
			return true
		}
	}
	return false
}

// dedent removes leading indentation from a multi-line string.
// It finds the minimum indentation (ignoring empty lines) and removes that
// amount from each line. Leading/trailing empty lines are also trimmed.
func dedent(s string) string {
	lines := strings.Split(s, "\n")

	minIndent := -1
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return strings.TrimSpace(s)
	}

	var result []string
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			result = append(result, "")
		} else if len(line) >= minIndent {
			result = append(result, line[minIndent:])
		} else {
			result = append(result, line)
		}
	}

	start, end := 0, len(result)
	for start < end && strings.TrimSpace(result[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(result[end-1]) == "" {
		end--
	}
	if start >= end {
		return ""
	}
	return strings.Join(result[start:end], "\n")
}
