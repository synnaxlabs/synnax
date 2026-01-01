// Copyright 2026 Synnax Labs, Inc.
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
)

// Schematic is the data for a visualization used to view and operate a schematic of a
// hardware system.
type Schematic struct {
	// Key is a unique identifier for the schematic.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Snapshot is true if the schematic can no longer be modified.
	Snapshot bool `json:"snapshot" msgpack:"snapshot"`
	// Name is a human-readable name for the schematic.
	Name string `json:"name" msgpack:"name"`
	// Data is JSON-encoded data for the schematic.
	Data string `json:"data" msgpack:"data"`
}

var _ gorp.Entry[uuid.UUID] = Schematic{}

// GorpKey implements gorp.Entry.
func (s Schematic) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (s Schematic) SetOptions() []any { return nil }
