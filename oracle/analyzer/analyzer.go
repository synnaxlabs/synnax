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
)

type analysisCtx struct {
	context.Context
	diag        *diagnostics.Diagnostics
	table       *resolution.Table
	loader      FileLoader
	ast         parser.ISchemaContext
	filePath    string
	namespace   string
	fileDomains map[string]resolution.Domain
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

	for _, typ := range c.table.TypesInNamespace(c.namespace) {
		validateExtends(c, typ)
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
	if _, exists := c.table.Get(qname); exists {
		c.diag.AddErrorf(def, c.filePath, "duplicate type definition: %s", qname)
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

func collectStructAlias(c *analysisCtx, def *parser.StructAliasContext) {
	name := def.IDENT().GetText()
	qname := c.namespace + "." + name

	if _, exists := c.table.Get(qname); exists {
		c.diag.AddErrorf(def, c.filePath, "duplicate type definition: %s", qname)
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

func collectEnum(c *analysisCtx, def parser.IEnumDefContext) {
	name := def.IDENT().GetText()
	qname := c.namespace + "." + name
	if _, exists := c.table.Get(qname); exists {
		c.diag.AddErrorf(def, c.filePath, "duplicate type definition: %s", qname)
		return
	}

	form := resolution.EnumForm{}
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
		c.diag.AddErrorf(def, c.filePath, "duplicate type definition: %s", qname)
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

	if resolution.IsPrimitive(name) && len(parts) == 1 {
		ref.Name = name
	} else if typ, ok := c.table.Lookup(ns, name); ok {
		ref.Name = typ.QualifiedName
	} else {
		c.diag.AddWarningf(nil, c.filePath, "unresolved type: %s", ref.Name)
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
		if typeRefersTo(field.Type, typ, table) {
			return true
		}
	}
	return false
}

func typeRefersTo(ref resolution.TypeRef, target *resolution.Type, table *resolution.Table) bool {
	if ref.Name == target.QualifiedName {
		return true
	}
	for _, arg := range ref.TypeArgs {
		if typeRefersTo(arg, target, table) {
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
			c.diag.AddErrorf(nil, c.filePath,
				"struct %s extends unresolved type at position %d: %s",
				typ.Name, i+1, extendsRef.Name)
			continue
		}
		parentForm, ok := parent.Form.(resolution.StructForm)
		if !ok {
			c.diag.AddErrorf(nil, c.filePath,
				"struct %s extends non-struct type at position %d: %s",
				typ.Name, i+1, parent.Name)
			continue
		}
		if parent.QualifiedName == typ.QualifiedName {
			c.diag.AddErrorf(nil, c.filePath, "struct %s cannot extend itself", typ.Name)
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
				c.diag.AddErrorf(nil, c.filePath,
					"struct %s extends %s but provides %d type arguments (need at least %d)",
					typ.Name, parent.Name, len(extendsRef.TypeArgs), requiredParams)
			}
		}
	}

	if hasCircularInheritance(typ, c.table, make(map[string]bool)) {
		c.diag.AddErrorf(nil, c.filePath, "circular inheritance detected: struct %s", typ.Name)
		return
	}

	allParentFields := make(map[string]bool)
	for _, extendsRef := range form.Extends {
		parent, ok := extendsRef.Resolve(c.table)
		if !ok {
			continue
		}
		for _, f := range resolution.UnifiedFields(parent, c.table) {
			allParentFields[f.Name] = true
		}
	}
	for _, omitted := range form.OmittedFields {
		if !allParentFields[omitted] {
			c.diag.AddErrorf(nil, c.filePath,
				"cannot omit field %q: not found in any parent struct",
				omitted)
		}
	}
}

func hasCircularInheritance(typ resolution.Type, table *resolution.Table, visited map[string]bool) bool {
	if visited[typ.QualifiedName] {
		return true
	}
	form, ok := typ.Form.(resolution.StructForm)
	if !ok || len(form.Extends) == 0 {
		return false
	}
	visited[typ.QualifiedName] = true
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
