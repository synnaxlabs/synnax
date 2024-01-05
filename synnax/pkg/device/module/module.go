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

package module

import (
	"github.com/synnaxlabs/synnax/pkg/device/rack"
	"github.com/synnaxlabs/x/gorp"
	"strconv"
)

type Key uint64

func NewKey(rack rack.Key, localKey uint32) Key {
	return Key(uint64(rack)<<32 | uint64(localKey))
}

func (k Key) Rack() rack.Key { return rack.Key(k >> 32) }

func (k Key) LocalKey() uint32 { return uint32(uint64(k) & 0xFFFFFFFF) }

func (k Key) String() string { return strconv.Itoa(int(k)) }

func (k Key) IsValid() bool { return k.Rack().IsValid() && k.LocalKey() != 0 }

type Module struct {
	Key    Key    `json:"key" msgpack:"key" gorp:"primary_key"`
	Name   string `json:"name" msgpack:"name"`
	Type   string
	Config string `json:"configuration" msgpack:"configuration"`
}

var _ gorp.Entry[Key] = Module{}

func (m Module) GorpKey() Key { return m.Key }

func (m Module) SetOptions() []interface{} { return []interface{}{m.Key.Rack().Node()} }

func (m Module) Rack() rack.Key { return m.Key.Rack() }
