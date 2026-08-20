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
	"slices"
	"strings"

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/resolver"
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
	// DefaultFills, EnumChecks, and ConstraintChecks are the variant's own inline
	// fields' fills and assertions, mirroring the struct-level equivalents.
	DefaultFills     []defaultFillData
	EnumChecks       []enumCheckData
	ConstraintChecks []constraintCheckData
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
	data.AddExternal("encoding/json")
	data.AddExternal("github.com/synnaxlabs/x/errors")

	ud := unionData{
		Name:          name,
		Doc:           doc.Get(entry.Domains),
		InterfaceName: name + "Variant",
		Marker:        "is" + name + "Variant",
		DiscType:      name + "Type",
		DiscJSONName:  casing.FieldSnake(form.Discriminator),
	}

	embedBases := func(refs []resolution.TypeRef) []embeddedType {
		var out []embeddedType
		for _, ext := range refs {
			parent, ok := ext.Resolve(data.table)
			if !ok {
				continue
			}
			out = append(
				out,
				embeddedType{ref: ext, rendered: resolveExtendsType(ext, parent, data)},
			)
		}
		return out
	}
	baseEmbeds := embedBases(form.Extends)

	for _, v := range form.Variants {
		vd := unionVariantData{
			TypeName:  casing.VariantTypeName(name, v.Name),
			ConstName: casing.VariantConstName(name, v.Name),
			Value:     v.Name,
			Doc:       doc.Get(v.Domains),
		}
		vd.Receiver = receiverName(vd.TypeName)
		embeds := append([]embeddedType{}, baseEmbeds...)
		var inlineFields []resolution.Field
		if v.Inline {
			if payload, ok := v.Type.Resolve(data.table); ok {
				pform := payload.Form.(resolution.StructForm)
				inherited, declared := resolver.VariantBases(form, v, data.table)
				embeds = embedBases(inherited)
				inlineFields = append(slices.Clone(declared), pform.Fields...)
				// A field that only restates an inherited default keeps the
				// embedded parent's declaration and contributes a fill alone.
				defaultOnly := resolver.DefaultOnlyOverrides(
					inherited, pform.Fields, data.table,
				)
				for _, f := range inlineFields {
					if !defaultOnly.Contains(f.Name) {
						vd.Fields = append(vd.Fields, processField(f, data))
					}
					vd.DefaultFills = append(
						vd.DefaultFills,
						goDefaultFills(f, data)...)
					if validateSkip(f, data) {
						continue
					}
					if chk, ok := goEnumCheck(f, data); ok {
						vd.EnumChecks = append(vd.EnumChecks, chk)
					}
					vd.ConstraintChecks = append(
						vd.ConstraintChecks,
						goConstraintChecks(f, data)...)
				}
			}
		} else {
			embeds = append(
				embeds,
				embeddedType{
					ref:      v.Type,
					rendered: data.resolver.ResolveTypeRef(v.Type, data.ctx),
				},
			)
		}
		for _, e := range embeds {
			vd.Embeds = append(vd.Embeds, e.rendered)
		}
		vd.DefaultRecurse = embedRecurseSteps(
			embeds,
			inlineFields,
			data,
			defaultsHasOwn,
			neverSkip,
		)
		vd.ValidateRecurse = embedRecurseSteps(
			embeds,
			inlineFields,
			data,
			validateHasOwn,
			validateSkip,
		)
		vd.NeedsApplyDefaults = len(vd.DefaultRecurse) > 0 ||
			len(vd.DefaultFills) > 0
		vd.NeedsValidate = len(vd.ValidateRecurse) > 0 ||
			len(vd.EnumChecks) > 0 || len(vd.ConstraintChecks) > 0
		if vd.NeedsApplyDefaults {
			ud.NeedsApplyDefaults = true
		}
		if vd.NeedsValidate {
			ud.NeedsValidate = true
		}
		if hasSliceRecurse(vd.ValidateRecurse) {
			data.AddExternal(strconvImportPath)
		}
		ud.Variants = append(ud.Variants, vd)
	}
	if ud.NeedsValidate {
		data.AddExternal(validateImportPath)
	}
	return ud
}

// embeddedType pairs an embedded type reference with its rendered Go type, so the
// recursion predicate can inspect the type while the field selector is derived from
// the rendered name.
type embeddedType struct {
	ref      resolution.TypeRef
	rendered string
}

// embedRecurseSteps returns the nested-method steps for a type that embeds others: a
// value step per embedded type that (transitively) needs the method, followed by the
// steps for any direct fields. Embed steps carry no JSONName, since an embedded type's
// fields are promoted to the embedder's level and take no Validate path segment.
func embedRecurseSteps(
	embeds []embeddedType,
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
