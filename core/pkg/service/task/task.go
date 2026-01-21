// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
	"github.com/vmihailenco/msgpack/v5"
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

func (k *Key) DecodeMsgpack(dec *msgpack.Decoder) error {
	n, err := binary.UnmarshalMsgpackUint64(dec)
	if err != nil {
		return err
	}
	*k = Key(n)
	return nil
}

type Task struct {
	Status   *Status `json:"status" msgpack:"status"`
	Name     string  `json:"name" msgpack:"name"`
	Type     string  `json:"type" msgpack:"type"`
	Config   string  `json:"config" msgpack:"config"`
	Key      Key     `json:"key" msgpack:"key"`
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
	Data    map[string]any `json:"data" msgpack:"data"`
	Cmd     string         `json:"cmd" msgpack:"cmd"`
	Task    Key            `json:"task" msgpack:"task"`
	Running bool           `json:"running" msgpack:"running"`
}

// Status represents the state of a task.
type Status = status.Status[StatusDetails]

// Command represents a command to be executed by a task.
type Command struct {
	// Type is the type of command (e.g. "start", "stop").
	Type string `json:"type"`
	// Key is the command key for acknowledgment.
	Key string `json:"key"`
	// Args contains command-specific arguments.
	Args json.RawMessage `json:"args"`
	// Task is the key of the task to execute the command on.
	Task Key `json:"task"`
}

// String returns a string representation of the command.
func (c Command) String() string {
	return fmt.Sprintf("%s (key=%s, task=%s)", c.Type, c.Key, c.Task)
}
