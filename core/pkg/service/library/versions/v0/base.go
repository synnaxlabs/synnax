// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"slices"

	"github.com/synnaxlabs/x/errors"
)

// Base returns the parts every entry kind shares, or the zero BaseEntry when e has no
// kind.
func (e Entry) Base() BaseEntry {
	switch v := e.Variant.(type) {
	case EnumEntry:
		return v.BaseEntry
	case MessageEntry:
		return v.BaseEntry
	default:
		return BaseEntry{}
	}
}

// SetBase replaces the parts every entry kind shares. It panics when e has no kind.
func (e *Entry) SetBase(b BaseEntry) {
	switch v := e.Variant.(type) {
	case EnumEntry:
		v.BaseEntry = b
		e.Variant = v
	case MessageEntry:
		v.BaseEntry = b
		e.Variant = v
	default:
		panic(errors.Newf("entry has no kind: %T", v))
	}
}

// Base returns the parts every text field encoding shares, or the zero BaseField when f
// has no encoding.
func (f TextField) Base() BaseField {
	switch v := f.Variant.(type) {
	case DelimitedTextField:
		return v.BaseField
	case TaggedTextField:
		return v.BaseField
	default:
		return BaseField{}
	}
}

// SetBase replaces the parts every text field encoding shares. It panics when f has no
// encoding.
func (f *TextField) SetBase(b BaseField) {
	switch v := f.Variant.(type) {
	case DelimitedTextField:
		v.BaseField = b
		f.Variant = v
	case TaggedTextField:
		v.BaseField = b
		f.Variant = v
	default:
		panic(errors.Newf("text field has no encoding: %T", v))
	}
}

// FieldBases returns the parts every field of p shares, in field order, with a zero
// BaseField for a text field with no encoding. It returns nil when p has no format.
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
	default:
		return nil
	}
}

// UpdateFieldBases calls update with the shared parts of each field of p, in field
// order, and keeps what update leaves. It skips text fields with no encoding. It copies
// the fields first, so values that shared them with p see no change. It does nothing
// when p has no format.
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
	}
}
