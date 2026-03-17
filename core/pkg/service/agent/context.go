// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package agent

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/telem"
)

const (
	maxSearchResults = 20
	statsLookback    = 1 * time.Hour
	iterChunkSize    = 10000
)

// ChannelContextProvider searches for channels matching the user's query and
// provides their metadata and recent statistics to the LLM.
type ChannelContextProvider struct {
	Channel *channel.Service
	Framer  *framer.Service
}

var _ ContextProvider = (*ChannelContextProvider)(nil)

func (p *ChannelContextProvider) Name() string { return "channels" }

func (p *ChannelContextProvider) BuildContext(ctx context.Context, query string) (string, error) {
	channels, err := p.searchChannels(ctx, query)
	if err != nil || len(channels) == 0 {
		return "", err
	}
	stats := computeChannelStats(ctx, p.Framer, channels)
	var b strings.Builder
	fmt.Fprintf(&b, "Available Channels:\n")
	for i, ch := range channels {
		flags := ""
		if ch.Virtual {
			flags += ", virtual"
		}
		arcType := types.FromTelem(ch.DataType).String()
		fmt.Fprintf(&b, "- %s (key=%d, type=%s%s)\n", ch.Name, ch.Key(), arcType, flags)
		s := stats[i]
		if s.Count > 0 {
			fmt.Fprintf(&b, "  Recent stats: mean=%.4f, std=%.4f, min=%.4f, max=%.4f (samples=%d)\n",
				s.Mean, s.StdDev, s.Min, s.Max, s.Count)
		} else {
			fmt.Fprintf(&b, "  No recent data available\n")
		}
	}
	return b.String(), nil
}

func (p *ChannelContextProvider) searchChannels(ctx context.Context, query string) ([]channel.Channel, error) {
	var channels []channel.Channel
	if err := p.Channel.NewRetrieve().
		Search(query).
		WhereInternal(false).
		WhereIsIndex(false).
		Limit(maxSearchResults).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	if len(channels) == 0 {
		if err := p.Channel.NewRetrieve().
			WhereInternal(false).
			WhereIsIndex(false).
			Limit(maxSearchResults).
			Entries(&channels).
			Exec(ctx, nil); err != nil {
			return nil, err
		}
	}
	return channels, nil
}

type stats struct {
	Mean   float64
	StdDev float64
	Min    float64
	Max    float64
	Count  int64
}

func computeChannelStats(
	ctx context.Context,
	framerSvc *framer.Service,
	channels []channel.Channel,
) []stats {
	result := make([]stats, len(channels))
	if framerSvc == nil {
		return result
	}
	keys := channel.KeysFromChannels(channels)
	now := telem.Now()
	start := now.Add(-telem.TimeSpan(statsLookback))
	iter, err := framerSvc.OpenIterator(ctx, framer.IteratorConfig{
		Keys:      keys,
		Bounds:    telem.TimeRange{Start: start, End: now},
		ChunkSize: iterChunkSize,
	})
	if err != nil {
		return result
	}
	defer iter.Close()

	accumulators := make(map[channel.Key]*statsAccumulator)
	for _, k := range keys {
		accumulators[k] = &statsAccumulator{}
	}
	if !iter.SeekFirst() {
		return result
	}
	for iter.Next(telem.TimeSpanMax) {
		f := iter.Value()
		for key, series := range f.Entries() {
			if acc, ok := accumulators[key]; ok {
				acc.addSeries(series)
			}
		}
	}
	for i, ch := range channels {
		if acc, ok := accumulators[ch.Key()]; ok {
			result[i] = acc.finalize()
		}
	}
	return result
}

type statsAccumulator struct {
	sum    float64
	sumSq  float64
	min    float64
	max    float64
	count  int64
	hasVal bool
}

func (a *statsAccumulator) addSeries(s telem.Series) {
	n := s.Len()
	for i := int64(0); i < n; i++ {
		v := readFloat64(s, i)
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		a.sum += v
		a.sumSq += v * v
		if !a.hasVal {
			a.min = v
			a.max = v
			a.hasVal = true
		} else {
			if v < a.min {
				a.min = v
			}
			if v > a.max {
				a.max = v
			}
		}
		a.count++
	}
}

func (a *statsAccumulator) finalize() stats {
	if a.count == 0 {
		return stats{}
	}
	mean := a.sum / float64(a.count)
	variance := (a.sumSq / float64(a.count)) - (mean * mean)
	if variance < 0 {
		variance = 0
	}
	return stats{
		Mean:   mean,
		StdDev: math.Sqrt(variance),
		Min:    a.min,
		Max:    a.max,
		Count:  a.count,
	}
}

func readFloat64(s telem.Series, i int64) float64 {
	switch s.DataType {
	case telem.Float64T:
		return telem.ValueAt[float64](s, int(i))
	case telem.Float32T:
		return float64(telem.ValueAt[float32](s, int(i)))
	case telem.Int64T:
		return float64(telem.ValueAt[int64](s, int(i)))
	case telem.Int32T:
		return float64(telem.ValueAt[int32](s, int(i)))
	case telem.Int16T:
		return float64(telem.ValueAt[int16](s, int(i)))
	case telem.Int8T:
		return float64(telem.ValueAt[int8](s, int(i)))
	case telem.Uint64T:
		return float64(telem.ValueAt[uint64](s, int(i)))
	case telem.Uint32T:
		return float64(telem.ValueAt[uint32](s, int(i)))
	case telem.Uint16T:
		return float64(telem.ValueAt[uint16](s, int(i)))
	case telem.Uint8T:
		return float64(telem.ValueAt[uint8](s, int(i)))
	default:
		return 0
	}
}
