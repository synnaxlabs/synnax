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
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
)

// unionData is the template view of a discriminated union. Each variant is
// rendered as a standalone internally-tagged z.object (base fields, the
// discriminator literal, then the variant's own fields), and the variants are
// combined into a single z.discriminatedUnion.
type unionData struct {
	// TSName is the union's TypeScript name (e.g. "Scale").
	TSName string
	// SchemaName is the discriminatedUnion schema const (e.g. "scaleZ").
	SchemaName string
	// Discriminator is the JSON field name dispatched on (e.g. "type").
	Discriminator string
	// Doc is the rendered documentation comment, if any.
	Doc string
	// TypesConst is the readonly array of discriminator values
	// (e.g. "SCALE_TYPES").
	TypesConst string
	// TypeSchemaName is the discriminator enum schema (e.g. "scaleTypeZ").
	TypeSchemaName string
	// TypeName is the discriminator type alias (e.g. "ScaleType").
	TypeName string
	// SchemasConst is the per-variant schema map (e.g. "SCALE_SCHEMAS").
	SchemasConst string
	// Variants lists every variant in declaration order.
	Variants []unionVariantData
}

// unionVariantData is the template view of one variant of a discriminated union.
type unionVariantData struct {
	// Value is the discriminator string value (e.g. "linear").
	Value string
	// TypeName is the variant's TypeScript interface name (e.g. "ScaleLinear").
	TypeName string
	// SchemaName is the variant's z.object schema const (e.g. "scaleLinearZ").
	SchemaName string
	// Doc is the rendered per-variant documentation comment, if any.
	Doc string
	// Fields are the flattened base + variant fields, excluding the
	// discriminator, which the template emits as a z.literal.
	Fields []fieldData
}

// processUnion builds the template view for a discriminated union. It flattens
// each variant's fields via resolution.UnifiedVariantFields and reuses the
// struct field processor so union variant fields render identically to struct
// fields (arrays, optionality, nested schemas, forward references).
func (p *Plugin) processUnion(entry resolution.Type, table *resolution.Table, data *templateData) unionData {
	form, ok := entry.Form.(resolution.UnionForm)
	if !ok {
		return unionData{}
	}
	tsName := domain.GetName(entry, "ts")
	screaming := strings.ToUpper(lo.SnakeCase(tsName))
	ud := unionData{
		TSName:         tsName,
		SchemaName:     camelCase(tsName) + "Z",
		Discriminator:  form.Discriminator,
		Doc:            doc.Get(entry.Domains),
		TypesConst:     screaming + "_TYPES",
		TypeSchemaName: camelCase(tsName) + "TypeZ",
		TypeName:       tsName + "Type",
		SchemasConst:   screaming + "_SCHEMAS",
	}
	for _, v := range form.Variants {
		variantTSName := tsName + pascalCase(v.Name)
		vd := unionVariantData{
			Value:      v.Name,
			TypeName:   variantTSName,
			SchemaName: camelCase(variantTSName) + "Z",
			Doc:        doc.Get(v.Domains),
		}
		for _, f := range resolution.UnifiedVariantFields(entry, v, table) {
			vd.Fields = append(vd.Fields, p.processField(f, entry, table, data, false, false))
		}
		ud.Variants = append(ud.Variants, vd)
	}
	return ud
}

// pascalCase upper-cases the first rune of the camelCased form of s without
// lower-casing the rest, so acronym runs produced by camelCasing survive
// (e.g. "ai_voltage" -> "AiVoltage").
func pascalCase(s string) string {
	c := lo.CamelCase(s)
	if c == "" {
		return c
	}
	return strings.ToUpper(c[:1]) + c[1:]
}
