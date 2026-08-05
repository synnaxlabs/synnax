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

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
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
	// NeedsApplyDefaults reports that at least one variant carries an ApplyDefaults
	// method, so the wrapper emits one that dispatches on the active variant.
	NeedsApplyDefaults bool
	// NeedsValidate reports that at least one variant carries a Validate method, so the
	// wrapper emits one that dispatches on the active variant.
	NeedsValidate bool
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
	// the union's shared base structs followed by the variant payload struct
	// (or, for inline variants, the payload's extends bases).
	Embeds []string
	// Fields are the variant's own fields, declared directly on the variant
	// struct. Populated only for inline variants, whose Synthetic payload has
	// no standalone type to embed.
	Fields []fieldData
	// Receiver is the receiver identifier used by the variant's generated methods.
	Receiver string
	// DefaultRecurse and ValidateRecurse are the nested-method calls the variant's
	// ApplyDefaults/Validate make into its embedded types and inline fields. An embed
	// step carries an empty JSONName: its fields are promoted to the variant's level,
	// so they take no Validate path segment.
	DefaultRecurse  []recurseStepData
	ValidateRecurse []recurseStepData
	// NeedsApplyDefaults and NeedsValidate report whether the variant emits the
	// respective method.
	NeedsApplyDefaults bool
	NeedsValidate      bool
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

	var baseEmbeds []variantEmbed
	for _, ext := range form.Extends {
		parent, ok := ext.Resolve(data.table)
		if !ok {
			continue
		}
		baseEmbeds = append(
			baseEmbeds,
			variantEmbed{ref: ext, rendered: resolveExtendsType(ext, parent, data)},
		)
	}

	for _, v := range form.Variants {
		vd := unionVariantData{
			TypeName:  casing.VariantTypeName(name, v.Name),
			ConstName: ud.DiscType + casing.PascalAcronym(v.Name),
			Value:     v.Name,
			Doc:       doc.Get(v.Domains),
		}
		vd.Receiver = strings.ToLower(vd.TypeName[:1])
		embeds := append([]variantEmbed{}, baseEmbeds...)
		var inlineFields []resolution.Field
		if v.Inline {
			if payload, ok := v.Type.Resolve(data.table); ok {
				pform := payload.Form.(resolution.StructForm)
				for _, ext := range pform.Extends {
					if parent, ok := ext.Resolve(data.table); ok {
						embeds = append(
							embeds,
							variantEmbed{
								ref:      ext,
								rendered: resolveExtendsType(ext, parent, data),
							},
						)
					}
				}
				inlineFields = pform.Fields
				for _, f := range pform.Fields {
					vd.Fields = append(vd.Fields, processField(f, data))
				}
			}
		} else {
			embeds = append(
				embeds,
				variantEmbed{
					ref:      v.Type,
					rendered: data.resolver.ResolveTypeRef(v.Type, data.ctx),
				},
			)
		}
		for _, e := range embeds {
			vd.Embeds = append(vd.Embeds, e.rendered)
		}
		vd.DefaultRecurse = variantRecurseSteps(
			embeds,
			inlineFields,
			data,
			defaultsHasOwn,
			neverSkip,
		)
		vd.ValidateRecurse = variantRecurseSteps(
			embeds,
			inlineFields,
			data,
			validateHasOwn,
			validateSkip,
		)
		vd.NeedsApplyDefaults = len(vd.DefaultRecurse) > 0
		vd.NeedsValidate = len(vd.ValidateRecurse) > 0
		if vd.NeedsApplyDefaults {
			ud.NeedsApplyDefaults = true
		}
		if vd.NeedsValidate {
			ud.NeedsValidate = true
		}
		if hasSliceRecurse(vd.ValidateRecurse) {
			data.imports.AddExternal(strconvImportPath)
		}
		ud.Variants = append(ud.Variants, vd)
	}
	if ud.NeedsValidate {
		data.imports.AddExternal(validateImportPath)
	}
	return ud
}

// variantEmbed pairs a variant's embedded type reference with its rendered Go type, so
// the recursion predicate can inspect the type while the field selector is derived from
// the rendered name.
type variantEmbed struct {
	ref      resolution.TypeRef
	rendered string
}

// variantRecurseSteps returns the nested-method steps for a union variant: a value step
// per embedded type that (transitively) needs the method, followed by the steps for any
// inline fields. Embed steps carry no JSONName, since an embedded type's fields are
// promoted to the variant's level and take no Validate path segment.
func variantRecurseSteps(
	embeds []variantEmbed,
	inlineFields []resolution.Field,
	data *templateData,
	hasOwn fieldHasOwn,
	skip fieldHasOwn,
) []recurseStepData {
	var steps []recurseStepData
	for _, e := range embeds {
		if resolvesToMethodType(e.ref, data) &&
			typeNeedsMethod(e.ref, data, set.New[string](), hasOwn, skip) {
			steps = append(
				steps,
				recurseStepData{GoName: embedFieldName(e.rendered), Kind: recurseValue},
			)
		}
	}
	for _, f := range inlineFields {
		if skip(f, data) {
			continue
		}
		if step, ok := goRecurseStep(f, data, hasOwn, skip); ok {
			steps = append(steps, step)
		}
	}
	return steps
}

// embedFieldName derives the Go field selector for an embedded type from its rendered
// type expression, dropping any package qualifier and type arguments (spatial.Bounds ->
// Bounds, Box[Item] -> Box).
func embedFieldName(rendered string) string {
	if i := strings.IndexByte(rendered, '['); i >= 0 {
		rendered = rendered[:i]
	}
	if i := strings.LastIndexByte(rendered, '.'); i >= 0 {
		return rendered[i+1:]
	}
	return rendered
}
