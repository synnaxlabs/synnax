// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

// Role is a named collection of policies that can be assigned to users.
// Roles enable easier permission management by grouping policies together.
// When a user is assigned a role, they receive all permissions from that role's policies.
type Role struct {
	// Name is the human-readable name of the role (e.g., "Administrator", "Engineer").
	Name string `json:"name" msgpack:"name"`
	// Description explains what permissions this role provides.
	Description string `json:"description" msgpack:"description"`
	// Key is a unique UUID to identify the role.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Internal indicates whether the role is built-in to the system.
	Internal bool `json:"internal" msgpack:"internal"`
}

var _ gorp.Entry[uuid.UUID] = Role{}

// GorpKey implements the gorp.Entry interface.
func (r Role) GorpKey() uuid.UUID { return r.Key }

// SetOptions implements the gorp.Entry interface.
func (r Role) SetOptions() []any { return nil }

// OntologyID returns the ontology ID for this role.
func (r Role) OntologyID() ontology.ID { return OntologyID(r.Key) }
