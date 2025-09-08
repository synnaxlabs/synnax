// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/spatial"
)

// Arc is a representation of an arc automation stored within the cluster
// meta-data store.
type Arc struct {
	// Key is a unique key for the automation.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Name is a human-readable name.
	Name string `json:"name" msgpack:"name"`
	// Graph is the raw representation of the arc program in its
	// graph format. Note that this graph does not necessarily represent
	// a valid arc program.
	Graph Graph `json:"graph" msgpack:"graph"`
	// Text is the raw representation of the arc program in its next format.
	// Note that this text content does not necessarily represent a valid arg program.
	Text Text `json:"text" msgpack:"text"`
	// IR is the pre-compiled intermediate representation of the arc program.
	// It is treated as a validated source of truth, and is used as a 'pivot' between
	// converting from the graph based representation to the text representation
	// and vice versa.
	IR IR `json:"ir" msgpack:"ir"`
}

var _ gorp.Entry[uuid.UUID] = Arc{}

// GorpKey implements gorp.Entry.
func (s Arc) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (s Arc) SetOptions() []any { return nil }

type IR struct{}

type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Text struct {
	Contents string `json:"contents"`
}

type Handle struct {
	Node  string `json:"node"`
	Param string `json:"param"`
}

type Edge struct {
	Source Handle `json:"source"`
	Target Handle `json:"target"`
}

type Node struct {
	Key      string         `json:"key"`
	Type     string         `json:"type"`
	Position spatial.XY     `json:"position"`
	Config   map[string]any `json:"config"`
}
