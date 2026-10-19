// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = View{}

// Base returns the fields shared by every variant of the view. A view with no variant
// has a zero Base.
func (v View) Base() Base {
	switch variant := v.Variant.(type) {
	case RangeView:
		return variant.Base
	case StatusView:
		return variant.Base
	}
	return Base{}
}

// Type returns the type of the view's variant, or an empty type if it has none.
func (v View) Type() ViewType {
	switch v.Variant.(type) {
	case RangeView:
		return RangeViewType
	case StatusView:
		return StatusViewType
	}
	return ""
}

// WithKey returns a copy of the view with its key set to key.
func (v View) WithKey(key Key) View {
	switch variant := v.Variant.(type) {
	case RangeView:
		variant.Key = key
		v.Variant = variant
	case StatusView:
		variant.Key = key
		v.Variant = variant
	}
	return v
}

// GorpKey returns the view's storage key.
func (v View) GorpKey() Key { return v.Base().Key }

// SetOptions returns no additional storage options.
func (View) SetOptions() []any { return nil }

// OntologyID returns the view's ontology identifier.
func (v View) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeView, Key: v.Base().Key.String()}
}
