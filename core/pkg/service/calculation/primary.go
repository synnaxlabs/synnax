// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// intervalWindow is the number of an input's newest intervals that set its rate, the
// same number that sets its hold limit in the join.
const intervalWindow = 16

// pickPrimary returns the read whose index has the shortest median interval over its
// newest stored stretch, or the earlier read on a tie. A virtual read, or one with
// fewer than 2 samples in that stretch, is no candidate. With no candidate it returns
// the first read, and with no reads nil.
func (w Writer) pickPrimary(
	ctx context.Context,
	reads []channel.Key,
) (*channel.Key, error) {
	if len(reads) == 0 {
		return nil, nil
	}
	var channels []channel.Channel
	if err := w.channels.NewRetrieve().
		Where(channel.MatchKeys(reads...)).
		Entries(&channels).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	indexOf := make(map[channel.Key]channel.Key, len(channels))
	for _, ch := range channels {
		if !ch.Virtual {
			indexOf[ch.Key()] = ch.Index()
		}
	}
	type rate struct {
		median telem.TimeSpan
		ok     bool
	}
	var (
		rates    = make(map[channel.Key]rate, len(indexOf))
		primary  = reads[0]
		shortest telem.TimeSpan
		found    bool
	)
	for _, read := range reads {
		index, stored := indexOf[read]
		if !stored {
			continue
		}
		r, measured := rates[index]
		if !measured {
			var err error
			if r.median, r.ok, err = w.svc.newestInterval(ctx, index); err != nil {
				return nil, err
			}
			rates[index] = r
		}
		if r.ok && (!found || r.median < shortest) {
			primary, shortest, found = read, r.median, true
		}
	}
	return &primary, nil
}

// newestInterval returns the median of the newest intervals of the index channel with
// key index, all from its newest stretch, and false when that stretch holds fewer than
// 2 samples.
func (s *Service) newestInterval(
	ctx context.Context,
	index channel.Key,
) (median telem.TimeSpan, ok bool, err error) {
	it, err := s.framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:      channel.Keys{index},
		Bounds:    telem.TimeRangeMax,
		ChunkSize: intervalWindow + 1,
	})
	if err != nil {
		return 0, false, err
	}
	defer func() { err = errors.Combine(err, it.Close()) }()
	if !it.SeekLast() || !it.Prev(framer.IteratorAutoSpan) {
		return 0, false, it.Error()
	}
	spans := newestSpans(it.Value().Get(index).Series)
	if len(spans) == 0 {
		return 0, false, nil
	}
	return medianOf(spans), true, nil
}

// newestSpans returns up to intervalWindow intervals between the newest stamps in
// series that share the alignment domain of the newest one, so the gap between two
// stretches is never an interval.
func newestSpans(series []telem.Series) []telem.TimeSpan {
	if len(series) == 0 {
		return nil
	}
	domain := series[len(series)-1].Alignment.DomainIndex()
	spans := make([]telem.TimeSpan, 0, intervalWindow)
	var (
		later telem.TimeStamp
		held  bool
	)
	for _, s := range slices.Backward(series) {
		if s.Alignment.DomainIndex() != domain {
			break
		}
		for j := s.Len() - 1; j >= 0; j-- {
			stamp := s.ValueAt[telem.TimeStamp](int(j))
			if held {
				spans = append(spans, stamp.Span(later))
				if len(spans) == intervalWindow {
					return spans
				}
			}
			later, held = stamp, true
		}
	}
	return spans
}

// medianOf returns the median of spans, which it sorts, as the join computes it: the
// midpoint of the two middle spans when their number is even.
func medianOf(spans []telem.TimeSpan) telem.TimeSpan {
	slices.Sort(spans)
	n := len(spans)
	median := spans[n/2]
	if n%2 == 0 {
		median = spans[n/2-1] + (median-spans[n/2-1])/2
	}
	return median
}
