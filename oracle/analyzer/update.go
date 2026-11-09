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
)

// The qualified names of the patch types that the fields of an update hold.
const (
	changeType    = patchNamespace + ".Change"
	structType    = patchNamespace + ".Struct"
	listType      = patchNamespace + ".List"
	valueListType = patchNamespace + ".ValueList"
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
		s.derive(typ, typ.Name+updateSuffix, wireFields(typ, c.table), nil, true)
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
	s.derive(typ, updateSuffix, fields, form.TypeParams, false)
	core := lo.FilterMap(fields, func(f resolution.Field, _ int) (string, bool) {
		return f.Name, coreOnly(f)
	})
	if len(core) > 0 {
		s.input(typ, core)
	}
}

// input registers UpdateInput, the update that a TypeScript client sends: Update
// without the fields that only the Core sets. Python and C++ updates already leave
// them out, so only TypeScript renders it.
func (s *updateSynth) input(owner resolution.Type, core []string) {
	update, ok := s.c.table.Get(owner.Namespace + "." + updateSuffix)
	if !ok || update.Updates != owner.QualifiedName {
		return
	}
	name := updateSuffix + "Input"
	qualified := owner.Namespace + "." + name
	if _, exists := s.c.table.Get(qualified); exists {
		s.report(owner, diagnostics.Errorf(ruleCtx(owner.AST),
			"%s derives the update type %s, which the schema already declares",
			owner.Name, qualified,
		))
		return
	}
	domains := map[string]resolution.Domain{}
	for _, lang := range []string{"go", "py", "cpp", "pb"} {
		domains[lang] = omitted(lang)
	}
	ts := layDomains(nil, update.Domains, "output", "concrete_types")["ts"]
	ts.Name = "ts"
	ts.Expressions = append(
		slices.Clone(ts.Expressions),
		resolution.Expression{Name: "use_input"},
		resolution.Expression{Name: "type_only"},
	)
	domains["ts"] = ts
	lo.Must0(s.c.table.Add(resolution.Type{
		Name:          name,
		Namespace:     owner.Namespace,
		QualifiedName: qualified,
		FilePath:      owner.FilePath,
		Form: resolution.StructForm{
			TypeParams:    update.Form.(resolution.StructForm).TypeParams,
			Extends:       []resolution.TypeRef{{Name: update.QualifiedName}},
			OmittedFields: core,
		},
		Domains: withDoc(domains, "is the update that a client sends."),
		Updates: owner.QualifiedName,
	}))
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
) {
	qualified := owner.Namespace + "." + name
	if existing, ok := s.c.table.Get(qualified); ok {
		if existing.Updates != owner.QualifiedName {
			s.report(owner, diagnostics.Errorf(ruleCtx(owner.AST),
				"%s derives the update type %s, which the schema already declares",
				owner.Name, qualified,
			))
		}
		return
	}
	updates := make([]resolution.Field, 0, len(fields))
	valid := true
	for _, f := range fields {
		uf, ok := s.field(owner, f, reached)
		valid = valid && ok
		updates = append(updates, uf)
	}
	if !valid {
		return
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
// ref: the operators of a list, the merge of a struct, or ref itself, which replaces
// the value whole.
func (s *updateSynth) change(ref resolution.TypeRef) resolution.TypeRef {
	if ref.IsTypeParam() {
		return ref
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
	if _, isStruct := typ.Form.(resolution.StructForm); isStruct {
		return generic(structType, s.updateOf(typ), ref)
	}
	return ref
}

// updateOf returns a reference to the update type of the reached typ, and queues typ
// for its derivation.
func (s *updateSynth) updateOf(typ resolution.Type) resolution.TypeRef {
	s.queue = append(s.queue, typ)
	return resolution.TypeRef{Name: typ.QualifiedName + updateSuffix}
}

// keyedEntry returns the entry type of a list whose entries an update names by key,
// and the type of the key.
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
	if _, isStruct := typ.Form.(resolution.StructForm); !isStruct {
		return resolution.Type{}, resolution.TypeRef{}, false
	}
	key, ok := lo.Find(wireFields(typ, s.c.table), func(f resolution.Field) bool {
		_, isKey := f.Domains["key"]
		return isKey && !f.Optional
	})
	return typ, key.Type, ok
}

// mergeable reports whether an update merges into a value of typ, field by field. A
// value of any other type, such as a filter or a struct with a hand-written Go form,
// is replaced whole.
func mergeable(typ resolution.Type) bool {
	if typ.Filter != nil || typ.Synthetic || omit.IsSkipped(typ, "go") {
		return false
	}
	if exprValue(typ.Domains, "go", "json") == "hand" {
		return false
	}
	form, isStruct := typ.Form.(resolution.StructForm)
	return isStruct && !form.IsGeneric()
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
