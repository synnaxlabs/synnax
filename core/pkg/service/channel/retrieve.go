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
	"context"
	"sync/atomic"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
)

// Retrieve is a builder for querying channels.
type Retrieve struct {
	gorp    gorp.Retrieve[Key, Channel]
	indexes indexes
	// builtIn holds the filters of MatchInternal.
	builtIn *builtInFilters
	// calculated holds the filters of MatchCalculated.
	calculated *calculatedFilters
}

// builtInFilters holds the filters of MatchInternal for one Service. The Service builds
// them once, so that a request allocates none.
type builtInFilters struct{ in, out gorp.Filter[Key, Channel] }

func newBuiltInFilters(builtIns *atomic.Pointer[builtInSet]) builtInFilters {
	return builtInFilters{
		in: gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
			return builtIns.Load().contains(ch.Key()), nil
		}),
		out: gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
			return !builtIns.Load().contains(ch.Key()), nil
		}),
	}
}

func matchBuiltIn(r Retrieve) gorp.Filter[Key, Channel] { return r.builtIn.in }

func matchNotBuiltIn(r Retrieve) gorp.Filter[Key, Channel] { return r.builtIn.out }

// MatchInternal returns a filter for the built-in channels when internal is true, or
// for every other channel when it is false.
func MatchInternal(internal bool) Filter {
	if internal {
		return matchBuiltIn
	}
	return matchNotBuiltIn
}

// calculatedFilters holds the filters of MatchCalculated for one Service. The Service
// builds them once, so that a request allocates none.
type calculatedFilters struct{ in, out gorp.Filter[Key, Channel] }

var errNoOutputs = errors.New(
	"the channel service has no source of calculation outputs",
)

func newCalculatedFilters(
	outputs func(context.Context, gorp.Snapshot) (Keys, error),
) calculatedFilters {
	in := gorp.MatchKeysFunc[Key, Channel](func(ctx gorp.Context) ([]Key, error) {
		if outputs == nil {
			return nil, errNoOutputs
		}
		return outputs(ctx, ctx.Snapshot)
	})
	return calculatedFilters{in: in, out: gorp.Not(in)}
}

func matchCalculated(r Retrieve) gorp.Filter[Key, Channel] { return r.calculated.in }

func matchNotCalculated(
	r Retrieve,
) gorp.Filter[Key, Channel] {
	return r.calculated.out
}

// MatchCalculated returns a filter for the channels that calculations write when
// calculated is true, or for every other channel when it is false. A query with the
// filter fails when the Service has no Outputs source.
func MatchCalculated(calculated bool) Filter {
	if calculated {
		return matchCalculated
	}
	return matchNotCalculated
}

// MatchVirtual returns a filter for channels that are virtual if virtual is true, or
// are not virtual if virtual is false.
func MatchVirtual(virtual bool) Filter {
	return Match(func(_ gorp.Context, _ Retrieve, ch *Channel) (bool, error) {
		return ch.Virtual == virtual, nil
	})
}
