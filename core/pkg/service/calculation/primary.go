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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// intervalWindow is the number of an input's newest intervals that set its rate, the
// same number that sets its hold limit in the join.
const intervalWindow = 16

// PrimaryStatusKey returns the key of the status that tells why the primary of the
// calculation with key moved. A later move replaces it.
func PrimaryStatusKey(key Key) status.Key {
	return "calculation_primary_" + key.String()
}

// settlePrimary returns the primary that a calculation whose expression reads reads
// stores after a save. held is the primary it stored before the save, and requested the
// one the save leaves. A candidate is a stored read, and a virtual read is a candidate
// only when every read is virtual. A save that sets the primary must name a candidate,
// or none to have one picked. Otherwise a held primary that is still a candidate stays,
// and the calculation picks one.
func (w Writer) settlePrimary(
	ctx context.Context,
	held, requested *channel.Key,
	set bool,
	reads []channel.Key,
) (*channel.Key, error) {
	indexOf, err := w.storedIndexes(ctx, reads)
	if err != nil {
		return nil, err
	}
	candidate := func(key channel.Key) bool {
		_, stored := indexOf[key]
		return stored || len(indexOf) == 0
	}
	switch {
	case set && requested != nil:
		if !slices.Contains(reads, *requested) {
			return nil, validate.PathedError(errors.Wrapf(
				validate.ErrValidation,
				"channel %s is not a channel that the expression reads",
				*requested,
			), "primary")
		}
		if !candidate(*requested) {
			return nil, validate.PathedError(errors.Wrapf(
				validate.ErrValidation,
				"channel %s is virtual, so it cannot be the primary of an expression "+
					"that reads a stored channel",
				*requested,
			), "primary")
		}
		return requested, nil
	case !set && held != nil && slices.Contains(reads, *held) && candidate(*held):
		return held, nil
	}
	return w.pickPrimary(ctx, reads, indexOf)
}

// setsPrimary reports whether acts change the primary.
func setsPrimary(acts []Action) bool {
	return slices.ContainsFunc(acts, func(a Action) bool {
		return a.Set != nil && slices.Equal(a.Set.Path, []string{"primary"}) ||
			a.Remove != nil && slices.Equal(a.Remove.Path, []string{"primary"})
	})
}

// storedIndexes returns the index of each stored channel in reads.
func (w Writer) storedIndexes(
	ctx context.Context,
	reads []channel.Key,
) (map[channel.Key]channel.Key, error) {
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
	return indexOf, nil
}

// pickPrimary returns the stored read whose index has the shortest median interval
// over its newest stored stretch, or the earlier read on a tie. indexOf holds the index
// of each stored read. A read with fewer than 2 samples in that stretch is not timed.
// With no timed read it returns the first stored read, with no stored read the first
// read, and with no reads nil.
func (w Writer) pickPrimary(
	ctx context.Context,
	reads []channel.Key,
	indexOf map[channel.Key]channel.Key,
) (*channel.Key, error) {
	if len(reads) == 0 {
		return nil, nil
	}
	// Storage makes a channel when the transaction that creates it commits, so an index
	// that this transaction creates holds no samples yet.
	var committed []channel.Channel
	if err := w.channels.NewRetrieve().
		Where(channel.MatchKeys(lo.Uniq(lo.Values(indexOf))...)).
		SkipMissingKeys().
		Entries(&committed).
		Exec(ctx, w.svc.db); err != nil {
		return nil, err
	}
	stored := make(set.Set[channel.Key], len(committed))
	for _, ch := range committed {
		stored.Add(ch.Key())
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
	if i := slices.IndexFunc(reads, func(read channel.Key) bool {
		_, ok := indexOf[read]
		return ok
	}); i >= 0 {
		primary = reads[i]
	}
	for _, read := range reads {
		index, ok := indexOf[read]
		if !ok || !stored.Contains(index) {
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
