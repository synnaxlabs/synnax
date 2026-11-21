// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// referenceChangeName is the name of the channel that announces the records whose
// holders changed.
const referenceChangeName = "sy_reference_change"

// ReferenceChange is a target whose holders changed in one commit, with the holders
// whose links to it changed.
type ReferenceChange struct {
	Target  resource.ID
	Holders []resource.ID
}

// ReferenceChanges is the reference change channel and the changes behind its samples.
type ReferenceChanges struct {
	// Key is the key of the channel. It is zero before PublishReferenceChanges runs.
	Key  channel.Key
	feed *referenceFeed
}

// OnChange calls handler with the changes of each commit that adds or removes a link,
// once the commit is visible. The handler runs on the committing goroutine, so it must
// not block, and it must not change the changes. Call it only when Key is not zero.
func (r ReferenceChanges) OnChange(
	handler func(context.Context, []ReferenceChange),
) observe.Disconnect {
	r.feed.watchers.Add(1)
	disconnect := r.feed.changes.OnChange(handler)
	return sync.OnceFunc(func() {
		disconnect()
		r.feed.watchers.Add(-1)
	})
}

// ReferenceChanges returns the reference change channel and its changes.
func (p *Provider) ReferenceChanges() ReferenceChanges { return p.references }

// PublishReferenceChanges declares the channel sy_reference_change. For each commit
// that adds or removes a link, Start writes to it the resource ID of each target of
// those links, once, and ReferenceChanges delivers each target with the holders whose
// links to it changed: holderOf returns the resource of a holder. The holders never
// reach the channel, so no reader of it learns a holder that its subject cannot
// retrieve. Call it once every table with references is open: it fails when a table
// that a reference points at names no resource type.
func (p *Provider) PublishReferenceChanges(
	ctx context.Context,
	holderOf func(gorp.Target) resource.ID,
) error {
	for _, table := range p.cfg.DB.ReferencedTables() {
		if !resource.Type(table).IsValid() {
			return errors.Newf(
				"[signals] - referenced table %q names no resource type",
				table,
			)
		}
	}
	f := newReferenceFeed(holderOf)
	if err := p.add(ctx, f, &f.ch); err != nil {
		return err
	}
	// The feed goes first, so it starts each frame that it writes to.
	p.feeds = slices.Insert(p.feeds[:len(p.feeds)-1], 0, feed(f))
	p.references = ReferenceChanges{Key: f.ch.Key(), feed: f}
	return nil
}

// referenceFeed writes the target of each link that a commit adds or removes, and
// delivers each target with the holders whose links to it changed.
type referenceFeed struct {
	ch       channel.Channel
	holderOf func(gorp.Target) resource.ID
	changes  observe.Observer[[]ReferenceChange]
	// watchers is the number of handlers of changes. With none, the feed tracks no
	// holders.
	watchers atomic.Int32
	// targets, index, links, and seen hold the work of one commit. Commits append one
	// at a time, so the feed reuses them.
	targets []gorp.Target
	index   map[gorp.Target]int
	links   []targetHolder
	seen    set.Set[targetHolder]
	// peak is the most targets or links of a commit since the feed last freed them.
	peak int
}

// referenceRetained is the most targets and links that a referenceFeed keeps room for
// after a commit that uses less than a quarter of its peak.
const referenceRetained = 1024

func newReferenceFeed(holderOf func(gorp.Target) resource.ID) *referenceFeed {
	return &referenceFeed{
		ch:       channel.Channel{Name: referenceChangeName, DataType: telem.StringT},
		holderOf: holderOf,
		changes:  observe.New[[]ReferenceChange](),
		index:    make(map[gorp.Target]int),
		seen:     make(set.Set[targetHolder]),
	}
}

// targetHolder is a holder whose link to the target at an index of targets changed.
type targetHolder struct {
	target int
	holder resource.ID
}

func (f *referenceFeed) append(c gorp.Commit, fr framer.Frame) framer.Frame {
	defer f.reset()
	watched := f.watchers.Load() > 0
	for _, ch := range c.Changes {
		if len(ch.Added) == 0 && len(ch.Removed) == 0 {
			continue
		}
		var holder resource.ID
		if watched {
			holder = f.holderOf(ch.Record)
		}
		for _, links := range [2][]gorp.Link{ch.Added, ch.Removed} {
			for _, l := range links {
				i := f.addTarget(l.Target)
				if watched {
					f.addHolder(i, holder)
				}
			}
		}
	}
	if len(f.targets) == 0 {
		return fr
	}
	size := 0
	for _, target := range f.targets {
		size += 4 + len(target.Table) + 1 + len(target.Key)
	}
	data := make([]byte, 0, size)
	for _, target := range f.targets {
		data = appendSample(data, resourceOf(target))
	}
	if watched {
		f.notify()
	}
	series := telem.Series{DataType: telem.StringT, Data: data}
	if !fr.Empty() {
		return fr.Append(f.ch.Key(), series)
	}
	e := &referenceFrame{}
	e.keys[0], e.series[0] = f.ch.Key(), series
	return frame.NewMulti(e.keys[:1], e.series[:1])
}

// referenceFrame holds the entries of a frame that the feed starts, with room for the
// entry of one more feed, so the frame of a commit takes one allocation.
type referenceFrame struct {
	keys   [2]channel.Key
	series [2]telem.Series
}

// notify delivers each target of the commit with the holders that changed it.
func (f *referenceFeed) notify() {
	slices.SortStableFunc(f.links, func(a, b targetHolder) int {
		return cmp.Compare(a.target, b.target)
	})
	changes := make([]ReferenceChange, len(f.targets))
	holders := make([]resource.ID, len(f.links))
	for i, j := 0, 0; i < len(f.targets); i++ {
		start := j
		for ; j < len(f.links) && f.links[j].target == i; j++ {
			holders[j] = f.links[j].holder
		}
		changes[i] = ReferenceChange{
			Target:  resourceOf(f.targets[i]),
			Holders: holders[start:j:j],
		}
	}
	// The DB delivers a commit to its readers once the commit is visible.
	f.changes.Notify(context.Background(), changes)
}

// reset empties the work of a commit. It frees the work when the peak outgrew
// referenceRetained and the commit used less than a quarter of it, so commits of one
// size keep their room.
func (f *referenceFeed) reset() {
	used := max(len(f.targets), len(f.links))
	f.peak = max(f.peak, used)
	if f.peak > referenceRetained && used < f.peak/4 {
		f.targets, f.links = nil, nil
		f.index = make(map[gorp.Target]int)
		f.seen = make(set.Set[targetHolder])
		f.peak = 0
		return
	}
	clear(f.targets)
	clear(f.links)
	f.targets, f.links = f.targets[:0], f.links[:0]
	clear(f.index)
	clear(f.seen)
}

// addTarget records that a link to target changed, and returns the index of target.
func (f *referenceFeed) addTarget(target gorp.Target) int {
	i, ok := f.index[target]
	if !ok {
		i = len(f.targets)
		f.index[target] = i
		f.targets = append(f.targets, target)
	}
	return i
}

// addHolder records that the link of holder to the target at index i changed.
func (f *referenceFeed) addHolder(i int, holder resource.ID) {
	if th := (targetHolder{target: i, holder: holder}); !f.seen.Contains(th) {
		f.seen.Add(th)
		f.links = append(f.links, th)
	}
}

// records returns no Record: the API streams the channel from ReferenceChanges.
func (f *referenceFeed) records() map[channel.Key]Record { return nil }

// resourceOf returns the resource of target. The name of a referenced table is its
// resource type, which PublishReferenceChanges checks.
func resourceOf(target gorp.Target) resource.ID {
	return resource.ID{Type: resource.Type(target.Table), Key: target.Key}
}

// appendSample appends the text of id to data as one variable-length sample.
func appendSample(data []byte, id resource.ID) []byte {
	n := len(id.Type) + 1 + len(id.Key)
	data = telem.ByteOrder.AppendUint32(data, uint32(n))
	data = append(data, id.Type...)
	data = append(data, ':')
	return append(data, id.Key...)
}
