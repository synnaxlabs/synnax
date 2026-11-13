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
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

const (
	validateImportPath = "github.com/synnaxlabs/x/validate"
	strconvImportPath  = "strconv"
	msgpackImportPath  = "github.com/vmihailenco/msgpack/v5"
)

// defaultFillData describes a single field assignment emitted by a generated
// ApplyDefaults method: when the field equals ZeroLit, Setup runs and the field is set
// to Expr.
type defaultFillData struct {
	GoName  string
	ZeroLit string
	Expr    string
	// Setup holds the statements that build a union literal's variant in local
	// variables before Expr assigns it. Empty for every other default.
	Setup []string
	// SetOnly marks a fill that overrides a default SetDefaults set first.
	// ApplyDefaults skips it, since it fills only zero fields.
	SetOnly bool
	// ApplyOnly marks a fill that only ApplyDefaults assigns, since a SetOnly fill
	// replaces it in SetDefaults.
	ApplyOnly bool
}

// enumCheckData describes a single enum-membership assertion emitted by a generated
// Validate method.
type enumCheckData struct {
	GoName    string
	FieldName string
}

// goDefaultFills returns the fills for a field's static default. A scalar default
// yields a single fill; a struct-literal default (e.g. `x1 Axis = { key = AxisKeyX1 }`)
// yields one fill per non-zero leaf component, keyed by a nested selector (X1.Key), and
// a SetOnly fill for a zero leaf that overrides a nested default. It returns nil for
// fields with no default, nullable/optional fields, defaults that equal the zero value,
// and non-static defaults such as create/now.
func goDefaultFills(field resolution.Field, data *templateData) []defaultFillData {
	if field.Default == nil || field.Optional {
		return nil
	}
	name := naming.GetFieldName(field)
	if field.Default.Kind == resolution.ValueKindStruct {
		if fill, ok := unionLiteralFill(name, field.Type, *field.Default, data); ok {
			return []defaultFillData{fill}
		}
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
// component and a SetOnly fill per zero leaf that overrides a nested default. prefix is
// the Go selector to the struct field (e.g. "X1"), structRef its type, and val the
// literal. Nested struct components recurse with an extended selector.
func structDefaultFills(
	prefix string,
	structRef resolution.TypeRef,
	val resolution.ExpressionValue,
	data *templateData,
) []defaultFillData {
	resolved, _, ok := resolution.Dealias(structRef, data.table)
	if !ok {
		return nil
	}
	form, ok := resolved.Form.(resolution.StructForm)
	if !ok {
		return nil
	}
	// SetDefaults fills a type that sets defaults before these fills run, so a zero
	// leaf must be written out to override it.
	defaulted := typeSetsDefaults(structRef, data, set.New[string]())
	var fills []defaultFillData
	for _, comp := range val.Fields {
		f, ok := form.Field(comp.Name)
		if !ok {
			continue
		}
		selector := prefix + "." + naming.GetFieldName(f)
		if comp.Value.Kind == resolution.ValueKindStruct {
			if fill, ok := unionLiteralFill(selector, f.Type, comp.Value, data); ok {
				fills = append(fills, fill)
				continue
			}
			fills = append(
				fills,
				structDefaultFills(selector, f.Type, comp.Value, data)...,
			)
			continue
		}
		if fill, ok := componentFill(selector, f.Type, &comp.Value, data); ok {
			fills = append(fills, fill)
		} else if defaulted {
			if fill, ok := zeroFill(selector, comp.Value); ok {
				fills = append(fills, fill)
			}
		}
	}
	return fills
}

// overrideFills returns the fills of a field that restates an inherited field's
// default. SetDefaults sets the parent's default first, so a zero default must be
// written out.
func overrideFills(field resolution.Field, data *templateData) []defaultFillData {
	if fill, ok := structOverrideFill(field, data); ok {
		apply := goDefaultFills(field, data)
		for i := range apply {
			apply[i].ApplyOnly = true
		}
		return append(apply, fill)
	}
	if fills := goDefaultFills(field, data); len(fills) > 0 ||
		field.Default == nil || field.Optional {
		return fills
	}
	if fill, ok := zeroFill(naming.GetFieldName(field), *field.Default); ok {
		return []defaultFillData{fill}
	}
	return nil
}

// structOverrideFill returns the SetOnly fill of a field that restates an inherited
// struct default with a struct literal. It builds the whole value, so no leaf of the
// parent's default survives: the type's own defaults, then every leaf of the literal.
// It returns ok=false when the default is not a plain struct literal.
func structOverrideFill(
	field resolution.Field,
	data *templateData,
) (defaultFillData, bool) {
	if field.Default == nil || field.Optional ||
		field.Default.Kind != resolution.ValueKindStruct {
		return defaultFillData{}, false
	}
	if _, ok := unionLiteralFill("", field.Type, *field.Default, data); ok {
		return defaultFillData{}, false
	}
	goType := stripPointer(data.resolver.ResolveTypeRef(field.Type, data.ctx))
	setup := []string{"v0 := " + goType + "{}"}
	defaulted := typeSetsDefaults(field.Type, data, set.New[string]())
	groups := typeHasGroups(field.Type, data, set.New[string]())
	setup = append(setup, literalDefaultsSetup("v0", defaulted, groups)...)
	next := 1
	lines, ok := literalAssignments(
		"v0",
		field.Type,
		false,
		defaulted,
		*field.Default,
		data,
		&next,
	)
	if !ok {
		return defaultFillData{}, false
	}
	return defaultFillData{
		GoName:  naming.GetFieldName(field),
		Expr:    "v0",
		Setup:   append(setup, lines...),
		SetOnly: true,
	}, true
}

// isEmptyList reports whether v is an empty list literal. In a value that SetDefaults
// never filled, the zero value already holds it, so no assignment is needed.
func isEmptyList(v resolution.ExpressionValue) bool {
	return v.Kind == resolution.ValueKindArray && len(v.Elements) == 0
}

// zeroFill returns the fill that writes the zero value v to the Go selector goName. It
// returns ok=false for a kind whose zero has no plain literal, such as an enum member.
func zeroFill(goName string, v resolution.ExpressionValue) (defaultFillData, bool) {
	var lit string
	switch v.Kind {
	case resolution.ValueKindString:
		lit = `""`
	case resolution.ValueKindInt, resolution.ValueKindFloat:
		lit = "0"
	case resolution.ValueKindBool:
		lit = "false"
	case resolution.ValueKindArray:
		lit = "nil"
	default:
		return defaultFillData{}, false
	}
	return defaultFillData{GoName: goName, ZeroLit: lit, Expr: lit, SetOnly: true}, true
}

// unionLiteralFill returns the fill for a union field defaulted by an object literal
// such as `{ variant = "window", key = "r7" }`. The fill builds the selected variant in
// a local variable and assigns it to the nil Variant. It returns ok=false when typeRef
// is not a union or the literal names no variant of it.
func unionLiteralFill(
	goName string,
	typeRef resolution.TypeRef,
	val resolution.ExpressionValue,
	data *templateData,
) (defaultFillData, bool) {
	var next int
	setup, variable, ok := unionLiteral(typeRef, val, data, &next)
	if !ok {
		return defaultFillData{}, false
	}
	return defaultFillData{
		GoName:  goName + ".Variant",
		ZeroLit: "nil",
		Expr:    variable,
		Setup:   setup,
	}, true
}

// unionLiteral renders the statements that build the variant a union literal selects,
// returning them with the name of the local variable that holds it. next numbers the
// variables, so nested literals declare distinct ones.
func unionLiteral(
	typeRef resolution.TypeRef,
	val resolution.ExpressionValue,
	data *templateData,
	next *int,
) ([]string, string, bool) {
	resolved, unionRef, ok := resolution.Dealias(typeRef, data.table)
	if !ok {
		return nil, "", false
	}
	form, ok := resolved.Form.(resolution.UnionForm)
	if !ok {
		return nil, "", false
	}
	var variant resolution.UnionVariant
	found := false
	for _, comp := range val.Fields {
		if comp.Name == form.Discriminator {
			variant, found = form.Variant(comp.Value.StringValue)
			break
		}
	}
	if !found {
		return nil, "", false
	}
	fields := resolution.UnifiedVariantFields(resolved, variant, data.table)
	unionType := stripPointer(data.resolver.ResolveTypeRef(unionRef, data.ctx))
	variable := fmt.Sprintf("v%d", *next)
	*next++
	setup := []string{
		variable + " := " +
			casing.QualifiedVariantTypeName(unionType, variant.Name, ".") + "{}",
	}
	defaulted := variantSetsDefaults(form, variant, data)
	groups := variantHasGroups(form, variant, data)
	setup = append(setup, literalDefaultsSetup(variable, defaulted, groups)...)
	for _, comp := range val.Fields {
		if comp.Name == form.Discriminator {
			continue
		}
		f, ok := lo.Find(fields, func(f resolution.Field) bool {
			return f.Name == comp.Name
		})
		if !ok {
			return nil, "", false
		}
		if !defaulted && isEmptyList(comp.Value) {
			continue
		}
		lines, ok := literalAssignments(
			variable+"."+naming.GetFieldName(f),
			f.Type,
			f.Optional,
			defaulted,
			comp.Value,
			data,
			next,
		)
		if !ok {
			return nil, "", false
		}
		setup = append(setup, lines...)
	}
	if groups {
		setup = append(setup, variable+".DefaultGroups()")
	}
	return setup, variable, true
}

// literalDefaultsSetup returns the statement that sets the defaults of the literal
// value in variable before its members are assigned. It leaves the default groups of
// the value to a later DefaultGroups call, so a member the literal sets keeps the rest
// of its group at zero.
func literalDefaultsSetup(variable string, defaulted, groups bool) []string {
	switch {
	case groups:
		return []string{variable + ".SetUngroupedDefaults()"}
	case defaulted:
		return []string{variable + ".SetDefaults()"}
	}
	return nil
}

// literalAssignments renders the statements that assign the literal v to the Go
// selector of type typeRef, recursing into struct and union literals. An optional
// selector is a pointer, so the statements allocate its value first. defaulted reports
// whether SetDefaults filled the value the selector reaches into.
func literalAssignments(
	selector string,
	typeRef resolution.TypeRef,
	optional bool,
	defaulted bool,
	v resolution.ExpressionValue,
	data *templateData,
	next *int,
) ([]string, bool) {
	goType := stripPointer(data.resolver.ResolveTypeRef(typeRef, data.ctx))
	switch v.Kind {
	case resolution.ValueKindStruct:
		var lines []string
		if optional {
			lines = append(lines, selector+" = &"+goType+"{}")
			defaulted = false
		}
		if setup, variable, ok := unionLiteral(typeRef, v, data, next); ok {
			lines = append(lines, setup...)
			return append(lines, selector+".Variant = "+variable), true
		}
		resolved, structRef, ok := resolution.Dealias(typeRef, data.table)
		if !ok {
			return nil, false
		}
		fields := resolution.UnifiedFields(resolved, data.table)
		if form, ok := resolved.Form.(resolution.StructForm); ok && form.IsGeneric() {
			fields = lo.Map(fields, func(f resolution.Field, _ int) resolution.Field {
				f.Type = bindTypeArgs(f.Type, form.TypeParams, structRef.TypeArgs, data)
				return f
			})
		}
		for _, comp := range v.Fields {
			f, ok := lo.Find(fields, func(f resolution.Field) bool {
				return f.Name == comp.Name
			})
			if !ok {
				return nil, false
			}
			if !defaulted && isEmptyList(comp.Value) {
				continue
			}
			sub, ok := literalAssignments(
				selector+"."+naming.GetFieldName(f),
				f.Type,
				f.Optional,
				defaulted,
				comp.Value,
				data,
				next,
			)
			if !ok {
				return nil, false
			}
			lines = append(lines, sub...)
		}
		return lines, true
	case resolution.ValueKindArray:
		if len(v.Elements) == 0 {
			// The literal overrides a default list that SetDefaults set.
			return []string{selector + " = nil"}, true
		}
		setup, expr, ok := arrayLiteral(typeRef, v, data, next)
		if !ok {
			return nil, false
		}
		return append(setup, selector+" = "+expr), true
	}
	if v.Kind == resolution.ValueKindIdent {
		if expr, defaulted, ok := emptyVariant(v.IdentValue, typeRef, data); ok {
			var lines []string
			if defaulted {
				variable := fmt.Sprintf("v%d", *next)
				*next++
				lines = []string{variable + " := " + expr, variable + ".SetDefaults()"}
				expr = variable
			}
			if optional {
				return append(
					lines,
					selector+" = &"+goType+"{Variant: "+expr+"}",
				), true
			}
			return append(lines, selector+".Variant = "+expr), true
		}
	}
	lit, ok := goLiteral(typeRef, &v, data)
	if !ok {
		return nil, false
	}
	if optional {
		return []string{selector + " = new(" + goType + "(" + lit + "))"}, true
	}
	return []string{selector + " = " + lit}, true
}

// bindTypeArgs returns ref with each of params replaced by the type argument at its
// position in args, so a generic struct's field types name its concrete instantiation.
// When the resolver renders defaulted parameters as their defaults, a defaulted
// parameter stays unbound so the field keeps the type its declaration has.
func bindTypeArgs(
	ref resolution.TypeRef,
	params []resolution.TypeParam,
	args []resolution.TypeRef,
	data *templateData,
) resolution.TypeRef {
	if ref.IsTypeParam() {
		for i, p := range params {
			if p.Name != ref.TypeParam.Name || i >= len(args) {
				continue
			}
			if p.HasDefault() && data.ctx.SubstituteDefaultedTypeParams {
				return ref
			}
			return args[i]
		}
		return ref
	}
	if len(ref.TypeArgs) == 0 {
		return ref
	}
	bound := ref
	bound.TypeArgs = lo.Map(
		ref.TypeArgs,
		func(a resolution.TypeRef, _ int) resolution.TypeRef {
			return bindTypeArgs(a, params, args, data)
		},
	)
	return bound
}

// arrayLiteral renders the array literal v of slice type typeRef, returning the
// statements that build its object elements in local variables and the slice
// expression. next numbers the variables.
func arrayLiteral(
	typeRef resolution.TypeRef,
	v resolution.ExpressionValue,
	data *templateData,
	next *int,
) ([]string, string, bool) {
	if typeRef.Name != "Array" || len(typeRef.TypeArgs) != 1 {
		return nil, "", false
	}
	elemRef := typeRef.TypeArgs[0]
	elemType := stripPointer(data.resolver.ResolveTypeRef(elemRef, data.ctx))
	var setup []string
	elems := make([]string, 0, len(v.Elements))
	for _, elem := range v.Elements {
		if elem.Kind != resolution.ValueKindStruct {
			lit, ok := goLiteral(elemRef, &elem, data)
			if !ok {
				return nil, "", false
			}
			elems = append(elems, lit)
			continue
		}
		variable := fmt.Sprintf("v%d", *next)
		*next++
		lines, ok := literalAssignments(
			variable,
			elemRef,
			false,
			false,
			elem,
			data,
			next,
		)
		if !ok {
			return nil, "", false
		}
		setup = append(setup, variable+" := "+elemType+"{}")
		setup = append(setup, lines...)
		elems = append(elems, variable)
	}
	return setup, fmt.Sprintf(
		"%s{%s}",
		data.resolver.ResolveTypeRef(typeRef, data.ctx),
		strings.Join(elems, ", "),
	), true
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
	if len(d.Elements) == 0 {
		return defaultFillData{}, false
	}
	var next int
	setup, expr, ok := arrayLiteral(typeRef, *d, data, &next)
	if !ok {
		return defaultFillData{}, false
	}
	return defaultFillData{
		GoName:  goName,
		ZeroLit: "nil",
		Expr:    expr,
		Setup:   setup,
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
		return enumConstRef(typeRef, ev, data), true
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
		if expr, defaulted, ok := emptyVariant(d.IdentValue, typeRef, data); ok {
			// The wrapper holds the active variant behind a nil-able interface, so
			// the fill targets that field rather than the wrapper itself.
			fill := defaultFillData{
				GoName:  goName + ".Variant",
				ZeroLit: "nil",
				Expr:    expr,
			}
			if defaulted {
				fill.Setup = []string{"v0 := " + expr, "v0.SetDefaults()"}
				fill.Expr = "v0"
			}
			return fill, true
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
		return defaultFillData{
			GoName:  goName,
			ZeroLit: `""`,
			Expr:    enumConstRef(typeRef, ev, data),
		}, true
	}
	return defaultFillData{}, false
}

// emptyVariant renders the empty Go variant struct that a bare-name union default such
// as `= const_val` selects, and reports whether that struct has SetDefaults. It returns
// ok=false when typeRef is not a union with that variant.
func emptyVariant(
	ident string,
	typeRef resolution.TypeRef,
	data *templateData,
) (expr string, defaulted, ok bool) {
	uv, ok := validation.ResolveUnionVariant(ident, typeRef, data.table)
	if !ok {
		return "", false, false
	}
	// The variant structs live with the union, not with an alias of it.
	_, unionRef, _ := resolution.Dealias(typeRef, data.table)
	unionType := stripPointer(data.resolver.ResolveTypeRef(unionRef, data.ctx))
	expr = casing.QualifiedVariantTypeName(unionType, uv.Variant.Name, ".") + "{}"
	return expr, variantSetsDefaults(uv.Union, uv.Variant, data), true
}

// enumConstRef renders the Go constant of the enum member ev that a value of type
// typeRef defaults to. The constants live with the enum, not with an alias of it.
func enumConstRef(
	typeRef resolution.TypeRef,
	ev validation.EnumVariant,
	data *templateData,
) string {
	_, enumRef, _ := resolution.Dealias(typeRef, data.table)
	enumType := stripPointer(data.resolver.ResolveTypeRef(enumRef, data.ctx))
	return enumType + naming.ToPascalCase(ev.Variant.Name)
}

// constraintCheckData describes a single @validate constraint assertion emitted by a
// generated Validate method.
type constraintCheckData struct {
	GoName    string
	FieldName string
	// Kind selects the assertion form: non_empty_string, min_len, max_len (string),
	// non_zero (numeric or uuid), or ge, le (numeric).
	Kind string
	// Arg is the length or numeric threshold literal for the kinds that take one.
	Arg string
	// Opens and Closes mark the first and last checks of an optional field, which the
	// template wraps in one nil guard.
	Opens, Closes bool
	// Deref is true for an optional field, whose pointer the checks dereference.
	Deref bool
}

// goConstraintChecks returns the @validate constraint assertions for a field and its
// type chain, classified against the field type's underlying primitive so a distinct
// numeric type (e.g. a Key over uint32) validates as a number. An optional field is
// checked only when present, and never for being required. An inherited field checks
// only its own rules, since the embedded parent's Validate checks its type's rules.
func goConstraintChecks(
	field resolution.Field,
	inherited bool,
	data *templateData,
) []constraintCheckData {
	rules := validation.FieldRules(field, data.table)
	if inherited {
		rules = validation.Parse(field.Domains["validate"])
	}
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
			Deref:     field.Optional,
		}
	}
	var checks []constraintCheckData
	if base == "uuid" {
		if rules.Required && !field.Optional {
			checks = append(checks, check("non_zero", ""))
		}
	} else if resolution.IsStringPrimitive(base) {
		switch {
		case rules.MinLength != nil && *rules.MinLength > 1:
			checks = append(
				checks,
				check("min_len", strconv.FormatInt(*rules.MinLength, 10)),
			)
		case rules.Required && !field.Optional,
			rules.MinLength != nil && *rules.MinLength == 1:
			checks = append(checks, check("non_empty_string", ""))
		}
		if rules.MaxLength != nil {
			checks = append(
				checks,
				check("max_len", strconv.FormatInt(*rules.MaxLength, 10)),
			)
		}
	}
	if resolution.IsNumberPrimitive(base) {
		if rules.Required && !field.Optional {
			checks = append(checks, check("non_zero", ""))
		}
		if rules.Min != nil {
			checks = append(checks, check("ge", numberLiteral(rules.Min)))
		}
		if rules.GT != nil {
			checks = append(checks, check("gt", numberLiteral(rules.GT)))
		}
		if rules.Max != nil {
			checks = append(checks, check("le", numberLiteral(rules.Max)))
		}
	}
	if field.Optional && len(checks) > 0 {
		checks[0].Opens = true
		checks[len(checks)-1].Closes = true
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
	resolved, _, ok := resolution.Dealias(field.Type, data.table)
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
	// Groups marks a SetDefaults step into a type that has a `@default group`, which
	// splits its defaults into SetUngroupedDefaults and DefaultGroups.
	Groups bool
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
	return len(goConstraintChecks(f, false, probe)) > 0
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

// probeData returns a copy of data whose import manager is a scratch instance, so
// rendering done only to answer a has-own probe registers no imports on the real file.
func probeData(data *templateData) *templateData {
	m := imports.NewManager()
	r := *data.resolver
	r.ImportAdder = m
	d := *data
	d.Manager = m
	d.resolver = &r
	return &d
}

// typeNeedsMethod reports whether the type named by ref emits a recursive method: it
// has a field satisfying hasOwn, or a nested struct field, slice/array element, map
// value, or union variant payload that (transitively) does. Generic types and type
// parameters never emit a method. visited guards against cycles in recursive types.
// Probing must not leak imports into the emitted file, so hasOwn runs against a
// scratch-import copy of data.
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
		probe := probeData(data)
		for _, field := range resolution.UnifiedFields(resolved, data.table) {
			if skip(field, data) {
				continue
			}
			if hasOwn(field, probe) ||
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
		fd := data.forField(f)
		member, ok := groupMember(f, fd)
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
		g.Fills = append(g.Fills, goDefaultFills(f, fd)...)
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

// variantSetsDefaults reports whether the Go struct of variant v of form emits
// SetDefaults, by the rule the generated variant follows.
func variantSetsDefaults(
	form resolution.UnionForm,
	v resolution.UnionVariant,
	data *templateData,
) bool {
	// A named payload embeds the union's bases beside it.
	if !v.Inline {
		return lo.SomeBy(
			append(slices.Clone(form.Extends), v.Type),
			func(ref resolution.TypeRef) bool {
				return typeSetsDefaults(ref, data, set.New[string]())
			},
		)
	}
	payload, ok := v.Type.Resolve(data.table)
	if !ok {
		return false
	}
	inherited, declared := resolver.VariantBases(form, v, "go", data.table)
	for _, ref := range inherited {
		if typeSetsDefaults(ref, data, set.New[string]()) {
			return true
		}
	}
	pform, _ := payload.Form.(resolution.StructForm)
	probe := probeData(data)
	return lo.SomeBy(
		slices.Concat(declared, pform.Fields),
		func(f resolution.Field) bool {
			return defaultsHasOwn(f, probe) ||
				(!f.Optional && typeSetsDefaults(f.Type, data, set.New[string]()))
		},
	)
}

// typeSetsDefaults reports whether ref names a struct that emits SetDefaults: one with
// a field that has a static default, or with a required struct field or embedded
// parent whose type emits SetDefaults. A union never does, since its zero value holds
// no variant. visited guards against cycles in recursive types.
func typeSetsDefaults(
	ref resolution.TypeRef,
	data *templateData,
	visited set.Set[string],
) bool {
	if ref.IsTypeParam() || ref.Name == "Array" || ref.Name == "Map" ||
		visited.Contains(ref.Name) {
		return false
	}
	visited.Add(ref.Name)
	// An alias names the same Go type as its target.
	resolved, _, ok := resolution.Dealias(ref, data.table)
	if !ok {
		return false
	}
	form, ok := resolved.Form.(resolution.StructForm)
	if !ok || form.IsGeneric() {
		return false
	}
	probe := probeData(data)
	return lo.SomeBy(
		resolution.UnifiedFields(resolved, data.table),
		func(f resolution.Field) bool {
			return defaultsHasOwn(f, probe) ||
				(!f.Optional && typeSetsDefaults(f.Type, data, visited))
		},
	)
}

// setDefaultsSteps returns the SetDefaults calls a generated SetDefaults makes into
// embedded types and required struct fields. Optional, slice, and map fields hold no
// value until the input supplies one, so their own decode hooks default them.
func setDefaultsSteps(
	embeds []embeddedType,
	fields []resolution.Field,
	data *templateData,
) []recurseStepData {
	var steps []recurseStepData
	for _, e := range embeds {
		if typeSetsDefaults(e.ref, data, set.New[string]()) {
			steps = append(steps, recurseStepData{
				GoName: embedFieldName(e.rendered),
				Groups: typeHasGroups(e.ref, data, set.New[string]()),
			})
		}
	}
	for _, f := range fields {
		if !f.Optional && typeSetsDefaults(f.Type, data, set.New[string]()) {
			steps = append(steps, recurseStepData{
				GoName: naming.GetFieldName(f),
				Groups: typeHasGroups(f.Type, data, set.New[string]()),
			})
		}
	}
	return steps
}

// variantHasGroups reports whether the variant v of the union form has a `@default
// group` in its payload or in a type it sets the defaults of.
func variantHasGroups(
	form resolution.UnionForm,
	v resolution.UnionVariant,
	data *templateData,
) bool {
	hasGroups := func(ref resolution.TypeRef) bool {
		return typeHasGroups(ref, data, set.New[string]())
	}
	if !v.Inline {
		return lo.SomeBy(append(slices.Clone(form.Extends), v.Type), hasGroups)
	}
	payload, ok := v.Type.Resolve(data.table)
	if !ok {
		return false
	}
	inherited, declared := resolver.VariantBases(form, v, "go", data.table)
	if lo.SomeBy(inherited, hasGroups) {
		return true
	}
	pform, _ := payload.Form.(resolution.StructForm)
	probe := probeData(data)
	return lo.SomeBy(
		slices.Concat(declared, pform.Fields),
		func(f resolution.Field) bool {
			return (defaultGroupName(f) != "" && defaultsHasOwn(f, probe)) ||
				(!f.Optional && hasGroups(f.Type))
		},
	)
}

// typeHasGroups reports whether ref names a struct that has a `@default group` with a
// default to set, in its own fields or in a required struct field or embedded parent.
// visited guards against cycles in recursive types.
func typeHasGroups(
	ref resolution.TypeRef,
	data *templateData,
	visited set.Set[string],
) bool {
	if ref.IsTypeParam() || ref.Name == "Array" || ref.Name == "Map" ||
		visited.Contains(ref.Name) {
		return false
	}
	visited.Add(ref.Name)
	resolved, _, ok := resolution.Dealias(ref, data.table)
	if !ok {
		return false
	}
	form, ok := resolved.Form.(resolution.StructForm)
	if !ok || form.IsGeneric() {
		return false
	}
	probe := probeData(data)
	return lo.SomeBy(
		resolution.UnifiedFields(resolved, data.table),
		func(f resolution.Field) bool {
			return (defaultGroupName(f) != "" && defaultsHasOwn(f, probe)) ||
				(!f.Optional && typeHasGroups(f.Type, data, visited))
		},
	)
}

// applyFills returns the fills that ApplyDefaults assigns: every fill but SetOnly ones.
func applyFills(fills []defaultFillData) []defaultFillData {
	return lo.Reject(fills, func(f defaultFillData, _ int) bool { return f.SetOnly })
}

// setFills returns the fills that SetDefaults assigns unconditionally: every fill but
// ApplyOnly ones. A group's fills run only when all of its members are zero.
func setFills(fills []defaultFillData) []defaultFillData {
	return lo.Reject(fills, func(f defaultFillData, _ int) bool { return f.ApplyOnly })
}
