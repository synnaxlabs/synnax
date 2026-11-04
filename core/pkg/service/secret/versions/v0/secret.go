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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

var _ gorp.Entry[Key] = Secret{}

// Base returns the fields shared by every variant of the secret. A secret with no
// variant has a zero Base.
func (s Secret) Base() Base {
	switch variant := s.Variant.(type) {
	case StoredSecret:
		return variant.Base
	case FileSecret:
		return variant.Base
	}
	return Base{}
}

// Source returns where the value of the secret lives, or an empty source if the secret
// has no variant.
func (s Secret) Source() Source {
	switch s.Variant.(type) {
	case StoredSecret:
		return SourceStored
	case FileSecret:
		return SourceFile
	}
	return ""
}

// WithBase returns a copy of the secret with its shared fields set to base.
func (s Secret) WithBase(base Base) Secret {
	switch variant := s.Variant.(type) {
	case StoredSecret:
		variant.Base = base
		s.Variant = variant
	case FileSecret:
		variant.Base = base
		s.Variant = variant
	}
	return s
}

// GorpKey returns the secret's storage key.
func (s Secret) GorpKey() Key { return s.Base().Key }

// SetOptions returns no additional storage options.
func (Secret) SetOptions() []any { return nil }

// OntologyID returns the secret's ontology identifier.
func (s Secret) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeSecret, Key: s.Base().Key.String()}
}
