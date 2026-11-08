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
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
)

// MatchVirtual returns a filter for channels that are virtual if virtual is true, or
// are not virtual if virtual is false. Calculated channels are excluded from the
// virtual bucket even though they are stored with Virtual=true.
func MatchVirtual(virtual bool) Filter {
	return Match(func(_ gorp.Context, _ Retrieve, ch *Channel) (bool, error) {
		isVirtual := ch.Virtual && !ch.IsCalculated()
		return isVirtual == virtual, nil
	})
}

// MatchCalculated returns a filter for channels that have a non-empty Expression field.
func MatchCalculated() Filter {
	return Match(func(_ gorp.Context, _ Retrieve, ch *Channel) (bool, error) {
		return ch.IsCalculated(), nil
	})
}

// MatchIndexes returns a filter for channels whose index is any of the given keys.
func MatchIndexes(indexes ...Key) Filter {
	localKeys := lo.Map(indexes, func(k Key, _ int) LocalKey { return k.LocalKey() })
	return func(r Retrieve) gorp.Filter[Key, Channel] {
		return gorp.And(
			r.indexes.localIndex.Filter(localKeys...),
			gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
				return slices.Contains(indexes, ch.Index()), nil
			}),
		)
	}
}
