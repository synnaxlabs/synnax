// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
	"strconv"
)

type Key uint32

func NewKey(node core.NodeKey, localKey uint16) Key {
	return Key(uint32(node)<<16 | uint32(localKey))
}

func (k Key) Node() core.NodeKey { return core.NodeKey(k >> 16) }

func (k Key) LocalKey() uint16 { return uint16(uint32(k) & 0xFFFF) }

func (k Key) OntologyID() ontology.ID { return OntologyID(k) }

func (k Key) IsValid() bool { return k.Node() != 0 && k.LocalKey() != 0 }

// String implements fmt.Stringer.
func (k Key) String() string { return strconv.Itoa(int(k)) }

type Rack struct {
	Key         Key    `json:"key" msgpack:"key"`
	Name        string `json:"name" msgpack:"name"`
	TaskCounter uint32 `json:"task_counter" msgpack:"task_counter"`
}

var _ gorp.Entry[Key] = Rack{}

// GorpKey implements gorp.Entry.
func (r Rack) GorpKey() Key { return r.Key }

// SetOptions implements gorp.Entry.
func (r Rack) SetOptions() []interface{} { return []interface{}{r.Key.Node()} }

// Validate implements config.Config.
func (r Rack) Validate() error {
	v := validate.New("rack")
	validate.NonZero(v, "Task", r.Key)
	validate.NotEmptyString(v, "Name", r.Name)
	return v.Error()
}
