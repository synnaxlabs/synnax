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
	"context"
	"slices"
	"strconv"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// ReferenceChangeName is the name of the channel that announces the records whose
// holders changed.
const ReferenceChangeName = "sy_reference_change"

// PublishReferenceChanges declares the channel named ReferenceChangeName. For each
// commit that adds or removes a link, Start writes to it the resource ID of each target
// of those links, once. A streamer sends a subject a target's sample only when the
// subject can retrieve the target and at least one holder whose link to it changed:
// holderOf returns the resource of a holder. A holder that names no resource type, such
// as a session, is one that no subject can retrieve. Call it once every table with
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
		ch:       channel.Channel{Name: ReferenceChangeName, DataType: telem.StringT},
		holderOf: holderOf,
		holders:  make(map[gorp.Target][]string),
	}
	return p.add(ctx, f, &f.ch)
}

// referenceFeed writes the targets of the links that each commit adds or removes. Each
// sample holds the target's resource ID and then the ID of each holder whose link to it
// changed, each as its length in decimal, a colon, and its text. The API streamer
// checks the holders and sends the target alone.
type referenceFeed struct {
	ch       channel.Channel
	holderOf func(gorp.Target) resource.ID
	// holders holds the changed holders of each target of one commit. Commits append
	// one at a time.
	holders map[gorp.Target][]string
}

func (f *referenceFeed) append(c gorp.Commit, fr framer.Frame) framer.Frame {
	clear(f.holders)
	var targets []gorp.Target
	for _, ch := range c.Changes {
		holder := f.holderOf(ch.Record).String()
		for _, links := range [2][]gorp.Link{ch.Added, ch.Removed} {
			for _, l := range links {
				held, seen := f.holders[l.Target]
				if !seen {
					targets = append(targets, l.Target)
				}
				if !slices.Contains(held, holder) {
					f.holders[l.Target] = append(held, holder)
				}
			}
		}
	}
	if len(targets) == 0 {
		return fr
	}
	samples := make([]string, 0, len(targets))
	for _, t := range targets {
		sample := appendID(nil, resourceOf(t).String())
		for _, h := range f.holders[t] {
			sample = appendID(sample, h)
		}
		samples = append(samples, string(sample))
	}
	return fr.Append(f.ch.Key(), telem.NewSeriesV(samples...))
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

func appendID(sample []byte, id string) []byte {
	sample = strconv.AppendInt(sample, int64(len(id)), 10)
	sample = append(sample, ':')
	return append(sample, id...)
}

var errMalformedSample = errors.New("[signals] - malformed reference change sample")

func idsOfSample(sample []byte) ([]string, error) {
	var ids []string
	for len(sample) > 0 {
		colon := bytes.IndexByte(sample, ':')
		if colon <= 0 {
			return nil, errMalformedSample
		}
		n, err := strconv.Atoi(string(sample[:colon]))
		if err != nil || n < 0 || n > len(sample)-colon-1 {
			return nil, errMalformedSample
		}
		ids = append(ids, string(sample[colon+1:colon+1+n]))
		sample = sample[colon+1+n:]
	}
	if len(ids) == 0 {
		return nil, errMalformedSample
	}
	return ids, nil
}

func targetOfSample(sample []byte) (resource.ID, error) {
	ids, err := idsOfSample(sample)
	if err != nil {
		return resource.ID{}, err
	}
	return resource.ParseID(ids[0])
}

func holdersOfSample(sample []byte) ([]resource.ID, []byte, error) {
	ids, err := idsOfSample(sample)
	if err != nil {
		return nil, nil, err
	}
	holders := make([]resource.ID, len(ids)-1)
	for i, id := range ids[1:] {
		if holders[i], err = resource.ParseID(id); err != nil {
			return nil, nil, err
		}
	}
	return holders, []byte(ids[0]), nil
}
