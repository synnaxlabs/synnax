// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/x/errors"

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

// Base returns the parts every field encoding shares, or the zero BaseField when f has
// no encoding.
func (f Field) Base() BaseField {
	switch v := f.Variant.(type) {
	case BinaryField:
		return v.BaseField
	case DelimitedField:
		return v.BaseField
	case TaggedField:
		return v.BaseField
	default:
		return BaseField{}
	}
}

// SetBase replaces the parts every field encoding shares. It panics when f has no
// encoding.
func (f *Field) SetBase(b BaseField) {
	switch v := f.Variant.(type) {
	case BinaryField:
		v.BaseField = b
		f.Variant = v
	case DelimitedField:
		v.BaseField = b
		f.Variant = v
	case TaggedField:
		v.BaseField = b
		f.Variant = v
	default:
		panic(errors.Newf("field has no encoding: %T", v))
	}
}
