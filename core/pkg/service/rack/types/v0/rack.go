// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	status "github.com/synnaxlabs/synnax/pkg/service/status/types/v0"
	"github.com/synnaxlabs/x/gorp"
)

// Status is rack-specific status information including operational state.
type Status = status.Status[StatusDetails]

// Key is a unique identifier for a rack. Each rack is leased to a particular node in
// the cluster. Why this over a UUID?
//
// The reason comes down to task configuration and communication mechanisms. Task
// configuration signals are passed down through gossip operations, which are much
// slower than regular channel communication. This means that gossip propagation through
// a large cluster means that it can take 15s+ for a task to be received and configured
// by a rack. By leasing a rack to the node it connects to, we can minimize the number
// of hops and the time it takes for a task to be configured.
//
// The downside is that it makes it challenging to move tasks between racks.
//
// The first 16 bits are the node key, and the last 16 bits are a unique, sequential key
// for the rack on the node.
type Key uint32

// Node returns the node that the rack is leased to.
func (k Key) Node() node.Key { return node.Key(k >> 16) }

// LocalKey returns unique key for the rack on its leaseholder node.
func (k Key) LocalKey() uint16 { return uint16(uint32(k) & 0xFFFF) }

// OntologyID returns the unique ontology identifier for the rack.
func (k Key) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeRack, Key: k.String()}
}

// IsZero returns true if the key is unset, i.e. both its Node and LocalKey are zero.
func (k Key) IsZero() bool { return k == 0 }

// String returns the key formatted as its decimal integer value.
func (k Key) String() string { return strconv.Itoa(int(k)) }

// StatusDetails is the rack-specific detail payload carried in a rack Status.
type StatusDetails struct {
	// Rack is the key of the rack the status describes.
	Rack Key `json:"rack" msgpack:"rack"`
}

// Rack is a manager for a collection of tasks deployed on a node.
type Rack struct {
	// Key uniquely identifies the rack in the cluster.
	Key Key `json:"key" msgpack:"key"`
	// Name is a human-readable name for the rack.
	Name string `json:"name" msgpack:"name"`
	// TaskCounter is the number of tasks ever created on the rack, used to issue
	// sequential local task keys.
	TaskCounter uint32 `json:"task_counter" msgpack:"task_counter"`
	// Embedded is true when the rack runs the node's embedded driver.
	Embedded bool `json:"embedded" msgpack:"embedded"`
	// Status is the last known status of the rack, if any.
	Status *Status `json:"status,omitempty" msgpack:"status,omitempty"`
	// Integrations are the driver integrations the rack supports.
	Integrations []string `json:"integrations" msgpack:"integrations"`
}

var _ gorp.Entry[Key] = Rack{}

// GorpKey implements gorp.Entry.
func (r Rack) GorpKey() Key { return r.Key }

// SetOptions implements gorp.Entry.
func (Rack) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the rack.
func (r Rack) OntologyID() ontology.ID { return r.Key.OntologyID() }
