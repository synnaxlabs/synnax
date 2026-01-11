// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ranger implements a service for managing ranges in a Synnax cluster. A range
// is a user defined region of time in a Synnax cluster. They act as a method for
// labeling and categorizing data.
package ranger

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

var _ gorp.Entry[uuid.UUID] = Range{}

// GorpKey implements gorp.Entry.
func (r Range) GorpKey() uuid.UUID { return r.Key }

// SetOptions implements gorp.Entry.
func (r Range) SetOptions() []any { return nil }

// OntologyID returns the semantic ID for this range to look it up from within the
func (r Range) OntologyID() ontology.ID { return OntologyID(r.Key) }
