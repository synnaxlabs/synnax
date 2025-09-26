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
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/x/gorp"
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
	Graph graph.Graph `json:"graph" msgpack:"graph"`
	// Text is the raw representation of the arc program in its next format.
	// Note that this text content does not necessarily represent a valid arg program.
	Text text.Text `json:"text" msgpack:"text"`
	// Deploy sets whether on not the arc program should be deployed.
	Deploy bool `json:"deploy" msgpack:"deploy"`
}

var _ gorp.Entry[uuid.UUID] = Arc{}

// GorpKey implements gorp.Entry.
func (s Arc) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (s Arc) SetOptions() []any { return nil }
