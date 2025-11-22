// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"fmt"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
)

type Key uint64

func NewKey(rack rack.Key, localKey uint32) Key {
	return Key(uint64(rack)<<32 | uint64(localKey))
}

func (k Key) Rack() rack.Key { return rack.Key(k >> 32) }

func (k Key) LocalKey() uint32 { return uint32(uint64(k) & 0xFFFFFFFF) }

func (k Key) String() string { return strconv.Itoa(int(k)) }

func (k Key) IsValid() bool { return !k.Rack().IsZero() && k.LocalKey() != 0 }

func (k *Key) UnmarshalJSON(b []byte) error {
	n, err := binary.UnmarshalJSONStringUint64(b)
	*k = Key(n)
	return err
}

type Task struct {
	Key      Key     `json:"key" msgpack:"key"`
	Name     string  `json:"name" msgpack:"name"`
	Type     string  `json:"type" msgpack:"type"`
	Config   string  `json:"config" msgpack:"config"`
	Status   *Status `json:"status" msgpack:"status"`
	Internal bool    `json:"internal" msgpack:"internal"`
	Snapshot bool    `json:"snapshot" msgpack:"snapshot"`
}

var _ gorp.Entry[Key] = Task{}

func (t Task) GorpKey() Key { return t.Key }

func (t Task) SetOptions() []any { return []any{t.Key.Rack().Node()} }

func (t Task) Rack() rack.Key { return t.Key.Rack() }

func (t Task) String() string {
	if t.Name != "" {
		return fmt.Sprintf("[%s]<%s>", t.Name, t.Key)
	}
	return t.Key.String()
}

type StatusDetails struct {
	// Task is the key of the task that the state update is for.
	Task    Key            `json:"task" msgpack:"task"`
	Running bool           `json:"running" msgpack:"running"`
	Data    map[string]any `json:"data" msgpack:"data"`
}

// Status represents the state of a task.
type Status = status.Status[StatusDetails]
