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
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = Panel{}

// GorpKey implements gorp.Entry.
func (p Panel) GorpKey() Key { return p.Key }

// SetOptions implements gorp.Entry.
func (p Panel) SetOptions() []any { return nil }

// OntologyID returns the ontology.ID of the resource.
func (p Panel) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypePanel, Key: p.Key.String()}
}

// Key returns the stable identifier of the tab regardless of its content variant.
// Returns uuid.Nil for a Tab with no variant set.
func (t Tab) Key() uuid.UUID {
	switch v := t.Variant.(type) {
	case TabResource:
		return v.Key
	case TabView:
		return v.Key
	case TabEmpty:
		return v.Key
	default:
		return uuid.Nil
	}
}
