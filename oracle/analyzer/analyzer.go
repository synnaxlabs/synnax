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
	"strconv"
	"strings"

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
	return table, diag
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

	for _, imp := range c.ast.AllImportStmt() {
		path := strings.Trim(imp.STRING_LIT().GetText(), `"`)
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
	for _, typ := range types {
		for i, t := range c.table.Types {
			if t.QualifiedName == typ.QualifiedName {
				c.table.Types[i] = typ
				break
			}
		}
	}

	finalizeEnumExtensions(c)

	for _, typ := range c.table.TypesInNamespace(c.namespace) {
		validateExtends(c, typ)
		validateTypeParams(c, typ)
		validateUnion(c, typ)
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

func addEnumDiag(c *analysisCtx, typ resolution.Type, format string, args ...any) {
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
		addEnumDiag(c, typ, "enum %s has a cyclic extends chain", typ.QualifiedName)
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
				addEnumDiag(c, typ, "enum %s inherits member %q with conflicting values from %s",
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
			addEnumDiag(c, typ, "enum %s extends unknown enum %q", typ.QualifiedName, form.Extends[i].Name)
			return nil, false, false
		}
		if _, ok := parent.Form.(resolution.EnumForm); !ok {
			addEnumDiag(c, typ, "enum %s extends %s, which is not an enum", typ.QualifiedName, parent.QualifiedName)
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
			addEnumDiag(c, typ, "enum %s extends enums of mixed integer and string kinds", typ.QualifiedName)
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
		addEnumDiag(c, typ, "enum %s adds members of a different kind than the enums it extends",
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
	for k, v := range c.fileDomains {
		domains[k] = v
	}

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
	body := def.ActionBody()
	if body == nil {
		return action
	}
	hasKeyDomain := false
	for _, f := range body.AllFieldDef() {
		field := collectField(c, f, typeParams, &hasKeyDomain)
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
	for k, v := range c.fileDomains {
		domains[k] = v
	}
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
	name := def.IDENT().GetText()
	tr := def.TypeRef()

	normalCtx, isNormal := tr.(*parser.TypeRefNormalContext)
	mapCtx, isMap := tr.(*parser.TypeRefMapContext)
	isOptional, isHardOptional := false, false
	isArray := false
	var arraySize *int64

	if isNormal {
		isOptional, isHardOptional = extractTypeModifiersNormal(normalCtx)
		if arrMod := normalCtx.ArrayModifier(); arrMod != nil {
			isArray = true
			if intLit := arrMod.INT_LIT(); intLit != nil {
				size, _ := strconv.ParseInt(intLit.GetText(), 10, 64)
				arraySize = &size
			}
		}
	} else if isMap {
		isOptional, isHardOptional = extractTypeModifiersMap(mapCtx)
	}

	typeRef := collectTypeRef(tr, typeParams)

	if isArray {
		typeRef = resolution.TypeRef{
			Name:      "Array",
			TypeArgs:  []resolution.TypeRef{typeRef},
			ArraySize: arraySize,
		}
	}

	field := resolution.Field{
		Name:           name,
		Type:           typeRef,
		Domains:        make(map[string]resolution.Domain),
		IsOptional:     isOptional,
		IsHardOptional: isHardOptional,
		AST:            def,
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

	if fb := def.FieldBody(); fb != nil {
		for _, d := range fb.AllDomain() {
			de := collectDomain(d)
			field.Domains[de.Name] = de
			if de.Name == "key" {
				*hasKeyDomain = true
			}
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
	out := resolution.ExpressionValue{Kind: resolution.ValueKindArray}
	if arr := fd.ArrayDefault(); arr != nil {
		for _, el := range arr.AllExpressionValue() {
			out.Elements = append(out.Elements, collectValue(el))
		}
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
	for k, v := range c.fileDomains {
		domains[k] = v
	}

	if body := def.UnionBody(); body != nil {
		for _, v := range body.AllUnionVariant() {
			form.Variants = append(form.Variants, collectUnionVariant(v))
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

func collectUnionVariant(def parser.IUnionVariantContext) resolution.UnionVariant {
	variant := resolution.UnionVariant{
		Name:    def.VariantName().GetText(),
		Type:    collectTypeRef(def.TypeRef(), nil),
		Domains: make(map[string]resolution.Domain),
		AST:     def,
	}
	if body := def.UnionVariantBody(); body != nil {
		for _, d := range body.AllDomain() {
			de := collectDomain(d)
			variant.Domains[de.Name] = de
		}
	}
	return variant
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
	for k, v := range c.fileDomains {
		domains[k] = v
	}

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
	for k, v := range c.fileDomains {
		domains[k] = v
	}

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

func extractTypeModifiersNormal(tr *parser.TypeRefNormalContext) (isOptional, isHardOptional bool) {
	mods := tr.TypeModifiers()
	if mods == nil {
		return false, false
	}
	questionCount := len(mods.AllQUESTION())
	if questionCount >= 2 {
		return false, true
	}
	return questionCount >= 1, false
}

func extractTypeModifiersMap(tr *parser.TypeRefMapContext) (isOptional, isHardOptional bool) {
	mods := tr.TypeModifiers()
	if mods == nil {
		return false, false
	}
	questionCount := len(mods.AllQUESTION())
	if questionCount >= 2 {
		return false, true
	}
	return questionCount >= 1, false
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
