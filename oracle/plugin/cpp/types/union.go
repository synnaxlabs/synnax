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

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/plugin/cpp/keywords"
	cppnaming "github.com/synnaxlabs/oracle/plugin/cpp/naming"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
)

// unionData is the template view of a discriminated union's std::variant alias.
// The variant structs themselves are emitted as ordinary struct declarations
// ahead of the alias so they pick up the standard parse/to_json members.
type unionData struct {
	// Name is the union alias name (e.g. "Scale").
	Name string
	// SnakeName is the snake_case form used for the free parse function
	// (e.g. "scale" -> parse_scale).
	SnakeName string
	// Doc is the rendered documentation comment, if any.
	Doc string
	// DiscJSON is the discriminator JSON field name (e.g. "type").
	DiscJSON string
	// Variants holds the dispatch info for each variant.
	Variants []unionVariantData
}

// unionVariantData is the template view of one variant for the dispatch alias.
type unionVariantData struct {
	// TypeName is the variant struct name (e.g. "ScaleLinear").
	TypeName string
	// Value is the discriminator string value (e.g. "linear").
	Value string
}

// processUnion builds the variant struct declarations and the std::variant
// alias for a discriminated union. Each variant struct carries the
// discriminator as a defaulted std::string field followed by the flattened
// base and variant fields, so the existing struct template and JSON codec
// handle it like any other struct.
func (p *Plugin) processUnion(entry resolution.Type, data *templateData) ([]structData, unionData) {
	form := entry.Form.(resolution.UnionForm)
	name := domain.GetName(entry, "cpp")
	data.includes.addSystem("variant")
	data.includes.addSystem("string")

	discField := keywords.Escape(toSnakeCase(form.Discriminator))
	ud := unionData{
		Name:      name,
		SnakeName: toSnakeCase(name),
		Doc:       doc.Get(entry.Domains),
		DiscJSON:  toSnakeCase(form.Discriminator),
	}

	variants := make([]structData, 0, len(form.Variants))
	for _, v := range form.Variants {
		variantName := cppnaming.VariantTypeName(name, v.Name)
		sd := structData{
			Name:       variantName,
			Doc:        doc.Get(v.Domains),
			HasExtends: true,
			Fields: []fieldData{{
				Name:         discField,
				CppType:      "std::string",
				DefaultValue: fmt.Sprintf("%q", v.Name),
			}},
		}
		for _, ext := range form.Extends {
			if parent, ok := ext.Resolve(data.table); ok {
				sd.ExtendsTypes = append(sd.ExtendsTypes, p.resolveExtendsType(ext, parent, data))
			}
		}
		if payload, ok := v.Type.Resolve(data.table); ok {
			if v.Inline {
				pform := payload.Form.(resolution.StructForm)
				for _, ext := range pform.Extends {
					if parent, ok := ext.Resolve(data.table); ok {
						sd.ExtendsTypes = append(sd.ExtendsTypes,
							p.resolveExtendsType(ext, parent, data))
					}
				}
				for _, f := range pform.Fields {
					sd.Fields = append(sd.Fields, p.processField(f, payload, data))
				}
			} else {
				sd.ExtendsTypes = append(sd.ExtendsTypes, p.resolveExtendsType(v.Type, payload, data))
			}
		}
		data.includes.addInternal("x/cpp/json/json.h")
		variants = append(variants, sd)
		ud.Variants = append(ud.Variants, unionVariantData{
			TypeName: variantName,
			Value:    v.Name,
		})
	}
	return variants, ud
}
