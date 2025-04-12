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
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/x/gorp"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/types"
)

type Key types.StringParseableUint64

func NewKey(rack rack.Key, localKey uint32) Key {
	return Key(uint64(rack)<<32 | uint64(localKey))
}

func (k Key) Rack() rack.Key { return rack.Key(k >> 32) }

func (k Key) LocalKey() uint32 { return uint32(uint64(k) & 0xFFFFFFFF) }

func (k Key) String() string { return strconv.Itoa(int(k)) }

func (k Key) IsValid() bool { return !k.Rack().IsZero() && k.LocalKey() != 0 }

func (k *Key) UnmarshalJSON(b []byte) error {
	// Try to unmarshal as a number first.
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		*k = Key(n)
		return nil
	}

	// Unmarshal as a string.
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	// Parse the string.
	n, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	*k = Key(n)
	return nil
}

type Task struct {
	Key      Key    `json:"key" msgpack:"key"`
	Name     string `json:"name" msgpack:"name"`
	Type     string `json:"type" msgpack:"type"`
	Config   string `json:"config" msgpack:"config"`
	State    *State `json:"state" msgpack:"state"`
	Internal bool   `json:"internal" msgpack:"internal"`
	Snapshot bool   `json:"snapshot" msgpack:"snapshot"`
}

var _ gorp.Entry[Key] = Task{}

func (t Task) GorpKey() Key { return t.Key }

func (t Task) SetOptions() []interface{} { return []interface{}{t.Key.Rack().Node()} }

func (t Task) Rack() rack.Key { return t.Key.Rack() }

func (t Task) String() string {
	if t.Name != "" {
		return fmt.Sprintf("[%s]<%s>", t.Name, t.Key)
	}
	return t.Key.String()
}

// State represents the state of a task.
type State struct {
	// Key is used to uniquely identify the state update, and is usually used to tie
	// back a command with its corresponding state value.
	Key string `json:"key" msgpack:"key"`
	// Internal is true if the state update is for an internal task.
	Internal bool `json:"internal" msgpack:"internal"`
	// Task is the key of the task that the state update is for.
	Task Key `json:"task" msgpack:"task"`
	// Variant is the status of the task.
	Variant status.Variant `json:"variant" msgpack:"variant"`
	// Details is an arbitrary string that provides additional information about the
	// state.
	Details xjson.String `json:"details" msgpack:"details"`
}

var _ gorp.Entry[Key] = State{}

// GorpKey implements the gorp.Entry interface.
func (s State) GorpKey() Key { return s.Task }

// SetOptions implements the gorp.Entry interface.
func (s State) SetOptions() []interface{} { return []interface{}{s.Task.Rack().Node()} }

// CustomTypeName implements types.CustomTypeName to ensure that State struct does
// not conflict with any other types in gorp.
func (s State) CustomTypeName() string { return "TaskState" }
