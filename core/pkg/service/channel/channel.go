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
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// These type aliases re-export the key types so service-layer callers can refer to
// them through this package without importing the distribution layer directly.
type (
	// Key is the cluster-unique identifier for a channel. It packs a leaseholder node
	// key (first 12 bits) and a node-local LocalKey (last 20 bits) into a single
	// uint32. Re-exported from [channel.Key].
	Key = versions.Key
	// Keys is a slice of Key with convenience methods for deduplication, grouping by
	// leaseholder, and conversion to storage keys. Re-exported from [channel.Keys].
	Keys = versions.Keys
	// LocalKey is the 20-bit, node-local portion of a channel Key. Re-exported from
	// [channel.LocalKey].
	LocalKey = versions.LocalKey
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
