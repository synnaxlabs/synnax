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

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// initParamData is one keyword parameter of a model's type-checking __init__.
type initParamData struct {
	// Name is the parameter name, escaped like its field.
	Name string
	// PyType is the annotation of the values the parameter accepts.
	PyType string
	// Required is true when the field has no default, so the caller must pass it.
	Required bool
}

// buildInitParams returns type-checking __init__ parameters that accept the base
// primitive of each hand-written distinct field, or nil when none are needed. The
// receiver is _self because a field may be named self.
func buildInitParams(
	fields []resolution.Field,
	table *resolution.Table,
	data *templateData,
) []initParamData {
	if !lo.SomeBy(fields, func(f resolution.Field) bool {
		_, ok := handPrimitive(f, table)
		return ok
	}) {
		return nil
	}
	data.imports.addTyping("TYPE_CHECKING")
	scratch := &templateData{
		Namespace:  data.Namespace,
		OutputPath: data.OutputPath,
		imports:    newImportManager(),
	}
	params := make([]initParamData, 0, len(fields))
	for _, f := range fields {
		var pyType string
		if primitive, ok := handPrimitive(f, table); ok {
			pyType = primitiveToPython(primitive, data)
			if f.Optional {
				pyType += " | None"
			}
		} else {
			pyType = fieldPyType(f, table, data)
		}
		fd := processField(f, table, scratch, nil, nil)
		params = append(params, initParamData{
			Name:     fd.Name,
			PyType:   pyType,
			Required: fd.Required,
		})
	}
	return params
}

// handPrimitive returns the base primitive of a field typed directly as a hand-written
// distinct type with a primitive base, or false otherwise. Hand-written distinct types
// validate from their base primitive and subclass it.
func handPrimitive(field resolution.Field, table *resolution.Table) (string, bool) {
	if field.Type.IsTypeParam() || field.Type.Name == "Array" {
		return "", false
	}
	resolved, ok := field.Type.Resolve(table)
	if !ok || !omit.IsHand(resolved, "py") {
		return "", false
	}
	form, ok := resolved.Form.(resolution.DistinctForm)
	if !ok || !resolution.IsPrimitive(form.Base.Name) {
		return "", false
	}
	return form.Base.Name, true
}

// variantInitParams returns the type-checking __init__ parameters of a union variant
// model: its discriminator, then the fields it inherits and declares. The discriminator
// replaces any inherited field of the same name, as the variant class overrides it.
func variantInitParams(
	discName string,
	value string,
	bases []resolution.TypeRef,
	declared []resolution.Field,
	table *resolution.Table,
	data *templateData,
) []initParamData {
	synthetic := resolution.Type{
		Form: resolution.StructForm{Extends: bases, Fields: declared},
	}
	params := buildInitParams(pyFields(synthetic, table), table, data)
	if params == nil {
		return nil
	}
	params = lo.Reject(params, func(p initParamData, _ int) bool {
		return p.Name == discName
	})
	disc := initParamData{Name: discName, PyType: fmt.Sprintf("Literal[%q]", value)}
	return append([]initParamData{disc}, params...)
}

// pyFields returns every field of the struct's Python class. Unlike
// resolution.UnifiedFields, it keeps inherited fields that a subclass excludes from
// serialization, since the Python class still inherits and accepts them.
func pyFields(typ resolution.Type, table *resolution.Table) []resolution.Field {
	form, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return nil
	}
	if len(form.Extends) == 0 || resolver.HasDomainOmissions(form) {
		return resolution.UnifiedFields(typ, table)
	}
	fields := inheritedFields(form, table)
	// A flattened class drops excluded fields instead of inheriting them.
	if flattensParents(form, table) {
		fields = lo.Reject(fields, func(f resolution.Field, _ int) bool {
			return form.IsFieldOmitted(f.Name)
		})
	}
	index := make(map[string]int, len(fields))
	for i, f := range fields {
		index[f.Name] = i
	}
	for _, f := range form.Fields {
		if i, ok := index[f.Name]; ok {
			fields[i] = f
			continue
		}
		index[f.Name] = len(fields)
		fields = append(fields, f)
	}
	return fields
}

// inheritedFields returns the fields a struct's Python class takes from its parents,
// first parent first, typed as the class sees them. The class rendering and the
// type-checking __init__ both read them, so the two always agree.
func inheritedFields(
	form resolution.StructForm,
	table *resolution.Table,
) []resolution.Field {
	var fields []resolution.Field
	seen := make(set.Set[string])
	for _, ext := range form.Extends {
		parent, ok := ext.Resolve(table)
		if !ok {
			continue
		}
		typeArgs := pyTypeArgs(parent, ext)
		for _, f := range pyFields(parent, table) {
			if seen.Contains(f.Name) {
				continue
			}
			seen.Add(f.Name)
			// Substitution erases the optional type parameter that made it nullable.
			f.Optional = f.Optional || isOptionalTypeParam(f)
			f.Type = resolution.SubstituteTypeRef(f.Type, typeArgs)
			fields = append(fields, f)
		}
	}
	return fields
}

// pyTypeArgs maps each type parameter of parent to the type its Python subclass binds
// it to through ext. It mirrors the class rendering: a defaulted parameter is not a
// TypeVar and stays pinned to its default, buildExtendsExpr passes the arguments of
// the others, and an omitted argument, which only an optional parameter allows, falls
// back to the TypeVar's default.
func pyTypeArgs(
	parent resolution.Type,
	ext resolution.TypeRef,
) map[string]resolution.TypeRef {
	form, _ := parent.Form.(resolution.StructForm)
	args := make(map[string]resolution.TypeRef, len(form.TypeParams))
	for i, tp := range form.TypeParams {
		switch {
		case tp.HasDefault():
			args[tp.Name] = *tp.Default
		case i < len(ext.TypeArgs):
			args[tp.Name] = ext.TypeArgs[i]
		default:
			args[tp.Name] = typeVarDefault(tp)
		}
	}
	return args
}

// typeVarDefault returns the default of an optional type parameter's TypeVar: its
// bound, or arbitrary JSON when unbounded.
func typeVarDefault(tp resolution.TypeParam) resolution.TypeRef {
	if tp.Constraint != nil && !resolution.IsConstraint(tp.Constraint.Name) {
		return *tp.Constraint
	}
	return resolution.TypeRef{Name: "any"}
}

// flattensParents reports whether the struct generates as a standalone class instead
// of inheriting its parents. Python inheritance cannot express a domain removal or an
// optional override of a required parent field.
func flattensParents(form resolution.StructForm, table *resolution.Table) bool {
	if resolver.HasDomainOmissions(form) {
		return true
	}
	seen := make(set.Set[string])
	required := make(set.Set[string])
	for _, ext := range form.Extends {
		parent, ok := ext.Resolve(table)
		if !ok {
			continue
		}
		for _, pf := range resolution.UnifiedFields(parent, table) {
			if seen.Contains(pf.Name) {
				continue
			}
			seen.Add(pf.Name)
			if !pf.Optional {
				required.Add(pf.Name)
			}
		}
	}
	return lo.SomeBy(form.Fields, func(f resolution.Field) bool {
		return f.Optional && required.Contains(f.Name)
	})
}
