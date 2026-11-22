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
	"sync/atomic"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Retrieve is a builder for querying channels.
type Retrieve struct {
	gorp    gorp.Retrieve[Key, Channel]
	indexes indexes
	// builtIn holds the filters of MatchInternal.
	builtIn *builtInFilters
	// folderAllowed reports whether a path can pass the folder with an ID, or is nil
	// when a path can pass any folder.
	folderAllowed func(resource.ID) bool
}

// AllowFolders makes a path that MatchPaths matches pass only folders that allowed
// accepts. It must come before Where, since a filter reads it when Where adds it.
func (r Retrieve) AllowFolders(allowed func(resource.ID) bool) Retrieve {
	r.folderAllowed = allowed
	return r
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

// MatchVirtual returns a filter for channels that are virtual if virtual is true, or
// are not virtual if virtual is false.
func MatchVirtual(virtual bool) Filter {
	return Match(func(_ gorp.Context, _ Retrieve, ch *Channel) (bool, error) {
		return ch.Virtual == virtual, nil
	})
}
