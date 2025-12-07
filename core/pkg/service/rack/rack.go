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
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

// Key is a unique identifier for a rack. Each rack is leased to a particular
// node in the cluster. Why this over a UUID?
//
// The reason comes down to task configuration and communication mechanisms. Task
// configuration signals are passed down through gossip operations, which are much
// slower than regular channel communication. This means that gossip propagation
// through a large cluster means that it can take 15s+ for a task to be received and
// configured by a rack. By leasing a rack to the node it connects to, we can minimize
// the number of hops and the time it takes for a task to be configured.
//
// The downside is that it makes it challenging to move tasks between racks.
//
// The first 16 bits are the node key, and the last 16 bits are a unique, sequential
// key for the rack on the node.
type Key uint32

// NewKey instantiates a new rack key from its node and local key components.
func NewKey(node cluster.NodeKey, localKey uint16) Key {
	return Key(uint32(node)<<16 | uint32(localKey))
}

// Node returns the node that the rack is leased to.
func (k Key) Node() cluster.NodeKey { return cluster.NodeKey(k >> 16) }

// LocalKey returns unique key for the rack on its leaseholder node.
func (k Key) LocalKey() uint16 { return uint16(uint32(k) & 0xFFFF) }

// OntologyID returns the unique ontology identifier for the rack.
func (k Key) OntologyID() ontology.ID { return OntologyID(k) }

// IsZero returns true if the key is invalid i.e. it's Node or LocalKey is zero.
func (k Key) IsZero() bool { return k == 0 }

// String implements fmt.Stringer.
func (k Key) String() string { return strconv.Itoa(int(k)) }

func (k *Key) DecodeMsgpack(dec *msgpack.Decoder) error {
	n, err := binary.UnmarshalMsgpackUint32(dec)
	if err != nil {
		return err
	}
	*k = Key(n)
	return nil
}

type StatusDetails struct {
	Rack Key `json:"rack" msgpack:"rack"`
}

type Status = status.Status[StatusDetails]

// Rack represents a driver that can communicate with devices and execute tasks.
type Rack struct {
	// Key is a unique identifier for a rack. This key is tied to a specific node.
	Key Key `json:"key" msgpack:"key"`
	// Name is the name for the rack.
	Name string `json:"name" msgpack:"name"`
	// TaskCounter is the total number of tasks that have been created on the rack,
	// and is used to assign keys to tasks.
	TaskCounter uint32 `json:"task_counter" msgpack:"task_counter"`
	// Embedded sets whether the rack is built-in to the Synnax node, or it is an
	// external rack.
	Embedded bool `json:"embedded" msgpack:"embedded"`
	// Status is the current state of the rack.
	Status *Status `json:"status" msgpack:"status"`
}

var _ gorp.Entry[Key] = Rack{}

// GorpKey implements gorp.Entry.
func (r Rack) GorpKey() Key { return r.Key }

// SetOptions implements gorp.Entry.
func (r Rack) SetOptions() []any { return []any{r.Key.Node()} }

// Validate implements config.Config.
func (r Rack) Validate() error {
	v := validate.New("rack")
	validate.NonZero(v, "Key", r.Key)
	validate.NotEmptyString(v, "Name", r.Name)
	return v.Error()
}
