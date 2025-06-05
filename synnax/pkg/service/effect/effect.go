// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package effect

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/status"
)

// Effect is a definition for a condition that results in a set of effects being
// executed.
type Effect struct {
	Key     uuid.UUID `json:"key" msgpack:"key"`
	Name    string    `json:"name" msgpack:"name"`
	Slate   uuid.UUID `json:"slate" msgpack:"slate"`
	Enabled bool      `json:"enabled" msgpack:"enabled"`
}

var _ gorp.Entry[uuid.UUID] = Effect{}

// GorpKey implements gorp.Entry.
func (e Effect) GorpKey() uuid.UUID { return e.Key }

// SetOptions implements gorp.Entry.
func (e Effect) SetOptions() []any { return nil }

type State struct {
	Key     uuid.UUID      `json:"key" msgpack:"key"`
	Variant status.Variant `json:"variant" msgpack:"variant"`
	Details xjson.String   `json:"details" msgpack:"details"`
}
