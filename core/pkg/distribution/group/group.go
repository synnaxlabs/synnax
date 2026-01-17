// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/zyn"
)

// Group is a simple grouping of resources within the cluster's ontology.
type Group struct {
	// Name is the name for the group.
	Name string `json:"name" msgpack:"name"`
	// Key is the unique identifier for the group. Will be generated on creation if not
	// set.
	Key uuid.UUID `json:"key" msgpack:"key"`
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"name": zyn.String(),
})

var _ gorp.Entry[uuid.UUID] = Group{}

// GorpKey implements gorp.Entry.
func (g Group) GorpKey() uuid.UUID { return g.Key }

// SetOptions implements gorp.Entry.
func (g Group) SetOptions() []any { return nil }

// OntologyID returns the ontology.ID for the group.
func (g Group) OntologyID() ontology.ID { return OntologyID(g.Key) }

// IsZero implements ZeroAble.
func (g Group) IsZero() bool { return g.Key == uuid.Nil && len(g.Name) == 0 }
