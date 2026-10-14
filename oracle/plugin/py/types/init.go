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
	"github.com/synnaxlabs/oracle/resolution"
)

// initParamData is one keyword parameter of a model's type-checking __init__.
type initParamData struct {
	Name     string
	PyType   string
	Required bool
}

// buildInitParams returns the parameters of a type-checking __init__ for a model whose
// complete field set is fields, or nil when the synthesized __init__ is already
// correct. Type checkers synthesize __init__ parameters from field annotations, so a
// field typed as a hand-written distinct type (e.g. telem.DataType) rejects the base
// primitive that Pydantic accepts at runtime. The explicit __init__ types such a
// parameter as its base primitive, while the attribute keeps the precise type.
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
// model: its discriminator, then the fields it inherits and declares.
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
	params := buildInitParams(resolution.UnifiedFields(synthetic, table), table, data)
	if params == nil {
		return nil
	}
	disc := initParamData{Name: discName, PyType: fmt.Sprintf("Literal[%q]", value)}
	return append([]initParamData{disc}, params...)
}
