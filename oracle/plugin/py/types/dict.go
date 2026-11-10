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
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// dictExpr is the @py expression that renders a struct or union as a TypedDict, with
// an absent key for an absent optional field.
const dictExpr = "dict"

// isDict reports whether typ renders as a TypedDict.
func isDict(typ resolution.Type) bool {
	return domain.HasExprFromType(typ, "py", dictExpr)
}

// dictForms returns the generic structs that a TypedDict holds, directly or through
// another such struct. Each gets a <Name>Dict form beside its model, and a TypedDict
// refers to it by that form.
func dictForms(table *resolution.Table) set.Set[string] {
	forms := make(set.Set[string])
	var walk func(resolution.TypeRef)
	walk = func(ref resolution.TypeRef) {
		for _, arg := range ref.TypeArgs {
			walk(arg)
		}
		typ, ok := table.Get(ref.Name)
		if !ok || forms.Contains(typ.QualifiedName) || isDict(typ) ||
			omit.IsSkipped(typ, "py") {
			return
		}
		form, isStruct := typ.Form.(resolution.StructForm)
		if !isStruct || len(form.TypeParams) == 0 {
			return
		}
		forms.Add(typ.QualifiedName)
		for _, f := range resolution.UnifiedFields(typ, table) {
			walk(f.Type)
		}
	}
	for _, typ := range table.Types {
		if !isDict(typ) {
			continue
		}
		for _, f := range dictFields(typ, table) {
			walk(f.Type)
		}
	}
	return forms
}

// dictFields returns the fields of a struct, or of every variant of a union.
func dictFields(typ resolution.Type, table *resolution.Table) []resolution.Field {
	form, ok := typ.Form.(resolution.UnionForm)
	if !ok {
		return resolution.UnifiedFields(typ, table)
	}
	return lo.FlatMap(
		form.Variants,
		func(v resolution.UnionVariant, _ int) []resolution.Field {
			return resolution.UnifiedVariantFields(typ, v, table)
		},
	)
}

// dictFieldsOf renders the fields of the struct typ as the keys of its dict form.
func dictFieldsOf(
	typ resolution.Type,
	table *resolution.Table,
	data *templateData,
	keyFields []keyFieldData,
) ([]fieldData, error) {
	data.dict = true
	defer func() { data.dict = false }()
	sd, err := processStruct(typ, table, data, keyFields)
	if err != nil {
		return nil, err
	}
	if lo.SomeBy(sd.Fields, func(f fieldData) bool { return f.IsRequired }) {
		data.imports.addTyping("Required")
	}
	data.imports.addTyping("TypedDict")
	return sd.Fields, nil
}

// addDictForm renders a struct marked dict as a TypedDict, and declares the dict form
// of a struct that a TypedDict holds after its model.
func addDictForm(
	typ resolution.Type,
	sd *structData,
	table *resolution.Table,
	data *templateData,
	keyFields []keyFieldData,
) error {
	if sd.Skip || sd.IsAlias {
		return nil
	}
	dict := isDict(typ)
	if !dict && !data.dictForms.Contains(typ.QualifiedName) {
		return nil
	}
	extends := typ.Form.(resolution.StructForm).Extends
	if sd.After != "" || (len(extends) > 0 && !dict) {
		return errors.Newf(
			"%s: a dict form needs a struct with no parent and no filter input",
			typ.QualifiedName,
		)
	}
	if err := checkDictParents(typ, table); err != nil {
		return err
	}
	fields, err := dictFieldsOf(typ, table, data, keyFields)
	if err != nil {
		return err
	}
	if dict {
		sd.IsDict, sd.Fields = true, fields
		sd.HasAlias = lo.SomeBy(fields, func(f fieldData) bool { return f.Alias != "" })
		if sd.HasAlias && len(extends) > 0 {
			return errors.Newf(
				"%s: a TypedDict with a parent cannot hold a key named for a keyword",
				typ.QualifiedName,
			)
		}
		if sd.HasAlias {
			sd.After = typedDict(sd.PyName, fields, typeParamNames(sd))
		}
		return nil
	}
	name := sd.PyName + "Dict"
	sd.After = "# " + name + " is the dict form of " + sd.PyName + ".\n" +
		typedDict(name, fields, typeParamNames(sd))
	return nil
}

// checkDictParents returns an error when a parent of the TypedDict typ is not a
// TypedDict, or when typ omits a field that a parent holds, which a TypedDict cannot
// drop.
func checkDictParents(typ resolution.Type, table *resolution.Table) error {
	form := typ.Form.(resolution.StructForm)
	for _, ref := range form.Extends {
		parent, _, ok := resolution.Dealias(ref, table)
		if !ok || !isDict(parent) {
			return errors.Newf(
				"%s: a TypedDict can only extend a TypedDict, not %s",
				typ.QualifiedName, ref.Name,
			)
		}
		for _, f := range resolution.UnifiedFields(parent, table) {
			if lo.Contains(form.OmittedFields, f.Name) {
				return errors.Newf(
					"%s: a TypedDict cannot omit the field %s of %s",
					typ.QualifiedName, f.Name, parent.QualifiedName,
				)
			}
		}
	}
	return nil
}

func typeParamNames(sd *structData) []string {
	return lo.Map(sd.TypeParams, func(p typeParamData, _ int) string { return p.Name })
}
