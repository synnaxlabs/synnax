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
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/py/keywords"
	"github.com/synnaxlabs/oracle/resolution"
)

// unionData is the template view of a discriminated union. A union renders as
// one Pydantic BaseModel per variant plus an Annotated[Union[...],
// Field(discriminator=...)] alias that ties them together.
type unionData struct {
	// Name is the union alias name (e.g. "Scale").
	Name string
	// Doc is the rendered documentation comment, if any.
	Doc string
	// DiscName is the discriminator field name (e.g. "type").
	DiscName string
	// Variants lists every variant in declaration order.
	Variants []unionVariantData
}

// unionVariantData is the template view of one variant of a discriminated union.
type unionVariantData struct {
	// ClassName is the variant model class name (e.g. "ScaleLinear").
	ClassName string
	// Value is the discriminator string value (e.g. "linear").
	Value string
	// Doc is the rendered per-variant documentation comment, if any.
	Doc string
	// HasKeywordAlias is true if any field needs a populate-by-name config.
	HasKeywordAlias bool
	// Fields are the flattened base + variant fields, excluding the
	// discriminator, which the template emits as a Literal field.
	Fields []fieldData
}

// processUnion builds the template view for a discriminated union, flattening
// each variant's base and own fields via resolution.UnifiedVariantFields and
// reusing the struct field processor so union variant fields render identically
// to struct fields.
func processUnion(
	entry resolution.Type,
	table *resolution.Table,
	data *templateData,
	keyFields []keyFieldData,
) unionData {
	form := entry.Form.(resolution.UnionForm)
	data.imports.addPydantic("BaseModel")
	data.imports.addPydantic("Field")
	data.imports.addTyping("Annotated")
	data.imports.addTyping("Union")
	data.imports.addTyping("Literal")
	ud := unionData{
		Name:     getPyName(entry),
		Doc:      doc.Get(entry.Domains),
		DiscName: keywords.Escape(form.Discriminator),
	}
	for _, v := range form.Variants {
		vd := unionVariantData{
			ClassName: casing.VariantTypeName(ud.Name, v.Name),
			Value:     v.Name,
			Doc:       doc.Get(v.Domains),
		}
		for _, f := range resolution.UnifiedVariantFields(entry, v, table) {
			fd := processField(f, table, data, keyFields, nil)
			if fd.Alias != "" {
				vd.HasKeywordAlias = true
				data.imports.addPydantic("ConfigDict")
			}
			vd.Fields = append(vd.Fields, fd)
		}
		ud.Variants = append(ud.Variants, vd)
	}
	return ud
}
