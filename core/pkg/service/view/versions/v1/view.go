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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

var _ gorp.Entry[Key] = View{}

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

// WithName returns a copy of the view with its name set to name.
func (v View) WithName(name string) View {
	switch variant := v.Variant.(type) {
	case RangeView:
		variant.Name = name
		v.Variant = variant
	case StatusView:
		variant.Name = name
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
