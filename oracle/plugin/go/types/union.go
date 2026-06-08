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
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/resolution"
)

// unionData is the template view of a discriminated union. A union renders as a
// sealed variant interface, one concrete struct per variant (each embedding the
// shared base and the variant payload), and a concrete wrapper struct that holds
// the active variant and owns internally-tagged JSON marshaling. The wrapper is
// what union-typed fields reference, so a union field round-trips like any other
// struct field.
type unionData struct {
	// Name is the wrapper struct name and the type union fields reference
	// (e.g. "Scale").
	Name string
	// Doc is the rendered documentation comment, if any.
	Doc string
	// InterfaceName is the sealed variant interface (e.g. "ScaleVariant").
	InterfaceName string
	// Marker is the unexported sealing method (e.g. "isScaleVariant").
	Marker string
	// DiscType is the string discriminator type (e.g. "ScaleType").
	DiscType string
	// DiscJSONName is the discriminator JSON tag (e.g. "type").
	DiscJSONName string
	// Variants lists every variant in declaration order.
	Variants []unionVariantData
}

// unionVariantData is the template view of one variant of a discriminated union.
type unionVariantData struct {
	// TypeName is the concrete variant struct name (e.g. "ScaleLinear").
	TypeName string
	// ConstName is the discriminator constant (e.g. "ScaleTypeLinear").
	ConstName string
	// Value is the discriminator string value (e.g. "linear").
	Value string
	// Doc is the rendered per-variant documentation comment, if any.
	Doc string
	// Embeds are the embedded type names contributing the variant's fields:
	// the union's shared base structs followed by the variant payload struct.
	Embeds []string
}

func processUnion(entry resolution.Type, data *templateData) unionData {
	form := entry.Form.(resolution.UnionForm)
	// Use the raw schema name (with @go name override) so the wrapper matches
	// resolver.GetTypeName, which union-typed fields resolve through.
	name := entry.Name
	if override := domain.GetStringFromType(entry, "go", "name"); override != "" {
		name = override
	}
	data.imports.AddExternal("encoding/json")
	data.imports.AddExternal("github.com/synnaxlabs/x/errors")

	ud := unionData{
		Name:          name,
		Doc:           doc.Get(entry.Domains),
		InterfaceName: name + "Variant",
		Marker:        "is" + name + "Variant",
		DiscType:      name + "Type",
		DiscJSONName:  casing.FieldSnake(form.Discriminator),
	}

	var baseEmbeds []string
	for _, ext := range form.Extends {
		parent, ok := ext.Resolve(data.table)
		if !ok {
			continue
		}
		baseEmbeds = append(baseEmbeds, resolveExtendsType(ext, parent, data))
	}

	for _, v := range form.Variants {
		vd := unionVariantData{
			TypeName:  casing.VariantTypeName(name, v.Name),
			ConstName: ud.DiscType + casing.PascalAcronym(v.Name),
			Value:     v.Name,
			Doc:       doc.Get(v.Domains),
		}
		vd.Embeds = append(vd.Embeds, baseEmbeds...)
		vd.Embeds = append(vd.Embeds, data.resolver.ResolveTypeRef(v.Type, data.ctx))
		ud.Variants = append(ud.Variants, vd)
	}
	return ud
}
