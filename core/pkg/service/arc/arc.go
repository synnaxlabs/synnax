// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/x/gorp"
)

// StatusDetails is the status details type for arc statuses.
type StatusDetails struct{ Running bool }

// Arc is a representation of an arc automation stored within the cluster meta-data
// store.
type Arc struct {
	// Text is the raw representation of the arc program in its next format. Note that
	// this text content does not necessarily represent a valid arg program.
	Text    text.Text `json:"text" msgpack:"text"`
	Version string    `json:"version" msgpack:"version"`
	// Name is a human-readable name.
	Name string `json:"name" msgpack:"name"`
	// Module is the compiled Arc module containing IR and WASM bytecode. This field is
	// computed on-demand and not persisted to the database.
	Module module.Module `json:"module" msgpack:"-"`
	// Graph is the raw representation of the arc program in its graph format. Note that
	// this graph does not necessarily represent a valid arc program.
	Graph graph.Graph `json:"graph" msgpack:"graph"`
	// Key is a unique key for the automation.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Mode indicates whether this arc uses "graph" or "text" representation.
	Mode string `json:"mode" msgpack:"mode"`
}

var _ gorp.Entry[uuid.UUID] = Arc{}

// GorpKey implements gorp.Entry.
func (s Arc) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (s Arc) SetOptions() []any { return nil }
