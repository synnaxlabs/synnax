// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/presence"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/typemap"
	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

const (
	patchImportPath  = "github.com/synnaxlabs/x/patch"
	errorsImportPath = "github.com/synnaxlabs/x/errors"
	changeType       = "patch.Change"
	structType       = "patch.Struct"
	listType         = "patch.List"
	valueListType    = "patch.ValueList"
)

// applier renders the ApplyTo and Defaults methods of the update types of one file.
type applier struct {
	data *templateData
	// defaulted holds the update structs that an optional field holds, which need a
	// Defaults method for a merge into an absent value.
	defaulted set.Set[string]
}

// applyMethods returns the methods that apply the update type u.
func applyMethods(
	u resolution.Type,
	data *templateData,
	defaulted set.Set[string],
) (string, error) {
	a := &applier{data: data, defaulted: defaulted}
	if _, isUnion := u.Form.(resolution.UnionForm); isUnion {
		return a.unionMethods(u)
	}
	return a.structMethods(u)
}

// defaultedUpdates returns the update structs that an optional field of an update
// holds.
func defaultedUpdates(table *resolution.Table) set.Set[string] {
	defaulted := make(set.Set[string])
	for _, t := range table.Types {
		if t.Updates == "" {
			continue
		}
		for _, f := range updateFields(t, table) {
			if f.Type.Name == changeType && len(f.Type.TypeArgs) == 1 &&
				f.Type.TypeArgs[0].Name == structType {
				defaulted.Add(f.Type.TypeArgs[0].TypeArgs[0].Name)
			}
		}
	}
	return defaulted
}

// updateFields returns the fields of an update struct, or of every variant of an
// update union.
func updateFields(t resolution.Type, table *resolution.Table) []resolution.Field {
	switch form := t.Form.(type) {
	case resolution.StructForm:
		return form.Fields
	case resolution.UnionForm:
		return lo.FlatMap(
			form.Variants,
			func(v resolution.UnionVariant, _ int) []resolution.Field {
				return resolution.UnifiedVariantFields(t, v, table)
			},
		)
	}
	return nil
}

// structMethods returns the methods of the update struct u. Each applies u to the
// struct that u changes: the shared base of a union for the update that holds no
// variant tag.
func (a *applier) structMethods(u resolution.Type) (string, error) {
	target, err := a.structTarget(u)
	if err != nil {
		return "", err
	}
	targetName := a.render(resolution.TypeRef{Name: target.QualifiedName})
	updateName := naming.GetGoName(u)
	fields := u.Form.(resolution.StructForm).Fields
	var b strings.Builder
	if err := a.applyTo(
		&b,
		updateName,
		targetName,
		fields,
		resolution.UnifiedFields(target, a.data.table),
		func(name string) string { return a.structSelector(target, name) },
	); err != nil {
		return "", errors.Wrap(err, u.QualifiedName)
	}
	if a.defaulted.Contains(u.QualifiedName) {
		a.defaults(&b, defaultsData{
			update:  updateName,
			target:  targetName,
			fields:  fields,
			record:  resolution.UnifiedFields(target, a.data.table),
			applies: a.hasApplyDefaults(resolution.TypeRef{Name: target.QualifiedName}),
		})
	}
	return b.String(), nil
}

// structTarget returns the struct that the update struct u changes.
func (a *applier) structTarget(u resolution.Type) (resolution.Type, error) {
	target, ok := a.data.table.Get(u.Updates)
	if !ok {
		return resolution.Type{}, errors.Newf(
			"%s changes the unresolved type %s", u.QualifiedName, u.Updates,
		)
	}
	form, isUnion := target.Form.(resolution.UnionForm)
	if !isUnion {
		return target, nil
	}
	if len(form.Extends) != 1 {
		return resolution.Type{}, errors.Newf(
			"%s changes the base of %s, which needs exactly one base struct",
			u.QualifiedName, target.QualifiedName,
		)
	}
	base, _, ok := resolution.Dealias(form.Extends[0], a.data.table)
	if !ok {
		return resolution.Type{}, errors.Newf(
			"%s extends the unresolved base %s",
			target.QualifiedName,
			form.Extends[0].Name,
		)
	}
	return base, nil
}

// unionBase is how generated code reads and writes the base fields of a union value.
type unionBase struct {
	// typ is the base struct and field is its selector in a variant that embeds it.
	typ, field string
	// shared reports that every variant embeds the base, so the union has the Base and
	// SetBase methods. Otherwise the update declares helpers of its own.
	shared bool
	// name is the Go name of the union.
	name string
}

// unionBase returns the base of the union record, if it has exactly one.
func (a *applier) unionBase(record resolution.Type) (unionBase, bool) {
	form := record.Form.(resolution.UnionForm)
	if len(form.Extends) != 1 {
		return unionBase{}, false
	}
	parent, ok := form.Extends[0].Resolve(a.data.table)
	if !ok {
		return unionBase{}, false
	}
	typ := resolveExtendsType(form.Extends[0], parent, probeData(a.data))
	_, shared := resolver.SharedBase(form, "go", a.data.table)
	return unionBase{
		typ:    typ,
		field:  embedFieldName(typ),
		shared: shared,
		name:   unionGoName(record),
	}, true
}

// get returns the expression for the base of the union that x holds, or points to
// when pointer is true.
func (b unionBase) get(x string, pointer bool) string {
	if b.shared {
		return x + ".Base()"
	}
	if !pointer {
		x = "&" + x
	}
	return "baseOf" + b.name + "(" + x + ")"
}

// set returns the statement that replaces the base of the union that x points to.
func (b unionBase) set(x, base string) string {
	if b.shared {
		return x + ".SetBase(" + base + ")"
	}
	return "setBaseOf" + b.name + "(" + x + ", " + base + ")"
}

// helpers returns the functions that read and write the base of a union whose
// variants do not all embed it. A variant that drops a base field holds the rest.
func (a *applier) helpers(record resolution.Type, base unionBase) string {
	table := a.data.table
	form := record.Form.(resolution.UnionForm)
	parent, _, _ := resolution.Dealias(form.Extends[0], table)
	resolved, _ := form.Extends[0].Resolve(table)
	base.typ = resolveExtendsType(form.Extends[0], resolved, a.data)
	baseFields := resolution.UnifiedFields(parent, table)
	var get, put strings.Builder
	fmt.Fprintf(&get, `
// baseOf%s returns the base fields that the variant of x holds.
func baseOf%s(x *%s) %s {
	var base %s
	switch v := x.Variant.(type) {
`, base.name, base.name, base.name, base.typ, base.typ)
	fmt.Fprintf(&put, `
// setBaseOf%s replaces the base fields that the variant of x holds.
func setBaseOf%s(x *%s, base %s) {
	switch v := x.Variant.(type) {
`, base.name, base.name, base.name, base.typ)
	for _, v := range form.Variants {
		vt := casing.VariantTypeName(base.name, v.Name)
		fmt.Fprintf(&get, "\tcase %s:\n", vt)
		fmt.Fprintf(&put, "\tcase %s:\n", vt)
		inherited, _ := resolver.VariantBases(form, v, "go", table)
		if slices.ContainsFunc(inherited, func(r resolution.TypeRef) bool {
			return r.Name == form.Extends[0].Name
		}) {
			fmt.Fprintf(&get, "\t\tbase = v.%s\n", base.field)
			fmt.Fprintf(&put, "\t\tv.%s = base\n", base.field)
		} else {
			held := set.New(lo.Map(
				resolution.UnifiedVariantFields(record, v, table),
				func(f resolution.Field, _ int) string { return f.Name },
			)...)
			for _, f := range baseFields {
				if held.Contains(f.Name) {
					from := a.structSelector(parent, f.Name)
					to := a.variantSelector(record, v, f.Name)
					fmt.Fprintf(&get, "\t\tbase.%s = v.%s\n", from, to)
					fmt.Fprintf(&put, "\t\tv.%s = base.%s\n", to, from)
				}
			}
		}
		put.WriteString("\t\tx.Variant = v\n")
	}
	get.WriteString("\t}\n\treturn base\n}\n")
	put.WriteString("\t}\n}\n")
	return get.String() + put.String()
}

// unionMethods returns the methods of the update union u and of its variant updates.
func (a *applier) unionMethods(u resolution.Type) (string, error) {
	table := a.data.table
	record, ok := table.Get(u.Updates)
	if !ok {
		return "", errors.Newf(
			"%s changes the unresolved type %s",
			u.QualifiedName,
			u.Updates,
		)
	}
	rform := record.Form.(resolution.UnionForm)
	uform := u.Form.(resolution.UnionForm)
	recordName, updateName := unionGoName(record), unionGoName(u)
	probe, err := processUnion(record, probeData(a.data))
	if err != nil {
		return "", err
	}
	base, hasBase := a.unionBase(record)
	shared := sharedFields(record, table)
	sharedNames := set.New(lo.Map(
		shared,
		func(f resolution.Field, _ int) string { return f.Name },
	)...)
	var b strings.Builder
	b.WriteString(strings.TrimPrefix(unionApplyDoc, "\n"))
	fmt.Fprintf(
		&b,
		`func (u %s) ApplyTo(x *%s) error {
	switch v := u.Variant.(type) {
	case nil:
`,
		updateName,
		recordName,
	)
	if uform.HasUntagged() {
		if !hasBase {
			return "", errors.Newf(
				"%s: an update without a %s needs one base of %s",
				u.QualifiedName, uform.Discriminator, record.QualifiedName,
			)
		}
		a.data.AddExternal(errorsImportPath)
		a.data.AddExternal(validateImportPath)
		fmt.Fprintf(&b, `	case %s:
		if x.Variant == nil {
			return errors.Wrap(validate.ErrValidation, "%s is required")
		}
		base := %s
		if err := v.ApplyTo(&base); err != nil {
			return err
		}
		%s
`,
			a.render(uform.Untagged),
			casing.FieldSnake(uform.Discriminator),
			base.get("x", true),
			base.set("x", "base"),
		)
	}
	var variants strings.Builder
	for i, v := range uform.Variants {
		rv, ok := lo.Find(rform.Variants, func(r resolution.UnionVariant) bool {
			return r.Name == v.Name
		})
		if !ok {
			return "", errors.Newf(
				"%s holds no variant %s",
				record.QualifiedName,
				v.Name,
			)
		}
		vt := casing.VariantTypeName(recordName, v.Name)
		ut := casing.VariantTypeName(updateName, v.Name)
		fmt.Fprintf(&b, `	case %s:
		next, same := x.Variant.(%s)
		if !same {
			var err error
			if next, err = v.Defaults(); err != nil {
				return err
			}
`, ut, vt)
		switch {
		case hasBase && base.shared:
			fmt.Fprintf(&b, `			if x.Variant != nil {
				next.%s = x.Base()
			}
`, base.field)
		case hasBase:
			parent, _, _ := resolution.Dealias(rform.Extends[0], table)
			fmt.Fprintf(&b, "\t\t\tif x.Variant != nil {\n\t\t\t\tbase := %s\n",
				base.get("x", true))
			for _, f := range shared {
				fmt.Fprintf(&b, "\t\t\t\tnext.%s = base.%s\n",
					a.variantSelector(record, rv, f.Name),
					a.structSelector(parent, f.Name))
			}
			b.WriteString("\t\t\t}\n")
		}
		b.WriteString(`		}
		if err := v.ApplyTo(&next); err != nil {
			return err
		}
		x.Variant = next
`)
		payload, ok := v.Type.Resolve(table)
		if !ok {
			return "", errors.Newf("%s: unresolved variant %s", u.QualifiedName, v.Name)
		}
		fields := payload.Form.(resolution.StructForm).Fields
		recordFields := resolution.UnifiedVariantFields(record, rv, table)
		variants.WriteString("\n")
		if err := a.applyTo(
			&variants,
			ut,
			vt,
			fields,
			recordFields,
			func(name string) string { return a.variantSelector(record, rv, name) },
		); err != nil {
			return "", errors.Wrap(err, u.QualifiedName+"."+v.Name)
		}
		a.defaults(&variants, defaultsData{
			update: ut,
			target: vt,
			fields: lo.Filter(fields, func(f resolution.Field, _ int) bool {
				return !hasBase || !sharedNames.Contains(f.Name)
			}),
			record:  recordFields,
			applies: probe.Variants[i].NeedsApplyDefaults,
		})
	}
	b.WriteString("\t}\n\treturn nil\n}\n")
	if key, ok := a.unionKey(uform); ok {
		fmt.Fprintf(&b, `
// Key returns the key of the entry that u changes.
func (u %s) Key() %s {
	var key %s
	switch v := u.Variant.(type) {
`, updateName, key.typ, key.typ)
		if uform.HasUntagged() {
			fmt.Fprintf(
				&b,
				"\tcase %s:\n\t\tkey = v.%s\n",
				a.render(uform.Untagged),
				key.field,
			)
		}
		for _, v := range uform.Variants {
			fmt.Fprintf(
				&b, "\tcase %s:\n\t\tkey = v.%s\n",
				casing.VariantTypeName(updateName, v.Name), key.field,
			)
		}
		b.WriteString("\t}\n\treturn key\n}\n")
	}
	if hasBase && !base.shared {
		b.WriteString(a.helpers(record, base))
	}
	b.WriteString(variants.String())
	return b.String(), nil
}

// unionApplyDoc is the doc comment of the ApplyTo method of an update union.
const unionApplyDoc = `
// ApplyTo applies u to x. An update of the variant that x holds merges into it. An
// update of another variant replaces it, keeping the fields that every variant shares
// and filling the rest with their defaults.
`

// unionGoName returns the Go name of the union typ, which @go name overrides.
func unionGoName(typ resolution.Type) string {
	if override := domain.GetStringFromType(typ, "go", "name"); override != "" {
		return override
	}
	return typ.Name
}

// sharedFields returns the base fields of the union record that every variant holds, in
// the order of the base.
func sharedFields(record resolution.Type, table *resolution.Table) []resolution.Field {
	form := record.Form.(resolution.UnionForm)
	var fields []resolution.Field
	for _, ref := range form.Extends {
		if base, _, ok := resolution.Dealias(ref, table); ok {
			fields = append(fields, resolution.UnifiedFields(base, table)...)
		}
	}
	for _, v := range form.Variants {
		held := set.New(lo.Map(
			resolution.UnifiedVariantFields(record, v, table),
			func(f resolution.Field, _ int) string { return f.Name },
		)...)
		fields = lo.Filter(fields, func(f resolution.Field, _ int) bool {
			return held.Contains(f.Name)
		})
	}
	return fields
}

type keyData struct{ field, typ string }

// unionKey returns the key that every variant update of form holds, which a reached
// update keeps required.
func (a *applier) unionKey(form resolution.UnionForm) (keyData, bool) {
	if !form.HasUntagged() {
		return keyData{}, false
	}
	base, ok := form.Untagged.Resolve(a.data.table)
	if !ok {
		return keyData{}, false
	}
	f, ok := lo.Find(
		base.Form.(resolution.StructForm).Fields,
		func(f resolution.Field) bool {
			return !f.Optional
		},
	)
	if !ok {
		return keyData{}, false
	}
	return keyData{field: naming.GetFieldName(f), typ: a.render(f.Type)}, true
}

// applyTo writes the ApplyTo method of the update struct named update, which applies
// fields to the matching fields of record in a value of target. sel returns the
// selector of a record field in target.
func (a *applier) applyTo(
	b *strings.Builder,
	update, target string,
	fields, record []resolution.Field,
	sel func(name string) string,
) error {
	fmt.Fprintf(
		b,
		`// ApplyTo applies u to x. An absent field of u leaves its field of x alone.
func (u %s) ApplyTo(x *%s) error {
`,
		update,
		target,
	)
	for _, uf := range fields {
		if !uf.Optional {
			continue
		}
		rf, ok := lo.Find(
			record,
			func(f resolution.Field) bool { return f.Name == uf.Name },
		)
		if !ok {
			return errors.Newf("%s holds no field %s", target, uf.Name)
		}
		stmt, err := a.applyField(uf, rf, "x."+sel(rf.Name))
		if err != nil {
			return errors.Wrap(err, uf.Name)
		}
		b.WriteString(stmt)
	}
	b.WriteString("\treturn nil\n}\n")
	return nil
}

// applyField returns the statement that applies the update field uf to the field rf,
// which rec selects.
func (a *applier) applyField(uf, rf resolution.Field, rec string) (string, error) {
	table := a.data.table
	upd := "u." + naming.GetFieldName(uf)
	check := func(call string) string {
		return fmt.Sprintf("\tif err := %s; err != nil {\n\t\treturn err\n\t}\n", call)
	}
	present := func(body string) string {
		return fmt.Sprintf("\tif %s != nil {\n%s\t}\n", upd, indentLines(body))
	}
	if presence.Field(uf, table) {
		inner := uf.Type.TypeArgs[0]
		switch {
		case inner.Name == structType:
			a.data.AddExternal(patchImportPath)
			su, st := a.render(inner.TypeArgs[0]), a.render(inner.TypeArgs[1])
			return check(fmt.Sprintf(
				"patch.ApplyOptionalStruct(\n\t\t&%s,\n\t\t%s,\n\t\t"+
					"func(v *%s, w %s) error { return w.ApplyTo(v) },\n"+
					"\t\t%s.Defaults,\n\t)",
				rec, upd, st, su, su,
			)), nil
		case a.operators(inner):
			return "", errors.New("an optional list or union takes no update")
		case typemap.Nilable(rf.Type, table):
			return fmt.Sprintf(
				"\tif %s.Cleared() {\n\t\t%s = nil\n\t} "+
					"else if v, ok := %s.Get(); ok {\n\t\t%s = v\n\t}\n",
				upd,
				rec,
				upd,
				rec,
			), nil
		default:
			return fmt.Sprintf("\t%s.ApplyOptional(&%s)\n", upd, rec), nil
		}
	}
	value := "*" + upd
	if typemap.Nilable(uf.Type, table) {
		value = upd
	}
	switch uf.Type.Name {
	case listType:
		a.data.AddExternal(patchImportPath)
		e, k, eu := uf.Type.TypeArgs[0], uf.Type.TypeArgs[1], uf.Type.TypeArgs[2]
		entryKey, err := a.entryKey(e, "e")
		if err != nil {
			return "", err
		}
		updateKey, err := a.updateKey(eu, "w")
		if err != nil {
			return "", err
		}
		et, kt, ut := a.render(e), a.render(k), a.render(eu)
		return present(check(fmt.Sprintf(
			"patch.ApplyList(\n\t\t&%s,\n\t\t%s,\n\t\tfunc(e %s) %s { return %s },\n"+
				"\t\tfunc(w %s) %s { return %s },\n"+
				"\t\tfunc(e *%s, w %s) error { return w.ApplyTo(e) },\n\t)",
			rec, value, et, kt, entryKey, ut, kt, updateKey, et, ut,
		))), nil
	case valueListType:
		if !a.comparable(uf.Type.TypeArgs[0], make(set.Set[string])) {
			return a.setOnly(uf, upd, rec), nil
		}
		a.data.AddExternal(patchImportPath)
		return present(
			check(fmt.Sprintf("patch.ApplyValueList(&%s, %s)", rec, value)),
		), nil
	case structType:
		su, st := a.render(uf.Type.TypeArgs[0]), a.render(uf.Type.TypeArgs[1])
		return present(check(fmt.Sprintf(
			"%s.ApplyTo(&%s, func(v *%s, w %s) error { return w.ApplyTo(v) })",
			upd, rec, st, su,
		))), nil
	}
	if a.updateUnion(uf.Type) {
		return present(check(fmt.Sprintf("%s.ApplyTo(&%s)", upd, rec))), nil
	}
	return present(fmt.Sprintf("\t%s = %s\n", rec, value)), nil
}

// setOnly returns the statement that applies a list of values that Go cannot compare,
// which takes only set.
func (a *applier) setOnly(uf resolution.Field, upd, rec string) string {
	a.data.AddExternal(errorsImportPath)
	a.data.AddExternal(validateImportPath)
	return fmt.Sprintf(`	if %s != nil {
		if %s.Insert != nil || %s.Remove != nil || %s.Move != nil {
			return errors.Wrap(validate.ErrValidation, "%s takes only set")
		}
		if %s.Set != nil {
			%s = %s.Set
		}
	}
`, upd, upd, upd, upd, casing.FieldSnake(uf.Name), upd, rec, upd)
}

// comparable reports whether Go compares values of ref with ==. visited guards
// against recursive types.
func (a *applier) comparable(ref resolution.TypeRef, visited set.Set[string]) bool {
	table := a.data.table
	if ref.IsTypeParam() || typemap.Nilable(ref, table) {
		return false
	}
	if arrays.IsArray(ref, table) {
		elem, ok := arrays.ElementType(ref, table)
		return ok && a.comparable(elem, visited)
	}
	typ, ok := ref.Resolve(table)
	if !ok || visited.Contains(typ.QualifiedName) {
		return true
	}
	visited.Add(typ.QualifiedName)
	fields := func(fs []resolution.Field) bool {
		return lo.EveryBy(fs, func(f resolution.Field) bool {
			return f.Optional && !typemap.Nilable(f.Type, table) ||
				a.comparable(f.Type, visited)
		})
	}
	switch form := typemap.UnwrapType(typ, table).Form.(type) {
	case resolution.StructForm:
		return fields(resolution.UnifiedFields(typ, table))
	case resolution.UnionForm:
		return lo.EveryBy(form.Variants, func(v resolution.UnionVariant) bool {
			return fields(resolution.UnifiedVariantFields(typ, v, table))
		})
	}
	return true
}

// operators reports whether ref holds the operators of a list or the update of a union.
func (a *applier) operators(ref resolution.TypeRef) bool {
	return ref.Name == listType || ref.Name == valueListType || a.updateUnion(ref)
}

// updateUnion reports whether ref names the update of a union.
func (a *applier) updateUnion(ref resolution.TypeRef) bool {
	typ, ok := ref.Resolve(a.data.table)
	if !ok || typ.Updates == "" {
		return false
	}
	_, isUnion := typ.Form.(resolution.UnionForm)
	return isUnion
}

// entryKey returns the expression for the key of the list entry v of type ref.
func (a *applier) entryKey(ref resolution.TypeRef, v string) (string, error) {
	typ, _, ok := resolution.Dealias(ref, a.data.table)
	if !ok {
		return "", errors.Newf("unresolved list entry %s", ref.Name)
	}
	fields := resolution.UnifiedFields(typ, a.data.table)
	selector := v
	if form, isUnion := typ.Form.(resolution.UnionForm); isUnion {
		base, ok := a.unionBase(typ)
		if !ok {
			return "", errors.Newf(
				"list entry %s has no base that holds its key", typ.QualifiedName,
			)
		}
		rendered := a.data.probeResolver.ResolveTypeRef(ref, a.data.ctx)
		if !base.shared && strings.Contains(rendered, ".") {
			return "", errors.Newf(
				"list entry %s needs the base helpers of another package", rendered,
			)
		}
		parent, _, _ := resolution.Dealias(form.Extends[0], a.data.table)
		fields = resolution.UnifiedFields(parent, a.data.table)
		selector = base.get(v, false)
	}
	key, ok := lo.Find(fields, func(f resolution.Field) bool {
		_, isKey := f.Domains["key"]
		return isKey && !f.Optional
	})
	if !ok {
		return "", errors.Newf("list entry %s has no key", typ.QualifiedName)
	}
	if form, isUnion := typ.Form.(resolution.UnionForm); isUnion {
		parent, _, _ := resolution.Dealias(form.Extends[0], a.data.table)
		return selector + "." + a.structSelector(parent, key.Name), nil
	}
	return selector + "." + a.structSelector(typ, key.Name), nil
}

// updateKey returns the expression for the key of the entry update v of type ref.
func (a *applier) updateKey(ref resolution.TypeRef, v string) (string, error) {
	if a.updateUnion(ref) {
		return v + ".Key()", nil
	}
	typ, ok := ref.Resolve(a.data.table)
	if !ok {
		return "", errors.Newf("unresolved entry update %s", ref.Name)
	}
	key, ok := lo.Find(
		typ.Form.(resolution.StructForm).Fields,
		func(f resolution.Field) bool {
			return !f.Optional
		},
	)
	if !ok {
		return "", errors.Newf("entry update %s holds no key", typ.QualifiedName)
	}
	return v + "." + naming.GetFieldName(key), nil
}

type defaultsData struct {
	update, target string
	// fields are the update fields whose absence the method checks.
	fields []resolution.Field
	// record are the fields of the target.
	record []resolution.Field
	// applies reports that the target has an ApplyDefaults method.
	applies bool
}

// defaults writes the Defaults method of an update struct: the value of the target
// that an update merges into when there is none.
func (a *applier) defaults(b *strings.Builder, d defaultsData) {
	fmt.Fprintf(b, `
// Defaults returns the value that u merges into when there is none, which holds the
// defaults of the schema. It returns an error wrapping validate.ErrValidation when u
// leaves out a field that has no default.
func (u %s) Defaults() (%s, error) {
`, d.update, d.target)
	for _, uf := range d.fields {
		rf, ok := lo.Find(
			d.record,
			func(f resolution.Field) bool { return f.Name == uf.Name },
		)
		if !ok || !uf.Optional || rf.Optional || rf.Default != nil {
			continue
		}
		a.data.AddExternal(errorsImportPath)
		a.data.AddExternal(validateImportPath)
		fmt.Fprintf(b, `	if u.%s == nil {
		return %s{}, errors.Wrap(validate.ErrValidation, "%s is required")
	}
`, naming.GetFieldName(uf), d.target, casing.FieldSnake(uf.Name))
	}
	fmt.Fprintf(b, "\tvar x %s\n", d.target)
	if d.applies {
		b.WriteString("\tx.ApplyDefaults()\n")
	}
	b.WriteString("\treturn x, nil\n}\n")
}

// hasApplyDefaults reports whether the Go type that ref names has an ApplyDefaults
// method.
func (a *applier) hasApplyDefaults(ref resolution.TypeRef) bool {
	return resolvesToMethodType(ref, a.data) &&
		typeNeedsMethod(ref, a.data, make(set.Set[string]), defaultsHasOwn, neverSkip)
}

// shape is the layout of a Go struct: the fields it declares and the structs it
// embeds, in order.
type shape struct {
	declared []resolution.Field
	embeds   []resolution.TypeRef
}

// structShape returns the shape of the Go struct of typ.
func (a *applier) structShape(typ resolution.Type) shape {
	table := a.data.table
	form := typ.Form.(resolution.StructForm)
	if len(form.Extends) == 0 || !CanEmbed(form, table) {
		return shape{declared: resolution.UnifiedFields(typ, table)}
	}
	defaultOnly := resolver.DefaultOnlyOverrides(form.Extends, form.Fields, table)
	return shape{
		declared: lo.Reject(form.Fields, func(f resolution.Field, _ int) bool {
			return defaultOnly.Contains(f.Name)
		}),
		embeds: form.Extends,
	}
}

// variantShape returns the shape of the Go struct of the variant v of the union record.
func (a *applier) variantShape(
	record resolution.Type,
	v resolution.UnionVariant,
) shape {
	table := a.data.table
	form := record.Form.(resolution.UnionForm)
	payload, ok := v.Type.Resolve(table)
	if !ok || !v.Inline {
		return shape{embeds: append(slices.Clone(form.Extends), v.Type)}
	}
	inherited, declared := resolver.VariantBases(form, v, "go", table)
	fields := payload.Form.(resolution.StructForm).Fields
	defaultOnly := resolver.DefaultOnlyOverrides(inherited, fields, table)
	return shape{
		declared: append(
			declared,
			lo.Reject(fields, func(f resolution.Field, _ int) bool {
				return defaultOnly.Contains(f.Name)
			})...,
		),
		embeds: inherited,
	}
}

// step is one selector of a path through embedded structs, with the shape it reaches.
type step struct {
	name  string
	shape shape
}

// embedded returns the selector and the shape of each struct that sh embeds.
func (a *applier) embedded(sh shape) []step {
	steps := make([]step, 0, len(sh.embeds))
	for _, ref := range sh.embeds {
		parent, ok := ref.Resolve(a.data.table)
		target, _, dealiased := resolution.Dealias(ref, a.data.table)
		if !ok || !dealiased {
			continue
		}
		rendered := resolveExtendsType(ref, parent, probeData(a.data))
		steps = append(
			steps,
			step{name: embedFieldName(rendered), shape: a.structShape(target)},
		)
	}
	return steps
}

// path returns the selectors from a value of sh to the field name, through the structs
// that hold it. The first struct to hold the field wins, as in
// resolution.UnifiedFields.
func (a *applier) path(sh shape, name string) ([]step, bool) {
	if f, ok := lo.Find(sh.declared, func(f resolution.Field) bool {
		return f.Name == name
	}); ok {
		return []step{{name: naming.GetFieldName(f)}}, true
	}
	for _, e := range a.embedded(sh) {
		if rest, ok := a.path(e.shape, name); ok {
			return append([]step{e}, rest...), true
		}
	}
	return nil, false
}

// depths counts the Go fields of sh named goName at each embedding depth.
func (a *applier) depths(sh shape, goName string, depth int, counts map[int]int) {
	for _, f := range sh.declared {
		if naming.GetFieldName(f) == goName {
			counts[depth]++
		}
	}
	for _, e := range a.embedded(sh) {
		if e.name == goName {
			counts[depth]++
		}
		a.depths(e.shape, goName, depth+1, counts)
	}
}

// selector returns the shortest selector of the field name in a value of sh. Go
// promotes a field of an embedded struct unless a shallower field takes its name, as
// an embedded struct named for its own field does.
func (a *applier) selector(sh shape, name string) string {
	full, ok := a.path(sh, name)
	if !ok {
		panic(fmt.Sprintf("no struct of the selector holds the field %s", name))
	}
	last := len(full) - 1
	field := full[last].name
	for i := 0; i < last; i++ {
		counts := make(map[int]int)
		a.depths(sh, field, 0, counts)
		if shallowest := lo.Min(lo.Keys(counts)); shallowest == last-i &&
			counts[shallowest] == 1 {
			prefix := lo.Map(full[:i], func(s step, _ int) string { return s.name })
			return strings.Join(append(prefix, field), ".")
		}
		sh = full[i].shape
	}
	return strings.Join(lo.Map(full, func(s step, _ int) string { return s.name }), ".")
}

// structSelector returns the selector of the field name in a value of the struct typ.
func (a *applier) structSelector(typ resolution.Type, name string) string {
	return a.selector(a.structShape(typ), name)
}

// variantSelector returns the selector of the field name in a value of the variant v
// of the union record.
func (a *applier) variantSelector(
	record resolution.Type,
	v resolution.UnionVariant,
	name string,
) string {
	return a.selector(a.variantShape(record, v), name)
}

func (a *applier) render(ref resolution.TypeRef) string {
	return a.data.resolver.ResolveTypeRef(ref, a.data.ctx)
}

func indentLines(s string) string {
	lines := strings.SplitAfter(s, "\n")
	for i, l := range lines {
		if l != "" && l != "\n" {
			lines[i] = "\t" + l
		}
	}
	return strings.Join(lines, "")
}
