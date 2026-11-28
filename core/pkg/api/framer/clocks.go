// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// writerClocks returns the index channel of each group of the stored channels of keys,
// in key order, and the ones among them that keys do not name, which the writer adds.
// It leaves out an index that subject may not write.
func (s *Service) writerClocks(
	ctx context.Context,
	snap gorp.Snapshot,
	subject resource.ID,
	keys channel.Keys,
) (clocks, added channel.Keys, err error) {
	var channels []channel.Channel
	if err := s.channel.NewRetrieve().
		Entries(&channels).
		Where(channel.MatchKeys(keys...)).
		Exec(ctx, snap); err != nil {
		return nil, nil, err
	}
	allowed, err := s.access.NewEnforcer(snap).
		Allowed(ctx, subject, access.ActionCreate)
	if err != nil {
		return nil, nil, err
	}
	byKey := make(map[channel.Key]channel.Channel, len(channels))
	for _, ch := range channels {
		byKey[ch.Key()] = ch
	}
	for _, key := range keys {
		ch, ok := byKey[key]
		index := ch.Index()
		if !ok || ch.Virtual || index == 0 || clocks.Contains(index) {
			continue
		}
		if !keys.Contains(index) {
			if !allowed(framer.ResourceIDs(channel.Keys{index})[0]) {
				continue
			}
			added = append(added, index)
		}
		clocks = append(clocks, index)
	}
	return clocks, added, nil
}

// dropStored removes from clocks and added each index of added that holds data at or
// after start, or after now when start is zero. A writer cannot open such an index, so
// the data of its group takes its time from the stamps that the index stores.
func (s *Service) dropStored(
	ctx context.Context,
	clocks, added channel.Keys,
	start telem.TimeStamp,
) (channel.Keys, channel.Keys, error) {
	if start == 0 {
		start = telem.Now()
	}
	kept := make(channel.Keys, 0, len(added))
	for _, index := range added {
		stored, err := s.holdsData(ctx, index, start)
		if err != nil {
			return nil, nil, err
		}
		if !stored {
			kept = append(kept, index)
			continue
		}
		clocks = slices.DeleteFunc(
			clocks,
			func(k channel.Key) bool { return k == index },
		)
	}
	return clocks, kept, nil
}

// holdsData reports whether index holds a sample at or after start.
func (s *Service) holdsData(
	ctx context.Context,
	index channel.Key,
	start telem.TimeStamp,
) (found bool, err error) {
	it, err := s.internal.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   channel.Keys{index},
		Bounds: telem.TimeRange{Start: start, End: telem.TimeStampMax},
	})
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Combine(err, it.Close()) }()
	found = it.SeekFirst() && it.Next(framer.IteratorAutoSpan)
	return found, it.Error()
}
