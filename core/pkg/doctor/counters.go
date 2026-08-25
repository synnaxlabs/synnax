// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"github.com/synnaxlabs/aspen"
)

// checkCounters reports key counters that would re-issue a key already in use. Each
// counter holds the highest key its node has issued, so a counter below the highest
// stored key hands the next caller a duplicate.
func checkCounters(s *state, res walkResult, host aspen.NodeKey) {
	var leased, free, racks int64
	for key, c := range s.channels {
		local := int64(c.LocalKey)
		if key.Free() {
			free = max(free, local)
		} else if c.Leaseholder == host {
			leased = max(leased, local)
		}
	}
	for key := range s.racks {
		if key.Node() == host {
			racks = max(racks, int64(key.LocalKey()))
		}
	}
	prefix := host.String()
	checkCounter(s, res, prefix+channelCounterSuffix, leased, "leased channel keys")
	checkCounter(s, res, prefix+rackCounterSuffix, racks, "rack keys")
	if host == aspen.NodeKeyBootstrapper {
		checkCounter(s, res, prefix+freeChannelCounterSuffix, free, "free channel keys")
	}
}

// checkCounter reports one counter that is missing or below the highest key it guards.
func checkCounter(s *state, res walkResult, key string, highest int64, guards string) {
	if highest == 0 {
		return
	}
	value, found := res.counters[key]
	if !found {
		s.note(CheckCounter, "counter for "+guards+" is missing", key)
		return
	}
	if value < highest {
		s.note(CheckCounter, "counter for "+guards+" is behind", key)
	}
}
