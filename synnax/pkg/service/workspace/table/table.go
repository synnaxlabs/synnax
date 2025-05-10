// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

// Table is the data for a visualization component used to view a table of telemetry over
// time.
type Table struct {
	// Key is a unique identifier for the table.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Name is a human-readable name for the table.
	Name string `json:"name" msgpack:"name"`
	// Data is JSON-encoded data for the table.
	Data string `json:"data" msgpack:"data"`
}

var _ gorp.Entry[uuid.UUID] = Table{}

// GorpKey implements gorp.Entry.
func (t Table) GorpKey() uuid.UUID { return t.Key }

// SetOptions implements gorp.Entry.
func (t Table) SetOptions() []any { return nil }
