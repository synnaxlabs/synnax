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
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/domain/presence"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

const (
	// patchDomain marks a field of a record that an update changes, or a struct whose
	// fields an update changes.
	patchDomain = "patch"
	// coreExpr marks a @patch field that only the Core sets.
	coreExpr = "core"
	// updateSuffix ends the name of the update type of a struct or union.
	updateSuffix = "Update"
	// updateAction is the standard action that carries an update.
	updateAction = "Update"
)

// The qualified names of the patch types that the fields of an update hold.
const (
	changeType    = patchNamespace + ".Change"
	structType    = patchNamespace + ".Struct"
	listType      = patchNamespace + ".List"
	valueListType = patchNamespace + ".ValueList"
	typedMapType  = patchNamespace + ".TypedMap"
)

// operatorNames are the members that an update reads as operators of a struct or a
// list, so no field of a type with an update can take one.
var operatorNames = set.New("set", "insert", "remove", "update", "move")

// declaresPatch reports whether a struct of the file has a type-level @patch, whose
// update holds types of the patch schema.
func declaresPatch(ast parser.ISchemaContext) bool {
	return lo.SomeBy(ast.AllDefinition(), func(def parser.IDefinitionContext) bool {
		return lo.SomeBy(bodyDomains(def), func(d parser.IDomainContext) bool {
			return d.IDENT().GetText() == patchDomain
		})
	})
}

// synthesizeUpdateTypes derives the update types of the file: Update for the record
// with @patch fields, <Type>Update for each struct with a type-level mark, and
// <Type>Update for each struct that their fields reach, in any namespace. Version files
// declare no updates, so it does nothing there.
func synthesizeUpdateTypes(c *analysisCtx) {
	if _, _, inVersionFile := paths.VersionFile(c.filePath); inVersionFile {
		return
	}
	s := &updateSynth{c: c}
	for _, typ := range c.table.TypesInNamespace(c.namespace) {
		s.checkMarks(typ)
	}
	if !c.diag.Ok() {
		return
	}
	for _, typ := range c.table.TypesInNamespace(c.namespace) {
		s.root(typ)
	}
	for len(s.queue) > 0 {
		typ := s.queue[0]
		s.queue = s.queue[1:]
		name := typ.Name + updateSuffix
		if _, isUnion := typ.Form.(resolution.UnionForm); isUnion {
			s.deriveUnion(typ, name, func(f resolution.Field) bool {
				return !wireOmitted(f)
			}, true)
			continue
		}
		s.derive(typ, name, wireFields(typ, c.table), nil, true)
	}
}

// updateSynth derives the update types of one file and of the types that they reach.
type updateSynth struct {
	c *analysisCtx
	// queue holds the reached structs whose update types are not yet derived.
	queue []resolution.Type
}

// report records d against the file that declares typ.
func (s *updateSynth) report(typ resolution.Type, d diagnostics.Diagnostic) {
	s.c.diag.Report(typ.FilePath, d)
}

// checkMarks reports each @patch or @presence mark of typ that holds an expression it
// does not take: a field @patch takes only core, and a type mark takes none.
func (s *updateSynth) checkMarks(typ resolution.Type) {
	for _, name := range []string{patchDomain, presence.Domain} {
		for _, e := range typ.Domains[name].Expressions {
			s.report(typ, diagnostics.Errorf(e.AST,
				"@%s on %s takes no options, not %s", name, typ.Name, e.Name,
			))
		}
	}
	form, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return
	}
	for _, f := range form.Fields {
		for _, e := range f.Domains[patchDomain].Expressions {
			if e.Name != coreExpr || len(e.Values) > 0 {
				s.report(typ, diagnostics.Errorf(e.AST,
					"@patch on %s.%s takes only core, not %s", typ.Name, f.Name, e.Name,
				))
			}
		}
	}
}

// root derives the update type of typ when typ is a record with @patch fields or a
// struct with a type-level @patch.
func (s *updateSynth) root(typ resolution.Type) {
	if typ.Synthetic || typ.Updates != "" {
		return
	}
	if form, isUnion := typ.Form.(resolution.UnionForm); isUnion {
		s.rootUnion(typ, form)
		return
	}
	form, isStruct := typ.Form.(resolution.StructForm)
	if !isStruct {
		return
	}
	if mark, whole := typ.Domains[patchDomain]; whole {
		if !importsNamespace(s.c.ast, patchNamespace) {
			s.report(typ, diagnostics.Errorf(mark.AST,
				`@patch requires import "schemas/x/patch"`,
			))
			return
		}
		s.derive(typ, typ.Name+updateSuffix, wireFields(typ, s.c.table), nil, false)
		return
	}
	if _, record := typ.Domains[actionsDomain]; !record {
		return
	}
	fields := lo.Filter(
		resolution.UnifiedFields(typ, s.c.table),
		func(f resolution.Field, _ int) bool { return patched(f) },
	)
	if len(fields) == 0 {
		return
	}
	if !s.derive(typ, updateSuffix, fields, form.TypeParams, false) {
		return
	}
	core := lo.FilterMap(fields, func(f resolution.Field, _ int) (string, bool) {
		return f.Name, coreOnly(f)
	})
	s.input(typ, resolution.UnifiedFields(typ, s.c.table), core)
	s.updateAction(typ, core)
}

// rootUnion derives the update of a union record with @patch fields in its base or its
// variants. A union record takes no Core-only field.
func (s *updateSynth) rootUnion(typ resolution.Type, form resolution.UnionForm) {
	if _, record := typ.Domains[actionsDomain]; !record {
		return
	}
	fields := baseFields(form, s.c.table)
	for _, v := range form.Variants {
		fields = append(fields, resolution.UnifiedVariantFields(typ, v, s.c.table)...)
	}
	patchedFields := lo.Filter(fields, func(f resolution.Field, _ int) bool {
		return patched(f)
	})
	if len(patchedFields) == 0 {
		return
	}
	if f, ok := lo.Find(patchedFields, coreOnly); ok {
		s.report(typ, diagnostics.Errorf(ruleCtx(f.AST),
			"%s.%s: a union record takes no Core-only field", typ.Name, f.Name,
		))
		return
	}
	if s.deriveUnion(typ, updateSuffix, patched, false) {
		s.inputUnion(typ, baseFields(form, s.c.table))
		s.updateAction(typ, nil)
	}
}

// updateAction gives the record owner the update action after create. Its payload
// holds the update of owner and the guards that must hold before it applies. core names
// the fields that only the Core sets.
func (s *updateSynth) updateAction(owner resolution.Type, core []string) {
	owner = s.c.table.MustGet(owner.QualifiedName)
	union, ok := s.c.table.Get(resolution.ActionUnion(owner))
	if !ok {
		return
	}
	actions := owner.Actions()
	if existing, ok := lo.Find(actions, func(a resolution.Action) bool {
		return a.Name == updateAction
	}); ok {
		if !existing.Update {
			s.report(owner, diagnostics.Errorf(ruleCtx(existing.AST),
				"action %s of %s: @patch already declares %s",
				updateAction, owner.Name, updateAction,
			))
		}
		return
	}
	payload := resolution.TypeRef{
		Name: owner.Namespace + "." + updateAction + "Payload",
	}
	if _, exists := s.c.table.Get(payload.Name); exists {
		s.report(owner, diagnostics.Errorf(ruleCtx(owner.AST),
			"the update action of %s needs the name %sPayload, which the schema "+
				"declares",
			owner.Name, updateAction,
		))
		return
	}
	create := s.c.table.MustGet(owner.Namespace + "." + createAction + "Payload")
	noun := strings.ReplaceAll(casing.TypeSnake(owner.Name), "_", " ")
	lo.Must0(s.c.table.Add(resolution.Type{
		Name:          updateAction + "Payload",
		Namespace:     owner.Namespace,
		QualifiedName: payload.Name,
		FilePath:      owner.FilePath,
		Form: resolution.DistinctForm{Base: generic(
			patchNamespace+"."+updateAction,
			resolution.TypeRef{Name: owner.Namespace + "." + updateSuffix},
		)},
		Domains: withDoc(
			create.Domains,
			"changes the fields of the "+noun+" that the update holds, when its "+
				"guards hold.",
		),
	}))
	action := resolution.Action{
		AST:        owner.AST,
		Name:       updateAction,
		Payload:    payload,
		Update:     true,
		CoreFields: core,
	}
	form := union.Form.(resolution.UnionForm)
	form.Variants = slices.Insert(
		slices.Clone(form.Variants),
		1,
		resolution.UnionVariant{
			Name:    casing.TypeSnake(updateAction),
			Type:    payload,
			Domains: map[string]resolution.Domain{},
		},
	)
	union.Form = form
	replaceType(s.c.table, union)
	actions = slices.Insert(slices.Clone(actions), 1, action)
	switch form := owner.Form.(type) {
	case resolution.StructForm:
		form.Actions = actions
		owner.Form = form
	case resolution.UnionForm:
		form.Actions = actions
		owner.Form = form
	}
	replaceType(s.c.table, owner)
}

// baseFields returns the fields that every variant of a union shares.
func baseFields(form resolution.UnionForm, table *resolution.Table) []resolution.Field {
	dropped := make(set.Set[string])
	for _, v := range form.Variants {
		if payload, ok := v.Type.Resolve(table); ok && v.Inline {
			dropped.Add(payload.Form.(resolution.StructForm).OmittedFields...)
		}
	}
	isDropped := func(f resolution.Field, _ int) bool { return dropped.Contains(f.Name) }
	var fields []resolution.Field
	for _, ref := range form.Extends {
		if base, _, ok := resolution.Dealias(ref, table); ok {
			fields = append(
				fields,
				lo.Reject(resolution.UnifiedFields(base, table), isDropped)...,
			)
		}
	}
	return fields
}

// deriveUnion registers the update union of owner with the given name. Each variant
// update holds the variant tag and changes the fields that keep selects. An update
// without the tag changes only base fields, so the union takes it untagged when owner
// has base fields. A reached update always holds the key of owner.
func (s *updateSynth) deriveUnion(
	owner resolution.Type,
	name string,
	keep func(resolution.Field) bool,
	reached bool,
) bool {
	qualified := owner.Namespace + "." + name
	if s.declared(owner, qualified) {
		return false
	}
	form := owner.Form.(resolution.UnionForm)
	var untagged resolution.TypeRef
	kept := func(f resolution.Field, _ int) bool { return keep(f) }
	if base := lo.Filter(baseFields(form, s.c.table), kept); len(base) > 0 {
		baseName := strings.TrimSuffix(name, updateSuffix) + "Base" + updateSuffix
		if !s.derive(owner, baseName, base, nil, reached) {
			return false
		}
		untagged = resolution.TypeRef{Name: owner.Namespace + "." + baseName}
	}
	variants := make([]resolution.UnionVariant, 0, len(form.Variants))
	for _, v := range form.Variants {
		fields := lo.Filter(resolution.UnifiedVariantFields(owner, v, s.c.table), kept)
		updates, ok := s.fields(owner, fields, reached)
		if !ok {
			return false
		}
		payload := name + pascalIdent(v.Name) + "Payload"
		if s.declared(owner, owner.Namespace+"."+payload) {
			return false
		}
		lo.Must0(s.c.table.Add(resolution.Type{
			Name:          payload,
			Namespace:     owner.Namespace,
			QualifiedName: owner.Namespace + "." + payload,
			FilePath:      owner.FilePath,
			Form:          resolution.StructForm{Fields: updates},
			Domains:       updateDomains(owner),
			Synthetic:     true,
			Updates:       owner.QualifiedName,
		}))
		variants = append(variants, resolution.UnionVariant{
			Name:    v.Name,
			Domains: v.Domains,
			Type:    resolution.TypeRef{Name: owner.Namespace + "." + payload},
			Inline:  true,
		})
	}
	noun := strings.ReplaceAll(casing.TypeSnake(owner.Name), "_", " ")
	lo.Must0(s.c.table.Add(resolution.Type{
		Name:          name,
		Namespace:     owner.Namespace,
		QualifiedName: qualified,
		FilePath:      owner.FilePath,
		Form: resolution.UnionForm{
			Discriminator: form.Discriminator,
			Enum:          form.Enum,
			Variants:      variants,
			Untagged:      untagged,
		},
		Domains: withDoc(
			updateDomains(owner),
			"holds changes to the fields of the "+noun+". A "+form.Discriminator+
				" that differs from the stored one changes the variant.",
		),
		Updates: owner.QualifiedName,
	}))
	return true
}

// declared reports whether the table already holds the type qualified that owner
// derives, and reports a schema type that takes the name.
func (s *updateSynth) declared(owner resolution.Type, qualified string) bool {
	existing, ok := s.c.table.Get(qualified)
	if ok {
		s.collides(owner, existing)
	}
	return ok
}

// collides reports, and returns true, when the schema declares existing, the name of
// an update type that owner derives.
func (s *updateSynth) collides(owner, existing resolution.Type) bool {
	if existing.Updates == owner.QualifiedName {
		return false
	}
	s.report(owner, diagnostics.Errorf(ruleCtx(owner.AST),
		"%s derives the update type %s, which the schema already declares",
		owner.Name, existing.QualifiedName,
	))
	return true
}

// input registers UpdateInput, the update that a client sends. In TypeScript it is
// Update without the fields that only the Core sets, and only a record with such fields
// has one. In Python it is Update with the key of the record. Python and C++ updates
// already leave out the Core-only fields. fields are the fields of owner.
func (s *updateSynth) input(
	owner resolution.Type,
	fields []resolution.Field,
	core []string,
) {
	update, ok := s.c.table.Get(owner.Namespace + "." + updateSuffix)
	if !ok || update.Updates != owner.QualifiedName {
		return
	}
	key, keyed := inputKey(owner, fields)
	if len(core) == 0 && !keyed {
		return
	}
	qualified := owner.Namespace + "." + inputName
	if s.inputDeclared(owner, qualified) {
		return
	}
	domains := inputDomains(owner, keyed)
	if len(core) > 0 {
		ts := layDomains(nil, update.Domains, "output", "concrete_types")["ts"]
		ts.Name = "ts"
		ts.Expressions = append(
			slices.Clone(ts.Expressions),
			resolution.Expression{Name: "use_input"},
			resolution.Expression{Name: "type_only"},
		)
		domains["ts"] = ts
	}
	form := resolution.StructForm{
		TypeParams:    update.Form.(resolution.StructForm).TypeParams,
		Extends:       []resolution.TypeRef{{Name: update.QualifiedName}},
		OmittedFields: core,
	}
	if keyed {
		form.Fields = []resolution.Field{key}
	}
	lo.Must0(s.c.table.Add(resolution.Type{
		Name:          inputName,
		Namespace:     owner.Namespace,
		QualifiedName: qualified,
		FilePath:      owner.FilePath,
		Form:          form,
		Domains:       withDoc(domains, "is the update that a client sends."),
		Updates:       owner.QualifiedName,
	}))
}

// inputUnion registers UpdateInput for a union record, for Python only: the union of
// the variant updates and the base update, each with the key of the record. base are
// the fields that every variant of owner shares.
func (s *updateSynth) inputUnion(owner resolution.Type, base []resolution.Field) {
	key, keyed := inputKey(owner, base)
	if !keyed {
		return
	}
	update := s.c.table.MustGet(owner.Namespace + "." + updateSuffix)
	form := update.Form.(resolution.UnionForm)
	qualified := owner.Namespace + "." + inputName
	if s.inputDeclared(owner, qualified) {
		return
	}
	add := func(
		name string,
		f resolution.StructForm,
		doc string,
		synthetic bool,
	) resolution.TypeRef {
		ref := resolution.TypeRef{Name: owner.Namespace + "." + name}
		domains := inputDomains(owner, true)
		if doc != "" {
			domains = withDoc(domains, doc)
		}
		lo.Must0(s.c.table.Add(resolution.Type{
			Name:          name,
			Namespace:     owner.Namespace,
			QualifiedName: ref.Name,
			FilePath:      owner.FilePath,
			Form:          f,
			Domains:       domains,
			Synthetic:     synthetic,
			Updates:       owner.QualifiedName,
		}))
		return ref
	}
	var untagged resolution.TypeRef
	if form.HasUntagged() {
		name := "Base" + inputName
		if s.inputDeclared(owner, owner.Namespace+"."+name) {
			return
		}
		untagged = add(name, resolution.StructForm{
			Extends: []resolution.TypeRef{form.Untagged},
			Fields:  []resolution.Field{key},
		}, "is the update of the shared fields that a client sends.", false)
	}
	variants := make([]resolution.UnionVariant, 0, len(form.Variants))
	for _, v := range form.Variants {
		name := inputName + pascalIdent(v.Name) + "Payload"
		if s.inputDeclared(owner, owner.Namespace+"."+name) {
			return
		}
		payload := s.c.table.MustGet(v.Type.Name).Form.(resolution.StructForm)
		fields := append([]resolution.Field{key}, payload.Fields...)
		v.Type = add(name, resolution.StructForm{Fields: fields}, "", true)
		variants = append(variants, v)
	}
	form.Variants, form.Untagged = variants, untagged
	lo.Must0(s.c.table.Add(resolution.Type{
		Name:          inputName,
		Namespace:     owner.Namespace,
		QualifiedName: qualified,
		FilePath:      owner.FilePath,
		Form:          form,
		Domains: withDoc(
			inputDomains(owner, true),
			"is the update that a client sends.",
		),
		Updates: owner.QualifiedName,
	}))
}

// inputName names the update that a client sends.
const inputName = updateSuffix + "Input"

// inputKey returns the key field of an UpdateInput, which only Python holds: the key
// that the retrieve of owner names, or else the required @key field of fields. A
// record with neither takes no key.
func inputKey(
	owner resolution.Type,
	fields []resolution.Field,
) (resolution.Field, bool) {
	ref, ok := resolution.RetrieveKey(owner)
	if !ok {
		f, found := lo.Find(fields, func(f resolution.Field) bool {
			_, isKey := f.Domains["key"]
			return isKey && !f.Optional
		})
		if !found {
			return resolution.Field{}, false
		}
		ref = f.Type
	}
	noun := strings.ReplaceAll(casing.TypeSnake(owner.Name), "_", " ")
	key := requiredField(
		"key",
		ref,
		"is the key of the "+noun+" that the update changes.",
	)
	addDomain(key.Domains, omitted("ts"))
	return key, true
}

// inputDomains returns the domains of an UpdateInput of owner: a TypedDict in Python
// when it holds a key, and no form in any other language.
func inputDomains(owner resolution.Type, keyed bool) map[string]resolution.Domain {
	domains := layDomains(nil, owner.Domains, "output", "omit", "concrete_types")
	for _, lang := range []string{"go", "ts", "cpp", "pb"} {
		domains[lang] = omitted(lang)
	}
	if !keyed {
		domains["py"] = omitted("py")
		return domains
	}
	addDomain(domains, resolution.Domain{
		Name:        "py",
		Expressions: resolution.Expressions{{Name: "dict"}},
	})
	return domains
}

// inputDeclared reports, as an error, whether the schema already declares qualified,
// the name of an input type that owner derives.
func (s *updateSynth) inputDeclared(owner resolution.Type, qualified string) bool {
	if _, exists := s.c.table.Get(qualified); !exists {
		return false
	}
	s.report(owner, diagnostics.Errorf(ruleCtx(owner.AST),
		"%s derives the update type %s, which the schema already declares",
		owner.Name, qualified,
	))
	return true
}

// patched reports whether a record field takes an update.
func patched(f resolution.Field) bool {
	_, ok := f.Domains[patchDomain]
	return ok
}

// coreOnly reports whether only the Core sets a record field.
func coreOnly(f resolution.Field) bool {
	_, ok := f.Domains[patchDomain].Expressions.Find(coreExpr)
	return ok
}

// wireFields returns the fields of a struct that have a JSON form.
func wireFields(typ resolution.Type, table *resolution.Table) []resolution.Field {
	return lo.Reject(
		resolution.UnifiedFields(typ, table),
		func(f resolution.Field, _ int) bool { return wireOmitted(f) },
	)
}

func wireOmitted(f resolution.Field) bool {
	return exprValue(f.Domains, "go", "wire") == "omit"
}

// exprValue returns the first value of the expression name in the domain lang, or
// empty when it has none.
func exprValue(domains map[string]resolution.Domain, lang, name string) string {
	expr, ok := domains[lang].Expressions.Find(name)
	if !ok || len(expr.Values) == 0 {
		return ""
	}
	if v := expr.Values[0].StringValue; v != "" {
		return v
	}
	return expr.Values[0].IdentValue
}

// derive registers the update type of owner with the given name and the update fields
// of fields. params are the type parameters of owner that the update keeps for the
// fields that hold them. A reached update always holds the key of owner, which names a
// list entry.
func (s *updateSynth) derive(
	owner resolution.Type,
	name string,
	fields []resolution.Field,
	params []resolution.TypeParam,
	reached bool,
) bool {
	qualified := owner.Namespace + "." + name
	if existing, ok := s.c.table.Get(qualified); ok {
		return !s.collides(owner, existing)
	}
	updates, ok := s.fields(owner, fields, reached)
	if !ok {
		return false
	}
	kept := keptParams(params, updates)
	noun := strings.ReplaceAll(casing.TypeSnake(owner.Name), "_", " ")
	lo.Must0(s.c.table.Add(resolution.Type{
		Name:          name,
		Namespace:     owner.Namespace,
		QualifiedName: qualified,
		FilePath:      owner.FilePath,
		Form:          resolution.StructForm{Fields: updates, TypeParams: kept},
		Domains: withDoc(
			updateDomains(owner),
			"holds changes to the fields of the "+noun+". An absent field does not "+
				"change.",
		),
		Updates: owner.QualifiedName,
	}))
	return true
}

// fields returns the update fields that change fields of owner, and false after
// reporting why one takes none.
func (s *updateSynth) fields(
	owner resolution.Type,
	fields []resolution.Field,
	reached bool,
) ([]resolution.Field, bool) {
	updates := make([]resolution.Field, 0, len(fields))
	valid := true
	for _, f := range fields {
		uf, ok := s.field(owner, f, reached)
		valid = valid && ok
		updates = append(updates, uf)
	}
	return updates, valid
}

// updateDomains returns the domains of the update type of owner: the outputs and omits
// of owner, without its protobuf form, and a TypedDict form in Python.
func updateDomains(owner resolution.Type) map[string]resolution.Domain {
	domains := layDomains(nil, owner.Domains, "output", "omit", "concrete_types")
	if _, ok := owner.Domains["pb"]; ok {
		addDomain(domains, omitted("pb"))
	}
	// Python holds an update as a TypedDict, where an absent key leaves a field alone.
	addDomain(domains, resolution.Domain{
		Name:        "py",
		Expressions: resolution.Expressions{{Name: "dict"}},
	})
	return domains
}

func omitted(lang string) resolution.Domain {
	return resolution.Domain{
		Name:        lang,
		Expressions: resolution.Expressions{{Name: "omit"}},
	}
}

// keptParams returns the params that the update fields hold, bound to fresh copies so
// the update type owns them.
func keptParams(
	params []resolution.TypeParam,
	fields []resolution.Field,
) []resolution.TypeParam {
	held := make(set.Set[string])
	var walk func(resolution.TypeRef)
	walk = func(ref resolution.TypeRef) {
		if ref.TypeParam != nil {
			held.Add(ref.TypeParam.Name)
		}
		for _, arg := range ref.TypeArgs {
			walk(arg)
		}
	}
	for _, f := range fields {
		walk(f.Type)
	}
	kept := lo.Filter(params, func(p resolution.TypeParam, _ int) bool {
		return held.Contains(p.Name)
	})
	if len(kept) == 0 {
		return nil
	}
	kept = slices.Clone(kept)
	var bind func(*resolution.TypeRef)
	bind = func(ref *resolution.TypeRef) {
		if ref.TypeParam != nil {
			i := slices.IndexFunc(kept, func(p resolution.TypeParam) bool {
				return p.Name == ref.TypeParam.Name
			})
			ref.TypeParam = &kept[i]
		}
		for i := range ref.TypeArgs {
			bind(&ref.TypeArgs[i])
		}
	}
	for i := range fields {
		bind(&fields[i].Type)
	}
	return kept
}

// field returns the update field that changes f of owner, and false after reporting
// why f takes none. A reached update holds a required key as it is.
func (s *updateSynth) field(
	owner resolution.Type,
	f resolution.Field,
	reached bool,
) (resolution.Field, bool) {
	at := ruleCtx(f.AST)
	if operatorNames.Contains(f.Name) {
		s.report(owner, diagnostics.Errorf(at,
			"%s.%s: an update reads %s as an operator, so no field of %s can take "+
				"the name",
			owner.Name, f.Name, f.Name, owner.Name,
		))
		return resolution.Field{}, false
	}
	if wireOmitted(f) {
		s.report(owner, diagnostics.Errorf(at,
			"%s.%s: a field without a JSON form takes no update", owner.Name, f.Name,
		))
		return resolution.Field{}, false
	}
	uf := resolution.Field{
		Name:     f.Name,
		AST:      f.AST,
		Domains:  updateFieldDomains(f),
		Optional: true,
	}
	// A record written by hand in a language holds no field whose type is absent
	// there, and no field-level omit can say so.
	for lang, d := range owner.Domains {
		if _, hand := d.Expressions.Find("hand"); hand && s.absent(f.Type, lang) {
			addDomain(uf.Domains, omitted(lang))
		}
	}
	if _, isKey := f.Domains["key"]; isKey && reached && !f.Optional {
		uf.Type, uf.Optional = f.Type, false
		return uf, true
	}
	uf.Type = s.change(f.Type)
	if f.Optional {
		uf.Type = generic(changeType, uf.Type)
	}
	return uf, true
}

// updateFieldDomains returns the domains of the update field that changes f: its doc
// and its omits. Python and C++ only send updates, so they leave out a field that only
// the Core sets.
func updateFieldDomains(f resolution.Field) map[string]resolution.Domain {
	domains := make(map[string]resolution.Domain)
	if d, ok := f.Domains["doc"]; ok {
		domains["doc"] = d
	}
	for _, lang := range []string{"ts", "py", "cpp", "pb"} {
		if _, ok := f.Domains[lang].Expressions.Find("omit"); ok {
			addDomain(domains, omitted(lang))
		}
	}
	if coreOnly(f) {
		addDomain(domains, omitted("py"))
		addDomain(domains, omitted("cpp"))
	}
	return domains
}

// change returns the type that an update field holds to change a required value of
// ref: the operators of a list or a map, the merge of a struct, the update of a union,
// or ref itself, which replaces the value whole.
func (s *updateSynth) change(ref resolution.TypeRef) resolution.TypeRef {
	if ref.IsTypeParam() {
		return ref
	}
	if ref.Name == "Map" && len(ref.TypeArgs) == 2 {
		key, value := ref.TypeArgs[0], ref.TypeArgs[1]
		return generic(typedMapType, key, value, s.valueUpdate(value))
	}
	if ref.Name == "Array" && ref.ArraySize == nil && len(ref.TypeArgs) == 1 {
		elem := ref.TypeArgs[0]
		if entry, key, ok := s.keyedEntry(elem); ok {
			return generic(listType, elem, key, s.updateOf(entry))
		}
		return generic(valueListType, elem)
	}
	typ, _, ok := resolution.Dealias(ref, s.c.table)
	if !ok || !mergeable(typ) {
		return ref
	}
	if _, isUnion := typ.Form.(resolution.UnionForm); isUnion {
		return s.updateOf(typ)
	}
	return generic(structType, s.updateOf(typ), ref)
}

// valueUpdate returns the type that merges into a map value of ref: the update of a
// struct or a union, or ref itself, which replaces the value.
func (s *updateSynth) valueUpdate(ref resolution.TypeRef) resolution.TypeRef {
	if ref.IsTypeParam() {
		return ref
	}
	typ, _, ok := resolution.Dealias(ref, s.c.table)
	if !ok || !mergeable(typ) {
		return ref
	}
	return s.updateOf(typ)
}

// updateOf returns a reference to the update type of the reached typ, and queues typ
// for its derivation.
func (s *updateSynth) updateOf(typ resolution.Type) resolution.TypeRef {
	s.queue = append(s.queue, typ)
	return resolution.TypeRef{Name: typ.QualifiedName + updateSuffix}
}

// keyedEntry returns the entry type of a list whose entries an update names by key,
// and the type of the key. The key of a union entry is in its base.
func (s *updateSynth) keyedEntry(
	elem resolution.TypeRef,
) (resolution.Type, resolution.TypeRef, bool) {
	if elem.IsTypeParam() {
		return resolution.Type{}, resolution.TypeRef{}, false
	}
	typ, _, ok := resolution.Dealias(elem, s.c.table)
	if !ok || !mergeable(typ) {
		return resolution.Type{}, resolution.TypeRef{}, false
	}
	fields := wireFields(typ, s.c.table)
	if form, isUnion := typ.Form.(resolution.UnionForm); isUnion {
		fields = baseFields(form, s.c.table)
	}
	key, ok := lo.Find(fields, func(f resolution.Field) bool {
		_, isKey := f.Domains["key"]
		return isKey && !f.Optional
	})
	return typ, key.Type, ok
}

// mergeable reports whether an update merges into a value of typ, a struct or a union,
// field by field. A value of any other type, such as a filter or a struct with a
// hand-written Go form, is replaced whole.
func mergeable(typ resolution.Type) bool {
	if typ.Filter != nil || typ.Synthetic || omit.IsSkipped(typ, "go") {
		return false
	}
	if exprValue(typ.Domains, "go", "json") == "hand" {
		return false
	}
	switch form := typ.Form.(type) {
	case resolution.StructForm:
		return !form.IsGeneric()
	case resolution.UnionForm:
		return !form.Adjacent
	}
	return false
}

// absent reports whether ref names a declared type, or holds one, that does not exist
// in lang. Built-in types exist in every language.
func (s *updateSynth) absent(ref resolution.TypeRef, lang string) bool {
	if lo.SomeBy(ref.TypeArgs, func(arg resolution.TypeRef) bool {
		return s.absent(arg, lang)
	}) {
		return true
	}
	if ref.IsTypeParam() {
		return false
	}
	typ, ok := s.c.table.Get(ref.Name)
	return ok && typ.FilePath != "" && omit.IsAbsent(typ, lang)
}

func generic(name string, args ...resolution.TypeRef) resolution.TypeRef {
	return resolution.TypeRef{Name: name, TypeArgs: args}
}
