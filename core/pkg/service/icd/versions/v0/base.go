// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "slices"

// FieldBases returns the parts every field of p shares, in field order, with a zero
// BaseField for a text field with no encoding or a JSON field with no type. It returns
// nil when p has no format.
func (p Payload) FieldBases() []BaseField {
	switch v := p.Variant.(type) {
	case BinaryPayload:
		bases := make([]BaseField, len(v.Fields))
		for i, f := range v.Fields {
			bases[i] = f.BaseField
		}
		return bases
	case TextPayload:
		bases := make([]BaseField, len(v.Fields))
		for i, f := range v.Fields {
			bases[i] = f.Base()
		}
		return bases
	case JSONPayload:
		bases := make([]BaseField, len(v.Fields))
		for i, f := range v.Fields {
			bases[i] = f.Base().BaseField
		}
		return bases
	default:
		return nil
	}
}

// UpdateFieldBases calls update with the shared parts of each field of p, in field
// order, and keeps what update leaves. It skips text fields with no encoding and JSON
// fields with no type. It copies the fields first, so values that shared them with p
// see no change. It does nothing when p has no format.
func (p *Payload) UpdateFieldBases(update func(*BaseField)) {
	switch v := p.Variant.(type) {
	case BinaryPayload:
		v.Fields = slices.Clone(v.Fields)
		for i := range v.Fields {
			update(&v.Fields[i].BaseField)
		}
		p.Variant = v
	case TextPayload:
		v.Fields = slices.Clone(v.Fields)
		for i := range v.Fields {
			if v.Fields[i].Variant == nil {
				continue
			}
			b := v.Fields[i].Base()
			update(&b)
			v.Fields[i].SetBase(b)
		}
		p.Variant = v
	case JSONPayload:
		v.Fields = slices.Clone(v.Fields)
		for i := range v.Fields {
			if v.Fields[i].Variant == nil {
				continue
			}
			b := v.Fields[i].Base()
			update(&b.BaseField)
			v.Fields[i].SetBase(b)
		}
		p.Variant = v
	}
}
