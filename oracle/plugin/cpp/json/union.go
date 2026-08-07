// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"github.com/samber/lo"
	cppnaming "github.com/synnaxlabs/oracle/plugin/cpp/naming"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
)

// unionDispatchData drives generation of the free parse_<name> /
// to_json(const <Name>&) functions for a discriminated union.
type unionDispatchData struct {
	// Name is the union alias name (e.g. "Scale").
	Name string
	// SnakeName is used for the free parse function (e.g. parse_scale).
	SnakeName string
	// DiscJSON is the discriminator JSON field name (e.g. "type").
	DiscJSON string
	// Variants holds the discriminator-value to variant-type mapping.
	Variants []unionDispatchVariant
}

type unionDispatchVariant struct {
	// TypeName is the variant struct name (e.g. "ScaleLinear").
	TypeName string
	// Value is the discriminator string value (e.g. "linear").
	Value string
}

// processUnion builds a serializer for every variant struct of a discriminated
// union (so each gets parse/to_json bodies like a regular struct) plus the
// dispatch data for the union's free parse and to_json functions. The variant
// serializers carry the discriminator as a leading std::string field followed
// by the flattened base and variant fields, matching the declarations emitted
// by the cpp/types plugin.
func (p *Plugin) processUnion(
	u resolution.Type,
	data *templateData,
) ([]serializerData, unionDispatchData) {
	form := u.Form.(resolution.UnionForm)
	name := domain.GetName(u, "cpp")
	data.includes.addSystem("variant")

	dispatch := unionDispatchData{
		Name:      name,
		SnakeName: lo.SnakeCase(name),
		DiscJSON:  toSnakeCase(form.Discriminator),
	}
	discField := resolution.Field{
		Name: form.Discriminator,
		Type: resolution.TypeRef{Name: "string"},
	}

	serializers := make([]serializerData, 0, len(form.Variants))
	for _, v := range form.Variants {
		variantName := cppnaming.VariantTypeName(name, v.Name)
		s := serializerData{
			Name:       variantName,
			HasExtends: true,
			Fields:     make([]fieldData, 0),
		}
		for _, ext := range form.Extends {
			if parent, ok := ext.Resolve(data.table); ok {
				s.ParentTypes = append(s.ParentTypes, parentTypeData{
					QualifiedName: p.resolveExtendsType(ext, parent, data),
				})
			}
		}
		if payload, ok := v.Type.Resolve(data.table); ok {
			if v.Inline {
				pform := payload.Form.(resolution.StructForm)
				for _, ext := range pform.Extends {
					if parent, ok := ext.Resolve(data.table); ok {
						s.ParentTypes = append(s.ParentTypes, parentTypeData{
							QualifiedName: p.resolveExtendsType(ext, parent, data),
						})
					}
				}
				for _, f := range pform.Fields {
					s.Fields = append(s.Fields, p.processField(f, payload, data))
				}
			} else {
				s.ParentTypes = append(s.ParentTypes, parentTypeData{
					QualifiedName: p.resolveExtendsType(v.Type, payload, data),
				})
			}
		}
		s.Fields = append(s.Fields, p.processField(discField, u, data))
		serializers = append(serializers, s)
		dispatch.Variants = append(dispatch.Variants, unionDispatchVariant{
			TypeName: variantName,
			Value:    v.Name,
		})
	}
	return serializers, dispatch
}
