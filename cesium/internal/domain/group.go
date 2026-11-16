// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"sync/atomic"

	"github.com/synnaxlabs/x/telem"
)

// Group makes commits of several Writers visible to readers in one step. A commit in a
// Group takes effect at once for writers, conflict checks, and persists, but readers
// see each domain as it was before the commit until Publish. A Group is not safe for
// concurrent use.
type Group struct {
	// published counts the calls to Publish. Readers load it.
	published atomic.Uint64
	// changes holds the history changes of the commits since the last Publish.
	changes []func()
}

// Publish makes every commit in the Group since the last Publish visible to readers,
// then reports their history changes.
func (g *Group) Publish() {
	g.published.Add(1)
	for i, change := range g.changes {
		change()
		g.changes[i] = nil
	}
	g.changes = g.changes[:0]
}

// staged is what readers see of a domain that a commit in a Group changed, until the
// Group publishes.
type staged struct {
	group *Group
	// publish is the count of publishes of group that makes the commit visible.
	publish uint64
	// inserted is true when the commit added the domain, so readers see no domain.
	// Otherwise readers see the domain with end and size.
	inserted bool
	end      telem.TimeStamp
	size     uint32
}

func (s staged) visible() bool { return s.group.published.Load() >= s.publish }
