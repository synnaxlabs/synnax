// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package inspect

import (
	"slices"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/x/telem"
)

// ChannelKey identifies a channel within the inspected database.
type ChannelKey = channel.Key

// Distribution summarizes the spread of an int64-valued sample set.
type Distribution[T ~int64] struct {
	Min T `json:"min"`
	P50 T `json:"p50"`
	P99 T `json:"p99"`
	Max T `json:"max"`
}

// newDistribution computes a distribution over values, which it sorts in place. An
// empty input returns the zero distribution.
func newDistribution[T ~int64](values []T) Distribution[T] {
	if len(values) == 0 {
		return Distribution[T]{}
	}
	slices.Sort(values)
	return Distribution[T]{
		Min: values[0],
		P50: values[len(values)/2],
		P99: values[len(values)*99/100],
		Max: values[len(values)-1],
	}
}

// ChannelStats reports per-channel storage statistics. All byte totals are computed
// from the index and file sizes, never from reading sample data.
type ChannelStats struct {
	// Domains is the number of domains in the channel's index.
	Domains int `json:"domains"`
	// Files is the number of data files in the channel directory.
	Files int `json:"files"`
	// GCEligibleFiles is the number of data files whose garbage exceeds the GC
	// threshold, making them eligible for collection.
	GCEligibleFiles int `json:"gc_eligible_files"`
	// LiveBytes is the total size of all domains, i.e. bytes reachable via the index.
	LiveBytes telem.Size `json:"live_bytes"`
	// DiskBytes is the total on-disk size of the channel directory.
	DiskBytes telem.Size `json:"disk_bytes"`
	// GarbageBytes is the total size of data-file bytes no domain references.
	GarbageBytes telem.Size `json:"garbage_bytes"`
	// GarbageRatio is GarbageBytes over total data-file bytes; zero with no files.
	GarbageRatio float64 `json:"garbage_ratio"`
	// Samples is the number of samples stored in the channel.
	Samples int64 `json:"samples"`
	// SamplesExact is false when Samples could not be derived: variable-length
	// channels require deep checks to count samples.
	SamplesExact bool `json:"samples_exact"`
	// TimeRange spans from the first domain's start to the last domain's end.
	TimeRange telem.TimeRange `json:"time_range"`
	// DomainSizes is the distribution of domain byte sizes.
	DomainSizes Distribution[telem.Size] `json:"domain_sizes"`
	// DomainSpans is the distribution of domain time spans.
	DomainSpans Distribution[telem.TimeSpan] `json:"domain_spans"`
	// Gaps is the number of positive gaps between adjacent domains.
	Gaps int `json:"gaps"`
	// MicroGaps is the number of gaps within the configured near-zero band.
	MicroGaps int `json:"micro_gaps"`
	// TinyDomains is the number of domains below the configured size or span floor.
	TinyDomains int `json:"tiny_domains"`
}

// ChannelReport is the result of inspecting one channel.
type ChannelReport struct {
	// Key is the channel's key, parsed from its directory name.
	Key channel.Key `json:"key"`
	// Channel is the decoded metadata; zero when meta.json could not be decoded.
	Channel channel.Channel `json:"channel"`
	// Stats is zero for virtual channels, which store no data.
	Stats ChannelStats `json:"stats"`
	// Findings holds every finding about this channel.
	Findings []Finding `json:"findings"`
}

// Totals aggregates statistics across every inspected channel.
type Totals struct {
	// Channels is the number of channel directories inspected.
	Channels int `json:"channels"`
	// VirtualChannels is the number of those that are virtual.
	VirtualChannels int `json:"virtual_channels"`
	// Domains sums ChannelStats.Domains.
	Domains int `json:"domains"`
	// LiveBytes sums ChannelStats.LiveBytes.
	LiveBytes telem.Size `json:"live_bytes"`
	// DiskBytes sums ChannelStats.DiskBytes.
	DiskBytes telem.Size `json:"disk_bytes"`
	// GarbageBytes sums ChannelStats.GarbageBytes.
	GarbageBytes telem.Size `json:"garbage_bytes"`
	// Samples sums ChannelStats.Samples where exact.
	Samples int64 `json:"samples"`
	// Errors counts error-level findings across the report.
	Errors int `json:"errors"`
	// Warnings counts warning-level findings across the report.
	Warnings int `json:"warnings"`
	// Infos counts info-level findings across the report.
	Infos int `json:"infos"`
}

// Report is the result of inspecting a Cesium database.
type Report struct {
	// Channels holds one report per channel directory, ordered by key.
	Channels []ChannelReport `json:"channels"`
	// Findings holds findings about the database root, outside any channel.
	Findings []Finding `json:"findings"`
	// Totals aggregates across the whole database.
	Totals Totals `json:"totals"`
}

// aggregate computes Totals from the report's channels and findings.
func (r *Report) aggregate() {
	t := Totals{Channels: len(r.Channels)}
	count := func(fs []Finding) {
		for _, f := range fs {
			switch f.Severity {
			case SeverityError:
				t.Errors++
			case SeverityWarning:
				t.Warnings++
			case SeverityInfo:
				t.Infos++
			}
		}
	}
	count(r.Findings)
	for _, ch := range r.Channels {
		count(ch.Findings)
		if ch.Channel.Virtual {
			t.VirtualChannels++
		}
		t.Domains += ch.Stats.Domains
		t.LiveBytes += ch.Stats.LiveBytes
		t.DiskBytes += ch.Stats.DiskBytes
		t.GarbageBytes += ch.Stats.GarbageBytes
		if ch.Stats.SamplesExact {
			t.Samples += ch.Stats.Samples
		}
	}
	r.Totals = t
}
