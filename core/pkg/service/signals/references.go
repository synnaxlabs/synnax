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
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// ReferenceChangeName is the name of the channel that announces the records whose
// holders changed.
const ReferenceChangeName = "sy_reference_change"

// PublishReferenceChanges declares the channel named ReferenceChangeName. For each
// commit that adds or removes a link, Start writes to it the resource ID of each target
// of those links, once. A streamer sends a sample only to a subject that can retrieve
// its target. Call it once every table with references is open: it fails when one of
// the tables that they point at names no resource type.
func (p *Provider) PublishReferenceChanges(ctx context.Context) error {
	for _, table := range p.cfg.DB.ReferencedTables() {
		if !resource.Type(table).IsValid() {
			return errors.Newf(
				"[signals] - referenced table %q names no resource type",
				table,
			)
		}
	}
	f := &referenceFeed{
		ch:      channel.Channel{Name: ReferenceChangeName, DataType: telem.StringT},
		targets: make(set.Set[gorp.Target]),
	}
	return p.add(ctx, f, &f.ch)
}

// referenceFeed writes the targets of the links that each commit adds or removes.
type referenceFeed struct {
	ch channel.Channel
	// targets holds the targets of one commit. Commits append one at a time.
	targets set.Set[gorp.Target]
}

func (f *referenceFeed) append(c gorp.Commit, fr framer.Frame) framer.Frame {
	clear(f.targets)
	var ids []string
	for _, ch := range c.Changes {
		for _, links := range [2][]gorp.Link{ch.Added, ch.Removed} {
			for _, l := range links {
				if f.targets.Contains(l.Target) {
					continue
				}
				f.targets.Add(l.Target)
				ids = append(ids, resourceOf(l.Target).String())
			}
		}
	}
	if len(ids) == 0 {
		return fr
	}
	return fr.Append(f.ch.Key(), telem.NewSeriesV(ids...))
}

func (f *referenceFeed) records() map[channel.Key]Record {
	return map[channel.Key]Record{f.ch.Key(): {Of: parseResourceSample}}
}

func parseResourceSample(sample []byte) (resource.ID, error) {
	return resource.ParseID(string(sample))
}

// resourceOf returns the resource of target. The name of a referenced table is its
// resource type, which PublishReferenceChanges checks.
func resourceOf(target gorp.Target) resource.ID {
	return resource.ID{Type: resource.Type(target.Table), Key: target.Key}
}
