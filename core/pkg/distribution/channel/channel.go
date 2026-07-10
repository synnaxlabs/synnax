// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/unsafe"
)

// LocalKey is a 20-bit unsigned integer representing the locally-unique portion of a
// channel key within a node. Combined with a [node.Key] to form the global channel Key.
type LocalKey types.Uint20

// Key is a unique identifier for a channel in the Synnax database. Composed of a
// cluster node key (first 12 bits) and a local key (last 20 bits), enabling distributed
// assignment while maintaining global uniqueness.
type Key uint32

// NewKey generates a new Key from the provided components.
func NewKey(nodeKey node.Key, localKey LocalKey) Key {
	// Node key is the first 12 bits,
	k1 := uint32(nodeKey) << 20
	// Local key is the last 20 bits
	k2 := uint32(localKey)
	return Key(k1 | k2)
}

// Lease implements the proxy.Entry interface, which routes Channel operations to the
// correct node in the cluster.
func (k Key) Lease() node.Key { return node.Key(k >> 20) }

// LocalKey returns the local key for the Key.
func (k Key) LocalKey() LocalKey { return LocalKey(k & 0xFFFFF) }

// Free returns true when the channel is not leased to any node i.e. it is a non-leased
// virtual channel.
func (k Key) Free() bool { return k.Lease() == node.KeyFree }

// StorageKey returns the storage layer representation of the channel key.
func (k Key) StorageKey() ts.ChannelKey { return ts.ChannelKey(k) }

// String implements fmt.Stringer.
func (k Key) String() string { return strconv.FormatUint(uint64(k), 10) }

// Keys extends []Key with a few convenience methods.
type Keys []Key

// KeysFromChannels returns a slice of Keys from a slice of Channel(s).
func KeysFromChannels(channels []Channel) Keys {
	return lo.Map(channels, func(ch Channel, _ int) Key { return ch.Key() })
}

// KeysFromUint32 returns a slice of Keys from a slice of uint32. NOTE: This does not
// copy the slice, it just reinterprets the memory.
func KeysFromUint32(keys []uint32) Keys {
	return unsafe.ReinterpretSlice[uint32, Key](keys)
}

// Uint32 converts the Keys to a slice of uint32. NOTE: This does not copy the slice, it
// just reinterprets the memory.
func (k Keys) Uint32() []uint32 { return unsafe.ReinterpretSlice[Key, uint32](k) }

// Storage returns the storage layer representation of the channel keys. NOTE: This does
// not copy the slice, it just reinterprets the memory.
func (k Keys) Storage() []ts.ChannelKey { return k.Uint32() }

// UniqueLeaseholders returns a slice of all UNIQUE leaseholders for the given Keys.
func (k Keys) UniqueLeaseholders() []node.Key {
	return lo.UniqMap(k, func(key Key, _ int) node.Key { return key.Lease() })
}

// Contains returns true if the slice contains the given key, false otherwise.
func (k Keys) Contains(key Key) bool { return slices.Contains(k, key) }

// Unique removes duplicate keys from the slice and returns the result.
func (k Keys) Unique() Keys { return lo.Uniq(k) }

// Channel is the minimal, distribution-layer representation of a channel. It carries
// only the storage and routing metadata the distribution layer needs to allocate local
// keys, create storage, and route frames across the cluster.
type Channel struct {
	// Name is the human-readable channel name.
	Name string
	// Leaseholder is the cluster node that holds the lease for this channel and is
	// authorized to accept writes.
	Leaseholder node.Key
	// DataType is the data type of samples stored in this channel.
	DataType telem.DataType
	// IsIndex is true if this channel is an index channel.
	IsIndex bool
	// LocalKey is the locally-unique portion of this channel's key.
	LocalKey LocalKey
	// LocalIndex is the local key of the channel used to index this channel's values.
	LocalIndex LocalKey
	// Virtual is true if this channel does not persist data and is used only for
	// streaming.
	Virtual bool
	// Concurrency sets the policy for concurrent writes to the channel's data. Only
	// virtual channels can have a policy of shared concurrency.
	Concurrency control.Concurrency
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

// Lease implements the proxy.UnaryServer interface.
func (c Channel) Lease() node.Key { return c.Leaseholder }

// Free returns true if the channel is not leased to any node i.e. it is a non-leased
// virtual channel.
func (c Channel) Free() bool { return c.Leaseholder == node.KeyFree }

// String implements stringer, returning a nicely formatted string representation of the
// Channel.
func (c Channel) String() string {
	if c.Name != "" {
		return fmt.Sprintf("[%s]<%d>", c.Name, c.Key())
	}
	return fmt.Sprintf("<%d>", c.Key())
}

// Storage returns the storage layer representation of the channel for creation in the
// storage ts.DB. Virtual channels are never persisted by the storage layer: they are
// registered at create time and re-registered from the channel table on boot.
func (c Channel) Storage() ts.Channel {
	return ts.Channel{
		Key:         c.Key().StorageKey(),
		Name:        c.Name,
		IsIndex:     c.IsIndex,
		DataType:    c.DataType,
		Index:       c.Index().StorageKey(),
		Virtual:     c.Virtual,
		Concurrency: c.Concurrency,
	}
}
