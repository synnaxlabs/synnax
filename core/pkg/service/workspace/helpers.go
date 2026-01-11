// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package workspace

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

var _ gorp.Entry[uuid.UUID] = Workspace{}

// GorpKey implements gorp.Entry.
func (w Workspace) GorpKey() uuid.UUID { return w.Key }

// SetOptions implements gorp.Entry.
func (w Workspace) SetOptions() []any { return nil }

// OntologyID returns the ontology.ID of the resource.
func (w Workspace) OntologyID() ontology.ID { return OntologyID(w.Key) }
