// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[uuid.UUID] = Project{}

// GorpKey implements gorp.Entry.
func (p Project) GorpKey() uuid.UUID { return p.Key }

// SetOptions implements gorp.Entry.
func (p Project) SetOptions() []any { return nil }

// OntologyID returns the ontology.ID of the resource.
func (p Project) OntologyID() ontology.ID { return OntologyID(p.Key) }
