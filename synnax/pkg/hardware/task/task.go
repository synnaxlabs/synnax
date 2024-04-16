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
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
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
	Key    Key    `json:"key" msgpack:"key"`
	Name   string `json:"name" msgpack:"name"`
	Type   string `json:"type" msgpack:"type"`
	Config string `json:"config" msgpack:"config"`
	State  State  `json:"state" msgpack:"state"`
}

type Status string

const (
	StatusStopped Status = "stopped"
	StatusRunning Status = "running"
	StatusFailed  Status = "failed"
)

type State struct {
	Key        Key    `json:"key" msgpack:"key"`
	Status     Status `json:"status" msgpack:"status"`
	DataSaving bool   `json:"data_saving" msgpack:"data_saving"`
	Details    string `json:"details" msgpack:"details"`
}

var _ gorp.Entry[Key] = Task{}

func (m Task) GorpKey() Key { return m.Key }

func (m Task) SetOptions() []interface{} { return []interface{}{m.Key.Rack().Node()} }

func (m Task) Rack() rack.Key { return m.Key.Rack() }
