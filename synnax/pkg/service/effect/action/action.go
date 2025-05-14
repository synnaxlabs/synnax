// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package action

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/effect/condition"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/gorp"
	xjson "github.com/synnaxlabs/x/json"
)

// Action is a definition for an action that can be taken.
type Action struct {
	Key    uuid.UUID    `json:"key" msgpack:"key"`
	Type   string       `json:"type" msgpack:"type"`
	Config xjson.String `json:"config" msgpack:"config"`
}

var _ gorp.Entry[uuid.UUID] = Action{}

// GorpKey implements gorp.Entry.
func (c Action) GorpKey() uuid.UUID { return c.Key }

// SetOptions implements gorp.Entry.
func (c Action) SetOptions() []any { return nil }

type Taker = confluence.Sink[condition.Result]
