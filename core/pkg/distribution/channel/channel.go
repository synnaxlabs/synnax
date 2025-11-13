// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/unsafe"
	"github.com/synnaxlabs/x/validate"
)

// Key represents a unique identifier for a Channel. This value is guaranteed to be
// unique across the entire cluster. It is composed of a uint32 Key representing the
// node holding the lease on the channel, and a uint16 key representing a unique
// node-local identifier.
type Key uint32

// LocalKey represents a unique identifier for a Channel with relation to its leaseholder.
// It has a maximum addressable space of 2^20 (1,048,576) unique keys. This is the limiting
// factor of the number of channels that can be created per node in Synnax.
type LocalKey types.Uint20

// NewKey generates a new Key from the provided components.
func NewKey(nodeKey cluster.NodeKey, localKey LocalKey) (key Key) {
	// Node key is the first 12 bits,
	k1 := uint32(nodeKey) << 20
	// Local key is the last 20 bits
	k2 := uint32(localKey)
	return Key(k1 | k2)
}

// MustParseKey is a convenience function that wraps ParseKey and panics if the key
// has an invalid format.
func MustParseKey(key string) Key { return lo.Must(ParseKey(key)) }

// ParseKey attempts to parse the string representation of a Key into a Key.
func ParseKey(s string) (Key, error) {
	k, err := strconv.Atoi(s)
	if err != nil {
		return Key(0), errors.Wrapf(validate.Error, "%s is not a valid channel key", s)
	}
	return Key(k), nil
}

// Leaseholder returns the id of the node embedded in the key. This node is the leaseholder
// node for the Channel.
func (c Key) Leaseholder() cluster.NodeKey { return cluster.NodeKey(c >> 20) }

// Free returns true when the channel has a leaseholder node i.e. it is not a non-leased
// virtual channel.
func (c Key) Free() bool { return c.Leaseholder() == cluster.Free }

// StorageKey returns the storage layer representation of the channel key.
func (c Key) StorageKey() ts.ChannelKey { return ts.ChannelKey(c) }

// LocalKey returns the local key for the Channel. See the LocalKey type for more.
func (c Key) LocalKey() LocalKey { return LocalKey(c & 0xFFFFF) }

// Lease implements the proxy.Entry interface, which routes Channel operations to the
// correct node in the cluster.
func (c Key) Lease() cluster.NodeKey { return c.Leaseholder() }

// String implements fmt.Stringer.
func (c Key) String() string { return strconv.Itoa(int(c)) }

// Keys extends []Key with a few convenience methods.
type Keys []Key

// KeysFromChannels returns a slice of Keys from a slice of Channel(s).
func KeysFromChannels(channels []Channel) (keys Keys) {
	return lo.Map(channels, func(channel Channel, _ int) Key { return channel.Key() })
}

// Names returns the names of the channels.
func Names(channels []Channel) []string {
	return lo.Map(channels, func(channel Channel, _ int) string { return channel.Name })
}

// KeysFromUint32 returns a slice of Keys from a slice of uint32. NOTE: This does
// not copy the slice, it just reinterprets the memory.
func KeysFromUint32(keys []uint32) Keys { return unsafe.ReinterpretSlice[uint32, Key](keys) }

// KeysFromOntologyIDs returns a slice of Keys from a slice of ontology.ID(s). This
// function will skip any ontology.ID(s) that are not of the correct type.
func KeysFromOntologyIDs(ids []ontology.ID) (keys Keys, err error) {
	keys = make(Keys, 0, len(ids))
	var key Key
	for _, id := range ids {
		if id.Type == OntologyType {
			key, err = ParseKey(id.Key)
			if err != nil {
				return
			}
			keys = append(keys, key)
		}
	}
	return
}

// Storage returns the storage layer representation of the channel keys.
func (k Keys) Storage() []ts.ChannelKey { return k.Uint32() }

// Uint32 converts the Keys to a slice of uint32.
func (k Keys) Uint32() []uint32 { return unsafe.ReinterpretSlice[Key, uint32](k) }

// UniqueLeaseholders returns a slice of all UNIQUE leaseholders for the given Keys.
func (k Keys) UniqueLeaseholders() (keys []cluster.NodeKey) {
	for _, key := range k {
		keys = append(keys, key.Leaseholder())
	}
	return lo.Uniq(keys)
}

func (k Keys) Local() []LocalKey {
	return lo.Map(k, func(k Key, _ int) LocalKey { return k.LocalKey() })
}

// Strings returns the keys as a slice of strings.
func (k Keys) Strings() []string {
	return lo.Map(k, func(key Key, _ int) string { return key.String() })
}

// Contains returns true if the slice contains the given key, false otherwise.
func (k Keys) Contains(key Key) bool {
	return slices.Contains(k, key)
}

// Unique removes duplicate keys from the slice and returns the result.
func (k Keys) Unique() Keys { return lo.Uniq(k) }

// Difference compares two sets of keys and returns the keys that are absent in other
// followed by the keys that are absent in k.
func (k Keys) Difference(other Keys) (Keys, Keys) { return lo.Difference(k, other) }

type Operation struct {
	Type         string         `json:"type"`
	ResetChannel Key            `json:"reset_channel"`
	Duration     telem.TimeSpan `json:"duration"`
}

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
// Series for a channel can only be written through the leaseholder. This helps solve a lot
// of consistency and atomicity issues.
type Channel struct {
	// Name is a human-readable name for the channel. This name does not have to be
	// unique.
	Name string `json:"name" msgpack:"name"`
	// Leaseholder is the leaseholder node for the channel.
	Leaseholder cluster.NodeKey `json:"node_id" msgpack:"node_id"`
	// DataType is the data type for the channel.
	DataType telem.DataType `json:"data_type" msgpack:"data_type"`
	// IsIndex is set to true if the channel is an index channel. LocalIndex channels must
	// be int64 values written in ascending order. LocalIndex channels are most commonly
	// unix nanosecond timestamps.
	IsIndex bool `json:"is_index" msgpack:"is_index"`
	// LocalKey is a unique identifier for the channel with relation to its leaseholder.
	// When creating a channel, a unique key will be generated.
	LocalKey LocalKey `json:"local_key" msgpack:"local_key"`
	// LocalIndex is the channel used to index the channel's values. The LocalIndex is
	// used to associate a value with a timestamp. If zero, the channel's data will be
	// indexed using its rate. One of LocalIndex or Rate must be non-zero.
	LocalIndex LocalKey `json:"local_index" msgpack:"local_index"`
	// Virtual is set to true if the channel is a virtual channel. The data from virtual
	// channels is not persisted into the DB.
	Virtual bool `json:"virtual" msgpack:"virtual"`
	// Concurrency sets the policy for concurrent writes to the same region of the
	// channel's data. Only virtual channels can have a policy of control.Shared.
	Concurrency control.Concurrency `json:"concurrency" msgpack:"concurrency"`
	// Internal determines if a channel is a channel created by Synnax or
	// created by the user.
	Internal   bool        `json:"internal" msgpack:"internal"`
	Operations []Operation `json:"operations" msgpack:"operations"`
	// Requires is only used for calculated channels, and specifies the channels that
	// are required for the calculation.
	Requires Keys `json:"requires" msgpack:"requires"`
	// Expression is only used for calculated channels, and specifies the Lua expression
	// to evaluate the calculated value.
	Expression string `json:"expression" msgpack:"expression"`
}

func (c Channel) IsCalculated() bool {
	return c.Expression != ""
}

func (c Channel) IsLegacyCalculated() bool {
	return len(c.Requires) > 0
}

// Equals returns true if the two channels are meaningfully equal to each other. This
// function should be used instead of a direct comparison, as it takes into account
// the contents of the Requires field, ignoring the order of the keys.
// If the exclude parameter is provided, the function will ignore the fields specified
// in the exclude parameter.
func (c Channel) Equals(other Channel, exclude ...string) bool {
	comparisons := []struct {
		field string
		equal bool
	}{
		{"Name", c.Name == other.Name},
		{"Leaseholder", c.Leaseholder == other.Leaseholder},
		{"DataType", c.DataType == other.DataType},
		{"IsIndex", c.IsIndex == other.IsIndex},
		{"LocalKey", c.LocalKey == other.LocalKey},
		{"LocalIndex", c.LocalIndex == other.LocalIndex},
		{"Virtual", c.Virtual == other.Virtual},
		{"Concurrency", c.Concurrency == other.Concurrency},
		{"Internal", c.Internal == other.Internal},
		{"Expression", c.Expression == other.Expression},
	}

	for _, comp := range comparisons {
		if !comp.equal && !lo.Contains(exclude, comp.field) {
			return false
		}
	}

	if !lo.Contains(exclude, "Operations") {
		if !slices.Equal(c.Operations, other.Operations) {
			return false
		}
	}

	if !lo.Contains(exclude, "Requires") {
		slices.Sort(c.Requires)
		slices.Sort(other.Requires)
		if !slices.Equal(c.Requires, other.Requires) {
			return false
		}
	}

	return true
}

// String implements stringer, returning a nicely formatted string representation of the
// Channel.
func (c Channel) String() string {
	if c.Name != "" {
		return fmt.Sprintf("[%s]<%d>", c.Name, c.Key())
	}
	return fmt.Sprintf("<%d>", c.Key())
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
func (c Channel) SetOptions() []any {
	if c.Free() {
		return []any{cluster.Bootstrapper}
	}
	return []any{c.Lease()}
}

// Lease implements the proxy.UnaryServer interface.
func (c Channel) Lease() cluster.NodeKey { return c.Leaseholder }

// Free returns true if the channel is leased to a particular node i.e. it is not
// a non-leased virtual channel.
func (c Channel) Free() bool { return c.Leaseholder == cluster.Free }

// Storage returns the storage layer representation of the channel for creation
// in the storage ts.DB.
func (c Channel) Storage() ts.Channel {
	return ts.Channel{
		Key:         c.Key().StorageKey(),
		Name:        c.Name,
		IsIndex:     c.IsIndex,
		DataType:    c.DataType,
		Index:       ts.ChannelKey(c.Index()),
		Virtual:     c.Virtual,
		Concurrency: c.Concurrency,
	}
}

// toStorage converts a slice of channels to their storage layer equivalent.
func toStorage(channels []Channel) []ts.Channel {
	return lo.Map(channels, func(c Channel, _ int) ts.Channel { return c.Storage() })
}
