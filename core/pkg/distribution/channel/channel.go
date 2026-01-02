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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/unsafe"
	"github.com/synnaxlabs/x/validate"
)

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

func (c Channel) IsCalculated() bool {
	return c.Expression != ""
}

// Equals returns true if the two channels are meaningfully equal to each other.
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
