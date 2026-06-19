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
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

const (
	validateImportPath = "github.com/synnaxlabs/x/validate"
	strconvImportPath  = "strconv"
)

// defaultFillData describes a single field assignment emitted by a generated
// ApplyDefaults method: when the field equals ZeroLit it is set to Expr.
type defaultFillData struct {
	GoName  string
	ZeroLit string
	Expr    string
}

// enumCheckData describes a single enum-membership assertion emitted by a generated
// Validate method.
type enumCheckData struct {
	GoName    string
	FieldName string
}

// goDefaultFill returns the fill for a required field whose static default differs
// from its type's zero value (RFC 0043 section 5). It returns ok=false for fields with
// no default, nullable/optional fields, defaults that equal the zero value (nothing to
// fill), and non-static defaults such as `create`/`now`.
func goDefaultFill(field resolution.Field, data *templateData) (defaultFillData, bool) {
	if field.Default == nil || field.Optional {
		return defaultFillData{}, false
	}
	name := naming.GetFieldName(field)
	d := field.Default
	switch d.Kind {
	case resolution.ValueKindString:
		if d.StringValue == "" {
			return defaultFillData{}, false
		}
		return defaultFillData{GoName: name, ZeroLit: `""`, Expr: strconv.Quote(d.StringValue)}, true
	case resolution.ValueKindInt:
		if d.IntValue == 0 {
			return defaultFillData{}, false
		}
		return defaultFillData{GoName: name, ZeroLit: "0", Expr: fmt.Sprintf("%d", d.IntValue)}, true
	case resolution.ValueKindFloat:
		if d.FloatValue == 0 {
			return defaultFillData{}, false
		}
		return defaultFillData{GoName: name, ZeroLit: "0", Expr: strconv.FormatFloat(d.FloatValue, 'g', -1, 64)}, true
	case resolution.ValueKindIdent:
		ev, ok := validation.ResolveEnumVariant(d.IdentValue, field.Type, data.table)
		if !ok {
			return defaultFillData{}, false
		}
		form, _ := ev.Type.Form.(resolution.EnumForm)
		if form.IsIntEnum {
			// A valid integer-enum default is the zeroth member (the zero value), so
			// there is nothing to fill; a non-zeroth default is an invariant violation.
			return defaultFillData{}, false
		}
		enumType := stripPointer(data.resolver.ResolveTypeRef(field.Type, data.ctx))
		return defaultFillData{
			GoName:  name,
			ZeroLit: `""`,
			Expr:    enumType + naming.ToPascalCase(ev.Variant.Name),
		}, true
	}
	return defaultFillData{}, false
}

// goEnumCheck returns an enum-membership validation for a required field whose type is
// an enum (RFC 0043 section 5.2). Nullable and optional fields are skipped: their
// pointer may be nil, and absence is legitimate.
func goEnumCheck(field resolution.Field, data *templateData) (enumCheckData, bool) {
	if field.Optional {
		return enumCheckData{}, false
	}
	resolved, ok := field.Type.Resolve(data.table)
	if !ok {
		return enumCheckData{}, false
	}
	form, ok := resolved.Form.(resolution.EnumForm)
	if !ok {
		return enumCheckData{}, false
	}
	// Int enums do not generate an IsValid method (they rely on stringer, and their
	// zero value is a valid first member), so emitting a membership check would
	// reference a method that does not exist. Only string enums, whose zero value
	// (the empty string) is never a valid member, carry a check.
	if form.IsIntEnum {
		return enumCheckData{}, false
	}
	return enumCheckData{GoName: naming.GetFieldName(field), FieldName: field.Name}, true
}

// stripPointer removes a leading pointer marker from a resolved Go type.
func stripPointer(goType string) string { return strings.TrimPrefix(goType, "*") }

// recurseKind describes how a generated ApplyDefaults or Validate method reaches into a
// field whose type carries its own method: a direct value, a nilable pointer, the
// elements of a slice/array, or the values of a map.
type recurseKind string

const (
	recurseValue   recurseKind = "value"
	recursePointer recurseKind = "pointer"
	recurseSlice   recurseKind = "slice"
	recurseMap     recurseKind = "map"
)

// recurseStepData describes one nested call a generated method makes into a field (or an
// embedded type) whose own type carries an ApplyDefaults/Validate method. JSONName is the
// wire field name used as the Validate error path segment; it is empty for an embedded
// type, whose fields are promoted to the embedder's level and so take no path segment.
type recurseStepData struct {
	GoName   string
	JSONName string
	Kind     recurseKind
}

// fieldHasOwn reports whether a field is itself a reason for its struct to emit a method,
// independent of any nested type: a fillable static default for ApplyDefaults, an enum
// membership check for Validate.
type fieldHasOwn func(resolution.Field, *templateData) bool

func defaultsHasOwn(f resolution.Field, data *templateData) bool {
	_, ok := goDefaultFill(f, data)
	return ok
}

func validateHasOwn(f resolution.Field, data *templateData) bool {
	_, ok := goEnumCheck(f, data)
	return ok
}

// resolvesToMethodType reports whether ref resolves directly to a struct or union — a
// type that can carry an ApplyDefaults/Validate method — rather than a primitive, a
// builtin container (Array/Map), or a type parameter.
func resolvesToMethodType(ref resolution.TypeRef, data *templateData) bool {
	if ref.IsTypeParam() || ref.Name == "Array" || ref.Name == "Map" {
		return false
	}
	resolved, ok := ref.Resolve(data.table)
	if !ok {
		return false
	}
	switch resolved.Form.(type) {
	case resolution.StructForm, resolution.UnionForm:
		return true
	}
	return false
}

// typeNeedsMethod reports whether the type named by ref emits a recursive method: it has
// a field satisfying hasOwn, or a nested struct field, slice/array element, map value, or
// union variant payload that (transitively) does. Generic types and type parameters never
// emit a method. visited guards against cycles in recursive types.
func typeNeedsMethod(
	ref resolution.TypeRef,
	data *templateData,
	visited set.Set[string],
	hasOwn fieldHasOwn,
) bool {
	if ref.IsTypeParam() {
		return false
	}
	switch ref.Name {
	case "Array":
		return len(ref.TypeArgs) == 1 &&
			typeNeedsMethod(ref.TypeArgs[0], data, visited, hasOwn)
	case "Map":
		return len(ref.TypeArgs) == 2 &&
			typeNeedsMethod(ref.TypeArgs[1], data, visited, hasOwn)
	}
	if visited.Contains(ref.Name) {
		return false
	}
	visited.Add(ref.Name)
	resolved, ok := ref.Resolve(data.table)
	if !ok {
		return false
	}
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		if form.IsGeneric() {
			return false
		}
		for _, field := range resolution.UnifiedFields(resolved, data.table) {
			if hasOwn(field, data) ||
				typeNeedsMethod(field.Type, data, visited, hasOwn) {
				return true
			}
		}
	case resolution.UnionForm:
		for _, variant := range form.Variants {
			if typeNeedsMethod(variant.Type, data, visited, hasOwn) {
				return true
			}
		}
	case resolution.AliasForm:
		return typeNeedsMethod(form.Target, data, visited, hasOwn)
	case resolution.DistinctForm:
		return typeNeedsMethod(form.Base, data, visited, hasOwn)
	}
	return false
}

// hasSliceRecurse reports whether any step iterates a slice or array, which requires the
// strconv import for the index path segment in a generated Validate method.
func hasSliceRecurse(steps []recurseStepData) bool {
	for _, step := range steps {
		if step.Kind == recurseSlice {
			return true
		}
	}
	return false
}

// goRecurseStep returns the nested-method step for a field whose type (transitively)
// needs the method identified by hasOwn, or ok=false when the field needs no recursion.
func goRecurseStep(
	field resolution.Field,
	data *templateData,
	hasOwn fieldHasOwn,
) (recurseStepData, bool) {
	ref := field.Type
	step := recurseStepData{
		GoName:   naming.GetFieldName(field),
		JSONName: casing.FieldSnake(field.Name),
	}
	needs := func(r resolution.TypeRef) bool {
		return resolvesToMethodType(r, data) &&
			typeNeedsMethod(r, data, set.New[string](), hasOwn)
	}
	switch ref.Name {
	case "Array":
		if len(ref.TypeArgs) == 1 && needs(ref.TypeArgs[0]) {
			step.Kind = recurseSlice
			return step, true
		}
		return recurseStepData{}, false
	case "Map":
		if len(ref.TypeArgs) == 2 && needs(ref.TypeArgs[1]) {
			step.Kind = recurseMap
			return step, true
		}
		return recurseStepData{}, false
	}
	if !needs(ref) {
		return recurseStepData{}, false
	}
	step.Kind = recurseValue
	if field.Optional {
		step.Kind = recursePointer
	}
	return step, true
}
