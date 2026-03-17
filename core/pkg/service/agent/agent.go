// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package agent

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

type State string

const (
	StateGenerating State = "generating"
	StateRunning    State = "running"
	StateError      State = "error"
	StateStopped    State = "stopped"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAgent Role = "agent"
)

type Message struct {
	Role    Role            `json:"role" msgpack:"role"`
	Content string          `json:"content" msgpack:"content"`
	Time    telem.TimeStamp `json:"time" msgpack:"time"`
}

type Agent struct {
	Key      uuid.UUID `json:"key" msgpack:"key"`
	Name     string    `json:"name" msgpack:"name"`
	Messages []Message `json:"messages" msgpack:"messages"`
	ArcKey   uuid.UUID `json:"arc_key" msgpack:"arc_key"`
	RackKey  rack.Key  `json:"rack_key" msgpack:"rack_key"`
	TaskKey  task.Key  `json:"task_key" msgpack:"task_key"`
	State    State     `json:"state" msgpack:"state"`
}

var _ gorp.Entry[uuid.UUID] = Agent{}

func (a Agent) GorpKey() uuid.UUID { return a.Key }

func (a Agent) SetOptions() []any { return nil }
