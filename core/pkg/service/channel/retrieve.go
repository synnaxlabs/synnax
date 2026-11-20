// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import "github.com/synnaxlabs/gorp"

// Retrieve is a builder for querying channels.
type Retrieve struct {
	gorp    gorp.Retrieve[Key, Channel]
	indexes indexes
	// builtIn reports whether a key is a built-in channel.
	builtIn func(Key) bool
}

// The filters of MatchInternal, built once so that a request allocates none.
var (
	matchBuiltIn    = Match(isBuiltIn)
	matchNotBuiltIn = Match(isNotBuiltIn)
)

func isBuiltIn(_ gorp.Context, r Retrieve, ch *Channel) (bool, error) {
	return r.builtIn(ch.Key()), nil
}

func isNotBuiltIn(_ gorp.Context, r Retrieve, ch *Channel) (bool, error) {
	return !r.builtIn(ch.Key()), nil
}

// MatchInternal returns a filter for the built-in channels when internal is true, or
// for every other channel when it is false.
func MatchInternal(internal bool) Filter {
	if internal {
		return matchBuiltIn
	}
	return matchNotBuiltIn
}

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
