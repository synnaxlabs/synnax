/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package task

import (
	"encoding/json"
	"fmt"
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/types"
	"strconv"
)

type Key types.StringParseableUint64

func NewKey(rack rack.Key, localKey uint32) Key {
	return Key(uint64(rack)<<32 | uint64(localKey))
}

func (k Key) Rack() rack.Key { return rack.Key(k >> 32) }

func (k Key) LocalKey() uint32 { return uint32(uint64(k) & 0xFFFFFFFF) }

func (k Key) String() string { return strconv.Itoa(int(k)) }

func (k Key) IsValid() bool { return k.Rack().IsValid() && k.LocalKey() != 0 }

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
}

func (t Task) String() string {
	if t.Name != "" {
		return fmt.Sprintf("[%s]<%s>", t.Name, t.Key)
	}
	return t.Key.String()
}

type Status string

const (
	StatusStopped Status = "stopped"
	StatusRunning Status = "running"
	StatusFailed  Status = "failed"
)

// Details is a custom type based on string
type Details string

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

type State struct {
	Key      string  `json:"key" msgpack:"key"`
	Internal bool    `json:"internal" msgpack:"internal"`
	Task     Key     `json:"task" msgpack:"task"`
	Variant  Status  `json:"variant" msgpack:"variant"`
	Details  Details `json:"details" msgpack:"details"`
}

var _ gorp.Entry[Key] = Task{}

func (t Task) GorpKey() Key { return t.Key }

func (t Task) SetOptions() []interface{} { return []interface{}{t.Key.Rack().Node()} }

func (t Task) Rack() rack.Key { return t.Key.Rack() }
