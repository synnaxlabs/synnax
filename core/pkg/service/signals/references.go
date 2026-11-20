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
	"bytes"
	"cmp"
	"context"
	"slices"
	"strconv"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// referenceChangeName is the name of the channel that announces the records whose
// holders changed.
const referenceChangeName = "sy_reference_change"

// PublishReferenceChanges declares the channel sy_reference_change. For each commit
// that adds or removes a link, Start writes to it the resource ID of each target of
// those links, once. A streamer sends a subject a target's sample only when the subject
// can retrieve the target and at least one holder whose link to it changed: holderOf
// returns the resource of a holder. A holder that names no resource type, such as a
// session, is one that no subject can retrieve. Call it once every table with
// references is open: it fails when a table that a reference points at names no
// resource type.
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
		index:    make(map[gorp.Target]int),
		seen:     make(set.Set[targetHolder]),
	}
	return p.add(ctx, f, &f.ch)
}

// referenceFeed writes the targets of the links that each commit adds or removes. Each
// sample holds the target's resource ID and then the ID of each holder whose link to it
// changed, each as its length in decimal, a colon, and its text, so that the sample
// stays valid UTF-8. The API streamer checks the holders and sends the target alone.
type referenceFeed struct {
	ch       channel.Channel
	holderOf func(gorp.Target) resource.ID
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
	var data []byte
	for i, j := 0, 0; i < len(f.targets); i++ {
		start := len(data)
		data = append(data, 0, 0, 0, 0)
		data = appendID(data, resourceOf(f.targets[i]))
		for ; j < len(f.links) && f.links[j].target == i; j++ {
			data = appendID(data, f.links[j].holder)
		}
		telem.ByteOrder.PutUint32(data[start:], uint32(len(data)-start-4))
	}
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

func (f *referenceFeed) records() map[channel.Key]Record {
	return map[channel.Key]Record{
		f.ch.Key(): {Of: targetOfSample, Changed: holdersOfSample},
	}
}

// resourceOf returns the resource of target. The name of a referenced table is its
// resource type, which PublishReferenceChanges checks.
func resourceOf(target gorp.Target) resource.ID {
	return resource.ID{Type: resource.Type(target.Table), Key: target.Key}
}

func appendID(sample []byte, id resource.ID) []byte {
	sample = strconv.AppendInt(sample, int64(len(id.Type)+1+len(id.Key)), 10)
	sample = append(sample, ':')
	sample = append(sample, id.Type...)
	sample = append(sample, ':')
	return append(sample, id.Key...)
}

var errMalformedSample = errors.New("[signals] - malformed reference change sample")

// nextID returns the text of the first ID in sample and the rest of sample.
func nextID(sample []byte) (id, rest []byte, err error) {
	colon := bytes.IndexByte(sample, ':')
	if colon <= 0 {
		return nil, nil, errMalformedSample
	}
	n, err := strconv.Atoi(string(sample[:colon]))
	if err != nil || n < 0 || n > len(sample)-colon-1 {
		return nil, nil, errMalformedSample
	}
	return sample[colon+1 : colon+1+n], sample[colon+1+n:], nil
}

func targetOfSample(sample []byte) (resource.ID, error) {
	target, _, err := nextID(sample)
	if err != nil {
		return resource.ID{}, err
	}
	return resource.ParseID(string(target))
}

func holdersOfSample(sample []byte) (resource.ID, []resource.ID, []byte, error) {
	text, rest, err := nextID(sample)
	if err != nil {
		return resource.ID{}, nil, nil, err
	}
	target, err := resource.ParseID(string(text))
	if err != nil {
		return resource.ID{}, nil, nil, err
	}
	var holders []resource.ID
	for len(rest) > 0 {
		var id []byte
		if id, rest, err = nextID(rest); err != nil {
			return resource.ID{}, nil, nil, err
		}
		holder, err := resource.ParseID(string(id))
		if err != nil {
			return resource.ID{}, nil, nil, err
		}
		holders = append(holders, holder)
	}
	return target, holders, text, nil
}
