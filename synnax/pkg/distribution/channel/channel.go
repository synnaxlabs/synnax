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
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
	"strconv"
	"strings"
)

// Key represents a unique identifier for a Channel. This value is guaranteed to be
// unique across the entire cluster. It is composed of a uint32 Key representing the
// node holding the lease on the channel, and a uint16 key representing a unique
// node-local identifier.
type Key uint32

// NewKey generates a new Key from the provided components.
func NewKey(nodeKey core.NodeKey, localKey uint16) (key Key) {
	return Key(nodeKey)<<16 | Key(localKey)
}

func MustParseKey(key string) Key { return lo.Must(ParseKey(key)) }

func ParseKey(s string) (k Key, err error) {
	split := strings.Split(s, "-")
	if len(split) != 2 {
		return k, errors.New("[channel] - invalid key format")
	}
	nodeID, err := strconv.Atoi(split[0])
	if err != nil {
		return k, errors.Wrapf(err, "[channel] - key - invalid node id")
	}
	localKey, err := strconv.Atoi(split[1])
	if err != nil {
		return k, errors.Wrapf(err, "[channel] - invalid local key")
	}
	return NewKey(core.NodeKey(nodeID), uint16(localKey)), nil
}

// Leaseholder returns the id of the node embedded in the key. This node is the leaseholder
// node for the Channel.
func (c Key) Leaseholder() core.NodeKey { return core.NodeKey(c >> 16) }

// StorageKey returns a unique identifier for the Channel to use with a ts.DB.
func (c Key) StorageKey() uint32 { return uint32(c) }

func (c Key) LocalKey() uint16 { return uint16(c & 0xFFFF) }

// Lease implements the proxy.Entry interface.
func (c Key) Lease() core.NodeKey { return c.Leaseholder() }

// Bytes returns the Key as a byte slice.
func (c Key) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(c))
	return b
}

const strKeySep = "-"

// String returns the Key as a string in the format of "Leaseholder-CesiumKey", so
// a channel with a Leaseholder of 1 and a CesiumKey of 2 would return "1-2".
func (c Key) String() string {
	if c == 0 {
		return ""
	}
	return strconv.Itoa(int(c.Leaseholder())) + strKeySep + strconv.Itoa(int(c.LocalKey()))
}

// Keys extends []Keys with a few convenience methods.
type Keys []Key

// KeysFromChannels returns a slice of Keys from a slice of Channel(s).
func KeysFromChannels(channels []Channel) (keys Keys) {
	nKeys := make(Keys, len(channels))
	for i, channel := range channels {
		nKeys[i] = channel.Key()
	}
	return nKeys
}

func KeysFromUint32(keys []uint32) Keys {
	nKeys := make(Keys, len(keys))
	for i, key := range keys {
		nKeys[i] = Key(key)
	}
	return nKeys
}

// Storage calls Key.StorageKey() on each key and returns a slice with the results.
func (k Keys) Storage() []ts.ChannelKey { return k.Uint32() }

// Uint32 converts the Keys to a slice of uint32.
func (k Keys) Uint32() []uint32 { return unsafe.ConvertSlice[Key, uint32](k) }

// UniqueNodeKeys returns a slice of all UNIQUE node Keys for the given keys.
func (k Keys) UniqueNodeKeys() (keys []core.NodeKey) {
	for _, key := range k {
		keys = append(keys, key.Leaseholder())
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

func (k Keys) Contains(key Key) bool {
	for _, k := range k {
		if k == key {
			return true
		}
	}
	return false
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
	// Leaseholder is the leaseholder node for the channel.
	Leaseholder core.NodeKey `json:"node_id" msgpack:"node_id"`
	// DataType is the data type for the channel.
	DataType telem.DataType `json:"data_type" msgpack:"data_type"`
	// IsIndex is set to true if the channel is an index channel. LocalIndex channels must
	// be int64 values written in ascending order. LocalIndex channels are most commonly
	// unix nanosecond timestamps.
	IsIndex bool `json:"is_index" msgpack:"is_index"`
	// Rate sets the rate at which the channels values are written. This is used to
	// determine the timestamp of each sample.
	Rate telem.Rate `json:"rate" msgpack:"rate"`
	// LocalKey is a unique identifier for the channel within the storage ts.DB. If not set
	// when creating a channel, a unique key will be generated.
	LocalKey uint16 `json:"local_key" msgpack:"local_key"`
	// LocalIndex is the channel used to index the channel's values. The LocalIndex is
	// used to associate a value with a timestamp. If zero, the channel's data will be
	// indexed using its rate. One of LocalIndex or Rate must be non-zero.
	LocalIndex uint16 `json:"local_index" msgpack:"local_index"`
}

// Key returns the key for the Channel.
func (c Channel) Key() Key { return NewKey(c.Leaseholder, c.LocalKey) }

// Index returns the key for the Channel's index channel.
func (c Channel) Index() Key {
	if c.LocalIndex == 0 {
		return 0
	}
	return NewKey(c.Leaseholder, c.LocalIndex)
}

// GorpKey implements the gorp.Entry interface.
func (c Channel) GorpKey() Key { return c.Key() }

// SetOptions implements the gorp.Entry interface. Returns a set of options that
// tell an aspen.DB to properly lease the Channel to the node it will be recording data
// from.
func (c Channel) SetOptions() []interface{} { return []interface{}{c.Lease()} }

// Lease implements the proxy.UnaryServer interface.
func (c Channel) Lease() core.NodeKey { return c.Leaseholder }

func (c Channel) Storage() ts.Channel {
	return ts.Channel{
		Key:      uint32(c.Key()),
		DataType: c.DataType,
		IsIndex:  c.IsIndex,
		Rate:     c.Rate,
		Index:    uint32(c.Index()),
	}
}

func toStorage(channels []Channel) []ts.Channel {
	return lo.Map(channels, func(channel Channel, _ int) ts.Channel {
		return channel.Storage()
	})
}
