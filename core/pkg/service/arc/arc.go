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

type Mode string

var (
	Text  Mode = "text"
	Graph      = "graph"
)

// StatusDetails is the status details type for arc statuses.
type StatusDetails struct{ Running bool }

// Arc is a representation of an arc automation stored within the cluster meta-data
// store.
type Arc struct {
	Text    text.Text     `json:"text" msgpack:"text"`
	Version string        `json:"version" msgpack:"version"`
	Name    string        `json:"name" msgpack:"name"`
	Mode    Mode          `json:"mode" msgpack:"mode"`
	Module  module.Module `json:"module" msgpack:"module"`
	Graph   graph.Graph   `json:"graph" msgpack:"graph"`
	Key     uuid.UUID     `json:"key" msgpack:"key"`
}

var _ gorp.Entry[uuid.UUID] = Arc{}

// GorpKey implements gorp.Entry.
func (s Arc) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (s Arc) SetOptions() []any { return nil }
