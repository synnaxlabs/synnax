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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
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
	Key channel.Key
	// Observable delivers the changes of each commit that adds or removes a link, once
	// the commit is visible. A handler runs on the committing goroutine, so it must not
	// block.
	observe.Observable[[]ReferenceChange]
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
	f := &referenceFeed{
		ch:       channel.Channel{Name: referenceChangeName, DataType: telem.StringT},
		holderOf: holderOf,
		changes:  observe.New[[]ReferenceChange](),
		index:    make(map[gorp.Target]int),
		seen:     make(set.Set[targetHolder]),
	}
	if err := p.add(ctx, f, &f.ch); err != nil {
		return err
	}
	p.references = ReferenceChanges{Key: f.ch.Key(), Observable: f.changes}
	return nil
}

// referenceFeed writes the target of each link that a commit adds or removes, and
// delivers each target with the holders whose links to it changed.
type referenceFeed struct {
	ch       channel.Channel
	holderOf func(gorp.Target) resource.ID
	changes  observe.Observer[[]ReferenceChange]
	// targets, index, links, and seen hold the work of one commit. Commits append one
	// at a time, so the feed reuses them.
	targets []gorp.Target
	index   map[gorp.Target]int
	links   []targetHolder
	seen    set.Set[targetHolder]
}

// targetHolder is a holder whose link to the target at an index of targets changed.
type targetHolder struct {
	target int
	holder resource.ID
}

func (f *referenceFeed) append(c gorp.Commit, fr framer.Frame) framer.Frame {
	f.targets, f.links = f.targets[:0], f.links[:0]
	clear(f.index)
	clear(f.seen)
	for _, ch := range c.Changes {
		if len(ch.Added) == 0 && len(ch.Removed) == 0 {
			continue
		}
		holder := f.holderOf(ch.Record)
		for _, links := range [2][]gorp.Link{ch.Added, ch.Removed} {
			for _, l := range links {
				f.add(l.Target, holder)
			}
		}
	}
	if len(f.targets) == 0 {
		return fr
	}
	slices.SortStableFunc(f.links, func(a, b targetHolder) int {
		return cmp.Compare(a.target, b.target)
	})
	changes := make([]ReferenceChange, len(f.targets))
	var data []byte
	for i, j := 0, 0; i < len(f.targets); i++ {
		target := resourceOf(f.targets[i])
		changes[i].Target = target
		for ; j < len(f.links) && f.links[j].target == i; j++ {
			changes[i].Holders = append(changes[i].Holders, f.links[j].holder)
		}
		data = appendSample(data, target)
	}
	// The DB delivers a commit to its readers once the commit is visible.
	f.changes.Notify(context.Background(), changes)
	return fr.Append(f.ch.Key(), telem.Series{DataType: telem.StringT, Data: data})
}

// add records that the link of holder to target changed.
func (f *referenceFeed) add(target gorp.Target, holder resource.ID) {
	i, ok := f.index[target]
	if !ok {
		i = len(f.targets)
		f.index[target] = i
		f.targets = append(f.targets, target)
	}
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
