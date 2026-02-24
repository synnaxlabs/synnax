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

var _ gorp.Entry[uuid.UUID] = Role{}

// GorpKey implements the gorp.Entry interface.
func (r Role) GorpKey() uuid.UUID { return r.Key }

// SetOptions implements the gorp.Entry interface.
func (r Role) SetOptions() []any { return nil }

// OntologyID returns the ontology ID for this role.
func (r Role) OntologyID() ontology.ID { return OntologyID(r.Key) }
