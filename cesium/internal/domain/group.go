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
	"context"
	"sync/atomic"

	"github.com/synnaxlabs/x/telem"
)

// Group makes commits of several Writers visible to readers in one step. A commit in a
// Group takes effect at once for writers, conflict checks, and persists, but readers
// see each domain as it was before the commit until Publish. Only one goroutine at a
// time may commit in a Group or publish it.
type Group struct {
	// published counts the calls to Publish. Readers load it.
	published atomic.Uint64
	// commits holds the domains of the commits since the last Publish.
	commits []stagedDomain
}

// stagedDomain is a domain that a commit in a Group changed.
type stagedDomain struct {
	idx   *index
	start telem.TimeStamp
}

// Publish makes every commit in the Group since the last Publish visible to readers,
// then reports their history changes.
func (g *Group) Publish() {
	g.published.Add(1)
	for i, d := range g.commits {
		d.idx.revealPublished(d.start, g)
		g.commits[i] = stagedDomain{}
	}
	g.commits = g.commits[:0]
}

// historyChange is a history change that waits until readers see its commit. One with
// a nil ctx holds no change.
type historyChange struct {
	// ctx is the context of the last commit of the change.
	ctx context.Context
	tr  telem.TimeRange
}

// then returns the change that covers c and next, which follows c.
func (c historyChange) then(next historyChange) historyChange {
	if c.ctx == nil {
		return next
	}
	if next.ctx == nil {
		return c
	}
	return historyChange{
		ctx: next.ctx,
		tr:  min(c.tr.Start, next.tr.Start).Range(max(c.tr.End, next.tr.End)),
	}
}

// staged is what readers see of a domain that a commit in a Group changed, until the
// Group publishes.
type staged struct {
	// start is the start of the domain.
	start telem.TimeStamp
	group *Group
	// publish is the count of publishes of group that makes the commit visible.
	publish uint64
	// inserted is true when the commit added the domain, so readers see no domain.
	// Otherwise readers see the domain with end and size.
	inserted bool
	end      telem.TimeStamp
	size     uint32
	// change is the history change to report once readers see the commit.
	change historyChange
}

func (s staged) visible() bool { return s.group.published.Load() >= s.publish }
