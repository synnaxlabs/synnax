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
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
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
	// ParentSchemas are the schema consts the variant composes via .extend: the
	// union's shared base schema(s) followed by the variant's payload schema.
	// The variant adds only the discriminator, so shared fields are not
	// duplicated.
	ParentSchemas []string
	// Fields are the variant's own fields, rendered directly into the member
	// schema. Populated only for inline variants, whose Synthetic payload has
	// no standalone schema const to extend.
	Fields []fieldData
}

// processUnion builds the template view for a discriminated union. Each variant
// composes the union's shared base schema(s) and its own payload schema via
// zod .extend, adding only the discriminator literal, so the shared fields are
// not duplicated across variants.
func (p *Plugin) processUnion(
	entry resolution.Type,
	table *resolution.Table,
	data *templateData,
) unionData {
	form, ok := entry.Form.(resolution.UnionForm)
	if !ok {
		return unionData{}
	}
	tsName := domain.GetName(entry, "ts")
	screaming := strings.ToUpper(lo.SnakeCase(tsName))
	ud := unionData{
		TSName:         tsName,
		SchemaName:     camelCase(tsName) + "Z",
		Discriminator:  fieldCamel(form.Discriminator),
		Doc:            doc.Get(entry.Domains),
		TypesConst:     screaming + "_TYPES",
		TypeSchemaName: camelCase(tsName) + "TypeZ",
		TypeName:       tsName + "Type",
		SchemasConst:   screaming + "_SCHEMAS",
	}
	for _, v := range form.Variants {
		typeName := casing.VariantTypeName(tsName, v.Name)
		vd := unionVariantData{
			Value:      v.Name,
			TypeName:   typeName,
			SchemaName: camelCase(typeName) + "Z",
			Doc:        doc.Get(v.Domains),
		}
		omitted := variantOmissions(v, table)
		for _, ext := range form.Extends {
			if pn, ok := parentSchemaName(ext, table, data); ok {
				vd.ParentSchemas = append(
					vd.ParentSchemas, pn+omitClause(ext, omitted, table),
				)
			}
		}
		if v.Inline {
			if payload, ok := v.Type.Resolve(table); ok {
				if pform, ok := payload.Form.(resolution.StructForm); ok {
					for _, ext := range pform.Extends {
						if pn, ok := parentSchemaName(ext, table, data); ok {
							vd.ParentSchemas = append(
								vd.ParentSchemas, pn+omitClause(ext, omitted, table),
							)
						}
					}
					for _, f := range pform.Fields {
						// The union entry is the parent, not the synthesized
						// payload, so a field referencing the union itself (a
						// recursive variant) renders as a lazy getter.
						vd.Fields = append(vd.Fields,
							p.processField(f, entry, table, data, false))
					}
				}
			}
		} else if pn, ok := parentSchemaName(v.Type, table, data); ok {
			vd.ParentSchemas = append(vd.ParentSchemas, pn)
		}
		ud.Variants = append(ud.Variants, vd)
	}
	return ud
}

// variantOmissions returns the fields a variant drops from its inherited shape.
func variantOmissions(
	v resolution.UnionVariant, table *resolution.Table,
) set.Set[string] {
	if !v.Inline {
		return nil
	}
	payload, ok := v.Type.Resolve(table)
	if !ok {
		return nil
	}
	form, ok := payload.Form.(resolution.StructForm)
	if !ok {
		return nil
	}
	return set.New(form.OmittedFields...)
}

// omitClause renders the `.omit({...})` a parent schema needs when the variant
// drops one of the fields that parent contributes. Empty when it drops none.
func omitClause(
	parent resolution.TypeRef, omitted set.Set[string], table *resolution.Table,
) string {
	if len(omitted) == 0 {
		return ""
	}
	resolved, ok := parent.Resolve(table)
	if !ok {
		return ""
	}
	var keys []string
	for _, f := range resolution.UnifiedFields(resolved, table) {
		if omitted.Contains(f.Name) {
			keys = append(keys, camelCase(f.Name)+": true")
		}
	}
	if len(keys) == 0 {
		return ""
	}
	return ".omit({ " + strings.Join(keys, ", ") + " })"
}
