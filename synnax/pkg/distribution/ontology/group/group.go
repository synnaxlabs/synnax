// Copyright 2023 Synnax Labs, Inc.
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
)

// Group is a simple grouping of resources within the cluster's ontology.
type Group struct {
	// Key is the unique identifier for the group. Will be generated on creation if not
	// set.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Name is the name for the group.
	Name string `json:"name" msgpack:"name"`
}

var _ gorp.Entry[uuid.UUID] = Group{}

// GorpKey implements gorp.Entry.
func (c Group) GorpKey() uuid.UUID { return c.Key }

// SetOptions implements gorp.Entry.
func (c Group) SetOptions() []interface{} { return nil }

// OntologyID returns the ontology.ID for the group.
func (c Group) OntologyID() ontology.ID { return OntologyID(c.Key) }

// IsZero implements ZeroAble.
func (c Group) IsZero() bool { return c.Key == uuid.Nil && len(c.Name) == 0 }
