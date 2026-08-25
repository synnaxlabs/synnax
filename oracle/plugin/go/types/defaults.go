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
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
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

// goDefaultFills returns the fills for a field's static default. A scalar default
// yields a single fill; a struct-literal default (e.g. `x1 Axis = { key = AxisKeyX1 }`)
// yields one fill per non-zero leaf component, keyed by a nested selector (X1.Key). It
// returns nil for fields with no default, nullable/optional fields, defaults that equal
// the zero value (nothing to fill), and non-static defaults such as create/now.
func goDefaultFills(field resolution.Field, data *templateData) []defaultFillData {
	if field.Default == nil || field.Optional {
		return nil
	}
	name := naming.GetFieldName(field)
	if field.Default.Kind == resolution.ValueKindStruct {
		return structDefaultFills(name, field.Type, *field.Default, data)
	}
	if fill, ok := componentFill(name, field.Type, field.Default, data); ok {
		return []defaultFillData{fill}
	}
	return nil
}

// componentFill returns the fill for a scalar or array default, dispatching on kind.
func componentFill(
	goName string,
	typeRef resolution.TypeRef,
	d *resolution.ExpressionValue,
	data *templateData,
) (defaultFillData, bool) {
	if d.Kind == resolution.ValueKindArray {
		return arrayFill(goName, typeRef, d, data)
	}
	return scalarFill(goName, typeRef, d, data)
}

// structDefaultFills walks a struct-literal default, emitting a fill per non-zero leaf
// component. prefix is the Go selector to the struct field (e.g. "X1"), structRef its
// type, and val the literal. Nested struct components recurse with an extended
// selector.
func structDefaultFills(
	prefix string,
	structRef resolution.TypeRef,
	val resolution.ExpressionValue,
	data *templateData,
) []defaultFillData {
	resolved, ok := structRef.Resolve(data.table)
	if !ok {
		return nil
	}
	form, ok := resolved.Form.(resolution.StructForm)
	if !ok {
		return nil
	}
	var fills []defaultFillData
	for _, comp := range val.Fields {
		f, ok := form.Field(comp.Name)
		if !ok {
			continue
		}
		selector := prefix + "." + naming.GetFieldName(f)
		if comp.Value.Kind == resolution.ValueKindStruct {
			fills = append(
				fills,
				structDefaultFills(selector, f.Type, comp.Value, data)...)
			continue
		}
		if fill, ok := componentFill(selector, f.Type, &comp.Value, data); ok {
			fills = append(fills, fill)
		}
	}
	return fills
}

// arrayFill returns the fill assigning the array default d to the Go selector goName of
// slice type typeRef. It returns ok=false when d is empty, since the nil slice the
// caller starts from already carries that value, and when an element has no Go literal.
func arrayFill(
	goName string,
	typeRef resolution.TypeRef,
	d *resolution.ExpressionValue,
	data *templateData,
) (defaultFillData, bool) {
	if len(d.Elements) == 0 || typeRef.Name != "Array" || len(typeRef.TypeArgs) != 1 {
		return defaultFillData{}, false
	}
	lits := make([]string, 0, len(d.Elements))
	for i := range d.Elements {
		lit, ok := goLiteral(typeRef.TypeArgs[0], &d.Elements[i], data)
		if !ok {
			return defaultFillData{}, false
		}
		lits = append(lits, lit)
	}
	return defaultFillData{
		GoName:  goName,
		ZeroLit: "nil",
		Expr: fmt.Sprintf(
			"%s{%s}",
			data.resolver.ResolveTypeRef(typeRef, data.ctx),
			strings.Join(lits, ", "),
		),
	}, true
}

// goLiteral renders v as a Go literal of type typeRef. Unlike scalarFill it does not
// suppress the zero value, because an element of an array default must be written out
// even when it is zero. It returns ok=false for a kind with no literal rendering, such
// as a struct or a magic default like create.
func goLiteral(
	typeRef resolution.TypeRef,
	v *resolution.ExpressionValue,
	data *templateData,
) (string, bool) {
	switch v.Kind {
	case resolution.ValueKindString:
		return strconv.Quote(v.StringValue), true
	case resolution.ValueKindInt:
		return fmt.Sprintf("%d", v.IntValue), true
	case resolution.ValueKindFloat:
		return strconv.FormatFloat(v.FloatValue, 'g', -1, 64), true
	case resolution.ValueKindBool:
		return strconv.FormatBool(v.BoolValue), true
	case resolution.ValueKindIdent:
		ev, ok := validation.ResolveEnumVariant(v.IdentValue, typeRef, data.table)
		if !ok {
			return "", false
		}
		enumType := stripPointer(data.resolver.ResolveTypeRef(typeRef, data.ctx))
		return enumType + naming.ToPascalCase(ev.Variant.Name), true
	}
	return "", false
}

// scalarFill returns the fill assigning the scalar or enum default d to the Go selector
// goName of type typeRef. It returns ok=false when d equals the type's zero value
// (nothing to fill) or is an integer-enum default (whose zeroth member is the zero
// value).
func scalarFill(
	goName string,
	typeRef resolution.TypeRef,
	d *resolution.ExpressionValue,
	data *templateData,
) (defaultFillData, bool) {
	switch d.Kind {
	case resolution.ValueKindString:
		if d.StringValue == "" {
			return defaultFillData{}, false
		}
		return defaultFillData{
			GoName:  goName,
			ZeroLit: `""`,
			Expr:    strconv.Quote(d.StringValue),
		}, true
	case resolution.ValueKindInt:
		if d.IntValue == 0 {
			return defaultFillData{}, false
		}
		return defaultFillData{
			GoName:  goName,
			ZeroLit: "0",
			Expr:    fmt.Sprintf("%d", d.IntValue),
		}, true
	case resolution.ValueKindFloat:
		if d.FloatValue == 0 {
			return defaultFillData{}, false
		}
		return defaultFillData{
			GoName:  goName,
			ZeroLit: "0",
			Expr:    strconv.FormatFloat(d.FloatValue, 'g', -1, 64),
		}, true
	case resolution.ValueKindIdent:
		if uv, ok := validation.ResolveUnionVariant(
			d.IdentValue,
			typeRef,
			data.table,
		); ok {
			// The wrapper holds the active variant behind a nil-able interface, so
			// the fill targets that field rather than the wrapper itself.
			unionType := stripPointer(data.resolver.ResolveTypeRef(typeRef, data.ctx))
			return defaultFillData{
				GoName:  goName + ".Variant",
				ZeroLit: "nil",
				Expr: casing.QualifiedVariantTypeName(
					unionType,
					uv.Variant.Name,
					".",
				) + "{}",
			}, true
		}
		ev, ok := validation.ResolveEnumVariant(d.IdentValue, typeRef, data.table)
		if !ok {
			return defaultFillData{}, false
		}
		form, _ := ev.Type.Form.(resolution.EnumForm)
		if form.IsIntEnum {
			// A valid integer-enum default is the zeroth member (the zero value), so
			// there is nothing to fill; a non-zeroth default is an invariant violation.
			return defaultFillData{}, false
		}
		enumType := stripPointer(data.resolver.ResolveTypeRef(typeRef, data.ctx))
		return defaultFillData{
			GoName:  goName,
			ZeroLit: `""`,
			Expr:    enumType + naming.ToPascalCase(ev.Variant.Name),
		}, true
	}
	return defaultFillData{}, false
}

// constraintCheckData describes a single @validate constraint assertion emitted by a
// generated Validate method.
type constraintCheckData struct {
	GoName    string
	FieldName string
	// Kind selects the assertion form: non_empty_string, min_len, max_len (string), or
	// ge, le (numeric).
	Kind string
	// Arg is the length or numeric threshold literal for the kinds that take one.
	Arg string
}

// goConstraintChecks returns the @validate constraint assertions for a field,
// classified against the field type's underlying primitive so a distinct numeric type
// (e.g. a Key over uint32) validates as a number. Optional fields are skipped: a bound
// on an absent value is ambiguous.
func goConstraintChecks(
	field resolution.Field,
	data *templateData,
) []constraintCheckData {
	if field.Optional {
		return nil
	}
	domain, ok := field.Domains["validate"]
	if !ok {
		return nil
	}
	rules := validation.Parse(domain)
	if validation.IsEmpty(rules) {
		return nil
	}
	name := naming.GetFieldName(field)
	jsonName := casing.FieldSnake(field.Name)
	base := resolution.PrimitiveBase(field.Type, data.table)
	check := func(kind, arg string) constraintCheckData {
		return constraintCheckData{
			GoName:    name,
			FieldName: jsonName,
			Kind:      kind,
			Arg:       arg,
		}
	}
	var checks []constraintCheckData
	if resolution.IsStringPrimitive(base) {
		switch {
		case rules.Required, rules.MinLength != nil && *rules.MinLength <= 1:
			checks = append(checks, check("non_empty_string", ""))
		case rules.MinLength != nil:
			checks = append(
				checks,
				check("min_len", strconv.FormatInt(*rules.MinLength, 10)),
			)
		}
		if rules.MaxLength != nil {
			checks = append(
				checks,
				check("max_len", strconv.FormatInt(*rules.MaxLength, 10)),
			)
		}
	}
	if resolution.IsNumberPrimitive(base) {
		if rules.Required {
			checks = append(checks, check("non_zero", ""))
		}
		if rules.Min != nil {
			checks = append(checks, check("ge", numberLiteral(rules.Min)))
		}
		if rules.Max != nil {
			checks = append(checks, check("le", numberLiteral(rules.Max)))
		}
	}
	return checks
}

// numberLiteral renders a numeric constraint value as a Go literal.
func numberLiteral(n *validation.Number) string {
	if n.IsInt {
		return strconv.FormatInt(n.Int, 10)
	}
	return strconv.FormatFloat(n.Float, 'g', -1, 64)
}

// goEnumCheck returns an enum-membership validation for a required field whose type is
// an enum. Nullable and optional fields are skipped: their pointer may be nil, and
// absence is legitimate.
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
	return enumCheckData{
		GoName:    naming.GetFieldName(field),
		FieldName: field.Name,
	}, true
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

// recurseStepData describes one nested call a generated method makes into a field (or
// an embedded type) whose own type carries an ApplyDefaults/Validate method. JSONName
// is the wire field name used as the Validate error path segment; it is empty for an
// embedded type, whose fields are promoted to the embedder's level and so take no path
// segment.
type recurseStepData struct {
	GoName   string
	JSONName string
	Kind     recurseKind
}

// fieldHasOwn reports whether a field is itself a reason for its struct to emit a
// method, independent of any nested type: a fillable static default for ApplyDefaults,
// an enum membership check for Validate.
type fieldHasOwn func(resolution.Field, *templateData) bool

func defaultsHasOwn(f resolution.Field, data *templateData) bool {
	return len(goDefaultFills(f, data.probe())) > 0
}

func validateHasOwn(f resolution.Field, data *templateData) bool {
	probe := data.probe()
	if _, ok := goEnumCheck(f, probe); ok {
		return true
	}
	return len(goConstraintChecks(f, probe)) > 0
}

// neverSkip is the skip predicate for ApplyDefaults: defaults apply to every field.
func neverSkip(resolution.Field, *templateData) bool { return false }

// validateSkip reports whether a field carries `@validate skip`, excluding it from
// generated validation and any recursion into it. Used for reference fields (a parent
// key, label keys) that hold a stub rather than embedded data.
func validateSkip(f resolution.Field, _ *templateData) bool {
	domain, ok := f.Domains["validate"]
	if !ok {
		return false
	}
	return validation.Parse(domain).Skip
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

// typeNeedsMethod reports whether the type named by ref emits a recursive method: it
// has a field satisfying hasOwn, or a nested struct field, slice/array element, map
// value, or union variant payload that (transitively) does. Generic types and type
// parameters never emit a method. visited guards against cycles in recursive types.
func typeNeedsMethod(
	ref resolution.TypeRef,
	data *templateData,
	visited set.Set[string],
	hasOwn fieldHasOwn,
	skip fieldHasOwn,
) bool {
	if ref.IsTypeParam() {
		return false
	}
	switch ref.Name {
	case "Array":
		return len(ref.TypeArgs) == 1 &&
			typeNeedsMethod(ref.TypeArgs[0], data, visited, hasOwn, skip)
	case "Map":
		return len(ref.TypeArgs) == 2 &&
			typeNeedsMethod(ref.TypeArgs[1], data, visited, hasOwn, skip)
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
			if skip(field, data) {
				continue
			}
			if hasOwn(field, data) ||
				typeNeedsMethod(field.Type, data, visited, hasOwn, skip) {
				return true
			}
		}
	case resolution.UnionForm:
		for _, variant := range form.Variants {
			if typeNeedsMethod(variant.Type, data, visited, hasOwn, skip) {
				return true
			}
		}
	case resolution.AliasForm:
		return typeNeedsMethod(form.Target, data, visited, hasOwn, skip)
	case resolution.DistinctForm:
		return typeNeedsMethod(form.Base, data, visited, hasOwn, skip)
	}
	return false
}

// hasSliceRecurse reports whether any step iterates a slice or array, which requires
// the strconv import for the index path segment in a generated Validate method.
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
	skip fieldHasOwn,
) (recurseStepData, bool) {
	ref := field.Type
	step := recurseStepData{
		GoName:   naming.GetFieldName(field),
		JSONName: casing.FieldSnake(field.Name),
	}
	needs := func(r resolution.TypeRef) bool {
		return resolvesToMethodType(r, data) &&
			typeNeedsMethod(r, data, set.New[string](), hasOwn, skip)
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

// defaultGroupData describes a set of fields whose defaults apply as a unit. The
// generated ApplyDefaults fills every member only when all of them hold their zero
// value, so a deliberate zero on any member survives the fill.
type defaultGroupData struct {
	// Members is every field in the group, rendered into the all-zero guard.
	Members []defaultFillData
	// Fills is the subset of members carrying a non-zero default, the only ones with
	// anything to assign.
	Fills []defaultFillData
}

// defaultGroupName returns the group a field's default belongs to, or "" when the
// field fills on its own.
func defaultGroupName(f resolution.Field) string {
	return domain.GetStringFromField(f, "default", "group")
}

// goDefaultGroups returns one entry per `@default group` declared across fields, in
// first-declaration order. A group whose members all default to their zero value has
// nothing to assign and is dropped.
func goDefaultGroups(
	fields []resolution.Field,
	data *templateData,
) []defaultGroupData {
	var order []string
	byName := make(map[string]*defaultGroupData)
	for _, f := range fields {
		name := defaultGroupName(f)
		if name == "" || f.Default == nil || f.Optional {
			continue
		}
		member, ok := groupMember(f, data)
		if !ok {
			continue
		}
		g, seen := byName[name]
		if !seen {
			g = &defaultGroupData{}
			byName[name] = g
			order = append(order, name)
		}
		g.Members = append(g.Members, member)
		g.Fills = append(g.Fills, goDefaultFills(f, data)...)
	}
	groups := make([]defaultGroupData, 0, len(order))
	for _, name := range order {
		if g := byName[name]; len(g.Fills) > 0 {
			groups = append(groups, *g)
		}
	}
	return groups
}

// groupMember returns the zero comparison for one group member. It returns ok=false
// for a default kind with no zero literal, which checkDefaultGroups rejects from a
// group.
func groupMember(
	f resolution.Field,
	data *templateData,
) (defaultFillData, bool) {
	name := naming.GetFieldName(f)
	switch f.Default.Kind {
	case resolution.ValueKindString:
		return defaultFillData{GoName: name, ZeroLit: `""`}, true
	case resolution.ValueKindInt, resolution.ValueKindFloat:
		return defaultFillData{GoName: name, ZeroLit: "0"}, true
	case resolution.ValueKindIdent:
		ev, ok := validation.ResolveEnumVariant(
			f.Default.IdentValue,
			f.Type,
			data.table,
		)
		if !ok {
			return defaultFillData{}, false
		}
		form, isEnum := ev.Type.Form.(resolution.EnumForm)
		if !isEnum || form.IsIntEnum {
			return defaultFillData{}, false
		}
		return defaultFillData{GoName: name, ZeroLit: `""`}, true
	}
	return defaultFillData{}, false
}
