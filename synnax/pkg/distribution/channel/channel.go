// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"encoding/binary"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/telem"
	"strconv"
)

// Key represents a unique identifier for a Channel. This value is guaranteed to be
// unique across the entire cluster. It is composed of a uint32 Key representing the
// node holding the lease on the channel, and a uint16 key representing a unique
// node-local identifier.
type Key uint32

// NewKey generates a new Key from the provided components.
func NewKey(nodeKey core.NodeKey, localKey storage.ChannelKey) (key Key) {
	return Key(nodeKey)<<16 | Key(localKey)
}

// NodeKey returns the id of the node embedded in the key. This node is the leaseholder
// node for the Channel.
func (c Key) NodeKey() core.NodeKey { return core.NodeKey(c >> 16) }

// LocalKey returns a unique identifier for the Channel within the leaseholder node's
// storage.TS db. This value is NOT guaranteed to be unique across the entire cluster.
func (c Key) LocalKey() storage.ChannelKey {
	return storage.ChannelKey(c & 0x0000FFFF)
}

// Lease implements the proxy.Entry interface.
func (c Key) Lease() core.NodeKey { return c.NodeKey() }

// Bytes returns the Key as a byte slice.
func (c Key) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(c))
	return b
}

const strKeySep = "-"

// String returns the Key as a string in the format of "NodeKey-CesiumKey", so
// a channel with a NodeKey of 1 and a CesiumKey of 2 would return "1-2".
func (c Key) String() string {
	return strconv.Itoa(int(c.NodeKey())) + strKeySep + strconv.Itoa(int(c.LocalKey()))
}

// Keys extends []Keys with a few convenience methods.
type Keys []Key

// KeysFromChannels returns a slice of Keys from a slice of Channel(s).
func KeysFromChannels(channels []Channel) (keys Keys) {
	for _, channel := range channels {
		keys = append(keys, channel.Key())
	}
	return keys
}

// StorageKeys calls Key.LocalKey() on each key and returns a slice with the results.
func (k Keys) StorageKeys() []storage.ChannelKey {
	keys := make([]storage.ChannelKey, len(k))
	for i, key := range k {
		keys[i] = key.LocalKey()
	}
	return keys
}

// UniqueNodeKeys returns a slice of all UNIQUE node Keys for the given keys.
func (k Keys) UniqueNodeKeys() (keys []core.NodeKey) {
	for _, key := range k {
		keys = append(keys, key.NodeKey())
	}
	return lo.Uniq(keys)
}

// Strings returns the keys as a slice of strings.
func (k Keys) Strings() []string {
	s := make([]string, len(k))
	for i, key := range k {
		s[i] = key.String()
	}
	return s
}

// Unique removes duplicate keys from the slice and returns the result.
func (k Keys) Unique() Keys { return lo.Uniq(k) }

// Difference compares two sets of keys and returns the keys that are absent in other
// followed by the keys that are absent in k.
func (k Keys) Difference(other Keys) (Keys, Keys) { return lo.Difference(k, other) }

// Channel is a collection is a container representing a collection of samples across
// a time range. The data within a channel typically arrives from a single source. This
// can be a physical sensor, software sensor, metric, event, or any other entity that
// emits regular, consistent, and time-ordered values.
//
// This Channel type (for the distribution layer) extends a cesium.DB's channel via
// composition to add fields necessary for cluster wide distribution.
//
// Key Channel "belongs to" a specific Node. Because delta is oriented towards data collection
// close to the hardware, it's natural to assume a sensor writes to one and only device.
// For example, we may have a temperature sensor for a carbon fiber oven connected to a
// Linux box. The temperature sensor is a Channel that writes to Node residing on the
// Linux box.
//
// Array for a channel can only be written through the leaseholder. This helps solve a lot
// of consistency and atomicity issues.
type Channel struct {
	// Name is a human-readable name for the channel. This name does not have to be
	// unique.
	Name string `json:"name" msgpack:"name"`
	// NodeKey is the leaseholder node for the channel.
	NodeKey core.NodeKey `json:"node_id" msgpack:"node_id"`
	// DataType is the data type for the channel.
	DataType telem.DataType `json:"data_type" msgpack:"data_type"`
	// IsIndex is set to true if the channel is an index channel. LocalIndex channels must
	// be int64 values written in ascending order. LocalIndex channels are most commonly
	// unix nanosecond timestamps.
	IsIndex bool `json:"is_index" msgpack:"is_index"`
	// Rate sets the rate at which the channels values are written. This is used to
	// determine the timestamp of each sample.
	Rate telem.Rate `json:"rate" msgpack:"rate"`
	// Key is a unique identifier for the channel within a cesium.DB. If not set when
	// creating a channel, a unique key will be generated.
	StorageKey storage.ChannelKey `json:"storage_key" msgpack:"storage_key"`
	// LocalIndex is the channel used to index the channel's values. The LocalIndex is used to
	// associate a value with a timestamp. If zero, the channel's data will be indexed
	// using its rate. One of LocalIndex or Rate must be non-zero.
	LocalIndex storage.ChannelKey `json:"local_index" msgpack:"local_index"`
}

// Key returns the key for the Channel.
func (c Channel) Key() Key { return NewKey(c.NodeKey, c.StorageKey) }

// Index returns the key for the Channel's index channel.
func (c Channel) Index() Key { return NewKey(c.NodeKey, c.LocalIndex) }

// GorpKey implements the gorp.Entry interface.
func (c Channel) GorpKey() Key { return c.Key() }

// SetOptions implements the gorp.Entry interface. Returns a set of options that
// tell an aspen.DB to properly lease the Channel to the node it will be recording data
// from.
func (c Channel) SetOptions() []interface{} { return []interface{}{c.Lease()} }

// Lease implements the proxy.UnaryServer interface.
func (c Channel) Lease() core.NodeKey { return c.NodeKey }

func (c Channel) Storage() storage.Channel {
	s := storage.Channel{
		Key:      c.Key().String(),
		DataType: c.DataType,
		IsIndex:  c.IsIndex,
		Rate:     c.Rate,
	}
	if c.LocalIndex != 0 {
		s.Index = c.Index().String()
	}
	return s
}

func toStorage(channels []Channel) []storage.Channel {
	return lo.Map(channels, func(channel Channel, _ int) storage.Channel {
		return channel.Storage()
	})
}
