// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package defaults renders schema default values as C++ expressions.
package defaults

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/cpp/keywords"
	cppnaming "github.com/synnaxlabs/oracle/plugin/cpp/naming"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/internal/telem"
	"github.com/synnaxlabs/oracle/resolution"
)

// Renderer renders default values for one generated namespace.
type Renderer struct {
	// Table resolves type references.
	Table *resolution.Table
	// TypeName returns the C++ spelling of a type reference.
	TypeName func(resolution.TypeRef) string
	// Namespace is the schema namespace of the generated file.
	Namespace string
	// Include, when set, receives the header of each cross-namespace enum that a
	// default references.
	Include func(path string)
}

// Literal renders val as a C++ brace-init literal. typeRef is the declared type of val.
// It returns "" for a default with no C++ form, such as the create sentinel.
func (r Renderer) Literal(
	typeRef resolution.TypeRef,
	val resolution.ExpressionValue,
) string {
	switch val.Kind {
	case resolution.ValueKindString:
		return r.wrapDistinct(typeRef, fmt.Sprintf("%q", val.StringValue))
	case resolution.ValueKindInt:
		return r.wrapDistinct(typeRef, fmt.Sprintf("%d", val.IntValue))
	case resolution.ValueKindFloat:
		return r.wrapDistinct(typeRef, fmt.Sprintf("%f", val.FloatValue))
	case resolution.ValueKindBool:
		return fmt.Sprintf("%t", val.BoolValue)
	case resolution.ValueKindIdent:
		if ev, ok := validation.ResolveEnumVariant(
			val.IdentValue,
			typeRef,
			r.Table,
		); ok {
			return r.EnumVariant(ev)
		}
		if val.IdentValue == "true" || val.IdentValue == "false" {
			return val.IdentValue
		}
		if val.IdentValue == "now" && telem.IsTimeStamp(typeRef, r.Table, "cpp") {
			return "x::telem::TimeStamp::now()"
		}
		if uv, ok := validation.ResolveUnionVariant(
			val.IdentValue,
			typeRef,
			r.Table,
		); ok {
			// std::variant default-constructs its first alternative, so a default
			// naming any other variant must be written out explicitly. The variant
			// structs live with the union, not with an alias of it.
			_, unionRef, _ := resolution.Dealias(typeRef, r.Table)
			return cppnaming.QualifiedVariantTypeName(
				r.TypeName(unionRef),
				uv.Variant.Name,
			) + "{}"
		}
		return ""
	case resolution.ValueKindArray:
		elem := arrayElementType(typeRef)
		parts := make([]string, 0, len(val.Elements))
		for _, el := range val.Elements {
			parts = append(parts, r.Literal(elem, el))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case resolution.ValueKindStruct:
		return r.structLiteral(typeRef, val)
	}
	return ""
}

// EnumVariant renders a fully-qualified C++ reference to an enum variant. Int enums
// generate as enum classes and are referenced Enum::Variant. String enums generate as
// ENUM_VARIANT string constants.
func (r Renderer) EnumVariant(ev validation.EnumVariant) string {
	ref := fmt.Sprintf("%s::%s", ev.Type.Name, lo.PascalCase(ev.Variant.Name))
	if form, ok := ev.Type.Form.(resolution.EnumForm); ok && !form.IsIntEnum {
		ref = fmt.Sprintf(
			"%s_%s",
			cppnaming.ScreamingSnake(ev.Type.Name),
			cppnaming.ScreamingSnake(ev.Variant.Name),
		)
	}
	if ev.Type.Namespace != r.Namespace {
		targetOutputPath := enum.FindOutputPath(ev.Type, r.Table, "cpp")
		if targetOutputPath != "" {
			if r.Include != nil {
				r.Include(targetOutputPath + "/types.gen.h")
			}
			ns := cppnaming.Namespace(targetOutputPath)
			return fmt.Sprintf("::%s::%s", ns, ref)
		}
	}
	return ref
}

// structLiteral renders a struct default as a designated-initializer list, in the
// struct's declaration order as C++ requires.
func (r Renderer) structLiteral(
	typeRef resolution.TypeRef,
	val resolution.ExpressionValue,
) string {
	resolved, _, ok := resolution.Dealias(typeRef, r.Table)
	if !ok {
		return "{}"
	}
	provided := make(map[string]resolution.ExpressionValue, len(val.Fields))
	for _, fv := range val.Fields {
		provided[fv.Name] = fv.Value
	}
	parts := make([]string, 0, len(val.Fields))
	for _, f := range resolution.UnifiedFields(resolved, r.Table) {
		v, has := provided[f.Name]
		if !has {
			continue
		}
		parts = append(
			parts,
			fmt.Sprintf(".%s = %s", fieldName(f), r.Literal(f.Type, v)),
		)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// wrapDistinct direct-initializes a scalar literal with its C++ type when typeRef
// resolves to a distinct type. Hand-written distinct types such as x::telem::Rate and
// x::telem::DataType expose only explicit constructors, so a bare literal fails
// copy-initialization both as a member initializer and as a parser.field fallback.
func (r Renderer) wrapDistinct(typeRef resolution.TypeRef, literal string) string {
	if !resolution.IsDistinct(typeRef, r.Table) {
		return literal
	}
	return fmt.Sprintf("%s(%s)", r.TypeName(typeRef), literal)
}

func fieldName(field resolution.Field) string {
	name := domain.GetFieldName(field, "cpp")
	if name == field.Name {
		name = casing.FieldSnake(field.Name)
	}
	return keywords.Escape(name)
}

func arrayElementType(typeRef resolution.TypeRef) resolution.TypeRef {
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return typeRef.TypeArgs[0]
	}
	return typeRef
}
