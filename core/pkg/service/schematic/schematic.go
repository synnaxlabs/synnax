// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/version"
)

type Node struct {
	Key      string     `json:"key"`
	Type     string     `json:"type"`
	Position spatial.XY `json:"position"`
	ZIndex   int        `json:"zIndex"`
}

type Handle struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Edge struct {
	Key    string `json:"key"`
	Source Handle `json:"source"`
	Target Handle `json:"target"`
}

// Schematic is the data for a visualization used to view and operate a schematic of a
// hardware system.
type Schematic struct {
	// Key is a unique identifier for the schematic.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Snapshot is true if the schematic can no longer be modified.
	Snapshot bool `json:"snapshot" msgpack:"snapshot"`
	// Name is a human-readable name for the schematic.
	Name    string          `json:"name" msgpack:"name"`
	Version version.Counter `json:"version" msgpack:"version"`
	// Data is JSON-encoded data for the schematic.
	Nodes []Node                    `json:"nodes" msgpack:"nodes"`
	Edges []Edge                    `json:"edges" msgpack:"edges"`
	Props map[string]map[string]any `json:"props"`
}

var _ gorp.Entry[uuid.UUID] = Schematic{}

// GorpKey implements gorp.Entry.
func (s Schematic) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (s Schematic) SetOptions() []any { return nil }
