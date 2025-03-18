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
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/x/errors"
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

type Status string

const (
	InfoStateVariant    Status = "info"
	SuccessStateVariant Status = "success"
	ErrorStateVariant   Status = "error"
	WarningStateVariant Status = "warning"
)

// Details is a custom type based on string
type Details string

var detailsCodec = &binary.JSONCodec{}

func NewStaticDetails(data interface{}) Details {
	b, err := detailsCodec.Encode(nil, data)
	if err != nil {
		panic(err)
	}
	return Details(b)
}

// UnmarshalJSON implements the json.Unmarshaler interface for Details.
// It should correctly handle a raw JSON string or a JSON object/array.
func (d *Details) UnmarshalJSON(data []byte) error {
	// Try to unmarshal data into a plain string
	var plainString string
	if err := json.Unmarshal(data, &plainString); err == nil {
		*d = Details(plainString)
		return nil
	}

	// If the above fails, it means the data might be an object or an array,
	// so we re-marshal it into a string regardless of its type.
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return errors.New("input data is neither a plain string nor valid JSON")
	}

	// Marshal the object back to string
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	*d = Details(bytes)
	return nil
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
	Variant Status `json:"variant" msgpack:"variant"`
	// Details is an arbitrary string that provides additional information about the
	// state.
	Details Details `json:"details" msgpack:"details"`
}

var _ gorp.Entry[Key] = State{}

// GorpKey implements the gorp.Entry interface.
func (s State) GorpKey() Key { return s.Task }

// SetOptions implements the gorp.Entry interface.
func (s State) SetOptions() []interface{} { return []interface{}{s.Task.Rack().Node()} }

// CustomTypeName implements types.CustomTypeName to ensure that State struct does
// not conflict with any other types in gorp.
func (s State) CustomTypeName() string { return "TaskState" }
