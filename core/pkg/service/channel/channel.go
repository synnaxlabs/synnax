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
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

// These type aliases re-export the distribution-layer key types so service-layer
// callers can refer to them through this package without importing the distribution
// layer directly.
type (
	// Key is the cluster-unique identifier for a channel. It packs a leaseholder node
	// key (first 12 bits) and a node-local LocalKey (last 20 bits) into a single
	// uint32. Re-exported from [channel.Key].
	Key = channel.Key
	// Keys is a slice of Key with convenience methods for deduplication, grouping by
	// leaseholder, and conversion to storage keys. Re-exported from [channel.Keys].
	Keys = channel.Keys
	// LocalKey is the 20-bit, node-local portion of a channel Key. Re-exported from
	// [channel.LocalKey].
	LocalKey = channel.LocalKey
	// Name is the human-readable identifier for a channel. It is an alias for string,
	// provided so callers can refer to channel names through a domain-specific type.
	Name = string
)

var (
	// NewKey composes a Key from a leaseholder node key and a node-local LocalKey.
	// Re-exported from [channel.NewKey].
	NewKey = channel.NewKey
	// KeysFromUint32 reinterprets a []uint32 as Keys without copying the underlying
	// memory. Re-exported from [channel.KeysFromUint32].
	KeysFromUint32 = channel.KeysFromUint32
)

// ParseKey attempts to parse the string representation of a Key into a Key.
func ParseKey(s string) (Key, error) {
	k, err := strconv.Atoi(s)
	if err != nil {
		return Key(0), errors.Wrapf(
			validate.ErrValidation, "%s is not a valid channel key", s,
		)
	}
	return Key(k), nil
}

// KeysFromChannels returns a slice of Keys from a slice of Channel(s).
func KeysFromChannels(channels []Channel) Keys {
	return lo.Map(channels, func(ch Channel, _ int) Key { return ch.Key() })
}

// Names returns the names of the channels.
func Names(channels []Channel) []string {
	return lo.Map(channels, func(channel Channel, _ int) string { return channel.Name })
}

var _ gorp.Entry[Key] = Channel{}

// IsCalculated returns true if the channel is a calculated channel, false otherwise.
func (c Channel) IsCalculated() bool { return c.Expression != "" }

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

// SetOptions implements the gorp.Entry interface. Returns a set of options that tell an
// aspen.DB to properly lease the Channel to the node it will be recording data from.
func (c Channel) SetOptions() []any {
	if c.Free() {
		return []any{node.KeyBootstrapper}
	}
	return []any{c.Leaseholder}
}

// Free returns true if the channel is not leased to a particular node i.e. it is a
// non-leased virtual channel.
func (c Channel) Free() bool { return c.Leaseholder == node.KeyFree }

// Distribution returns the minimal distribution-layer representation of the channel,
// carrying only the storage and routing metadata the distribution layer needs.
func (c Channel) Distribution() channel.Channel {
	return channel.Channel{
		Name:        c.Name,
		Leaseholder: c.Leaseholder,
		DataType:    c.DataType,
		IsIndex:     c.IsIndex,
		LocalKey:    c.LocalKey,
		LocalIndex:  c.LocalIndex,
		Virtual:     c.Virtual,
		Concurrency: c.Concurrency,
	}
}

// Equals returns true if the two channels are meaningfully equal to each other. If the
// exclude parameter is provided, the function will ignore the fields specified in the
// exclude parameter.
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
		if !comp.equal && !slices.Contains(exclude, comp.field) {
			return false
		}
	}
	if !slices.Contains(exclude, "Operations") {
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

// UnmarshalJSON implements json.Unmarshaler, supporting both legacy "node_id" and new
// "leaseholder" field names for backward compatibility.
func (c *Channel) UnmarshalJSON(data []byte) error {
	type alias Channel
	if err := json.Unmarshal(data, (*alias)(c)); err != nil {
		return errors.Wrap(err, "failed to decode channel from JSON")
	}
	if c.Leaseholder == 0 {
		var legacy struct {
			NodeID node.Key `json:"node_id"`
		}
		if err := json.Unmarshal(data, &legacy); err != nil {
			return errors.Wrap(err, "failed to decode legacy node_id from JSON")
		}
		c.Leaseholder = legacy.NodeID
	}
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase Go
// field names (e.g. "Type", "ResetChannel", "Duration") and new lowercase msgpack tag
// names for backward compatibility.
func (o *Operation) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Operation
	raw, err := dec.DecodeRaw()
	if err != nil {
		return errors.Wrap(err, "failed to read raw operation msgpack")
	}
	if err = msgpack.Unmarshal(raw, (*alias)(o)); err != nil {
		return errors.Wrap(err, "failed to decode operation from msgpack")
	}
	if len(o.Type) == 0 {
		var legacy struct {
			Type         OperationType
			ResetChannel Key
			Duration     telem.TimeSpan
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return errors.Wrap(err, "failed to decode legacy operation from msgpack")
		}
		o.Type = legacy.Type
		o.ResetChannel = legacy.ResetChannel
		o.Duration = legacy.Duration
	}
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy "node_id" and
// new "leaseholder" field names for backward compatibility.
func (c *Channel) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Channel
	raw, err := dec.DecodeRaw()
	if err != nil {
		return errors.Wrap(err, "failed to read raw channel msgpack")
	}
	if err = msgpack.Unmarshal(raw, (*alias)(c)); err != nil {
		return errors.Wrap(err, "failed to decode channel from msgpack")
	}
	if c.Leaseholder == 0 {
		var legacy struct {
			NodeID node.Key `msgpack:"node_id"`
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return errors.Wrap(err, "failed to decode legacy node_id from msgpack")
		}
		c.Leaseholder = legacy.NodeID
	}
	return nil
}

// NewRandomName generates a random channel name that should be unique. It is a test
// helper for generating distinct channel names; channel name uniqueness itself is
// enforced by the service layer during creation.
func NewRandomName() string {
	return fmt.Sprintf("test_ch_%09d", rand.Intn(999999999))
}
