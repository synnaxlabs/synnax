// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

// Log is the data for a visualization component used to view a log of telemetry over
// time.
type Log struct {
	// Key is a unique identifier for the log.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Name is a human-readable name for the log.
	Name string `json:"name" msgpack:"name"`
	// Data is JSON-encoded data for the log.
	Data string `json:"data" msgpack:"data"`
}

var _ gorp.Entry[uuid.UUID] = Log{}

// GorpKey implements gorp.Entry.
func (l Log) GorpKey() uuid.UUID { return l.Key }

// SetOptions implements gorp.Entry.
func (l Log) SetOptions() []any { return nil }
