// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package condition

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/gorp"
	xjson "github.com/synnaxlabs/x/json"
)

// Condition is a definition for an arbitrary condition that can be met.
type Condition struct {
	Key    uuid.UUID    `json:"key" msgpack:"key"`
	Type   string       `json:"type" msgpack:"type"`
	Config xjson.String `json:"config" msgpack:"config"`
}

var _ gorp.Entry[uuid.UUID] = Condition{}

// GorpKey implements gorp.Entry.
func (c Condition) GorpKey() uuid.UUID { return c.Key }

// SetOptions implements gorp.Entry.
func (c Condition) SetOptions() []any { return nil }

type Result struct{}

type Checker interface {
	confluence.Source[Result]
	Start() error
	Stop() error
}
