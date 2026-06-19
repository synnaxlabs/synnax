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
	"fmt"
	"testing"

	"github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/codec"
	"github.com/synnaxlabs/x/telem"
)

// benchFrame builds a multi-channel frame of float32 series for translator
// benchmarks. The sample shape (numChannels x samplesPerSeries) is the same
// shape the writer translator sees on a steady-state hot stream.
func benchFrame(numChannels, samplesPerSeries int) (
	channel.Keys, []telem.DataType, frame.Frame,
) {
	keys := make(channel.Keys, numChannels)
	dataTypes := make([]telem.DataType, numChannels)
	for i := range numChannels {
		keys[i] = channel.Key(i + 1)
		dataTypes[i] = telem.Float32T
	}
	data := make([]float32, samplesPerSeries)
	for i := range samplesPerSeries {
		data[i] = float32(i)
	}
	series := make([]telem.Series, numChannels)
	frameKeys := make(channel.Keys, numChannels)
	for i := range numChannels {
		series[i] = telem.NewSeries(data)
		frameKeys[i] = keys[i]
	}
	return keys, dataTypes, frame.NewMulti(frameKeys, series)
}

// benchIteratorFrame mirrors the iterator's "across-domains" shape: each
// channel appears numDomains times (one series per time domain). This is the
// frame shape SY-3556 actually optimizes — the request asks the iterator to
// span more than one domain and the response carries one series per domain per
// channel.
func benchIteratorFrame(numChannels, numDomains, samples int) (
	channel.Keys, []telem.DataType, frame.Frame,
) {
	keys := make(channel.Keys, numChannels)
	dataTypes := make([]telem.DataType, numChannels)
	for i := range numChannels {
		keys[i] = channel.Key(i + 1)
		dataTypes[i] = telem.Float32T
	}
	data := make([]float32, samples)
	for i := range samples {
		data[i] = float32(i)
	}
	total := numChannels * numDomains
	series := make([]telem.Series, total)
	frameKeys := make(channel.Keys, total)
	idx := 0
	for d := range numDomains {
		alignment := telem.Alignment(d * 1_000_000)
		tr := telem.TimeStamp(d).SpanRange(telem.Second)
		for c := range numChannels {
			s := telem.NewSeries(data)
			s.Alignment = alignment
			s.TimeRange = tr
			series[idx] = s
			frameKeys[idx] = keys[c]
			idx++
		}
	}
	return keys, dataTypes, frame.NewMulti(frameKeys, series)
}

// BenchmarkWriterRequestTranslator_Forward_BufferOnly simulates a hypothetical
// "fixed" Forward that skips telempb.FrameToPB on the codec path. The
// difference from BenchmarkWriterRequestTranslator_Forward is the cost of
// building the redundant protobuf Frame field that the receiver discards.
func BenchmarkWriterRequestTranslator_Forward_BufferOnly(b *testing.B) {
	for _, nc := range []int{1, 8, 64} {
		b.Run(fmt.Sprintf("channels=%d", nc), func(b *testing.B) {
			keys, dataTypes, fr := benchFrame(nc, 100)
			cdec := codec.NewStatic(keys, dataTypes)
			b.ReportAllocs()
			for b.Loop() {
				buf, err := cdec.Encode(b.Context(), fr)
				if err != nil {
					b.Fatalf("encode: %v", err)
				}
				_ = &WriterRequest{
					Command: int32(writer.CommandWrite),
					Buffer:  buf,
				}
			}
		})
	}
}

func BenchmarkWriterRequestTranslator_Forward(b *testing.B) {
	for _, nc := range []int{1, 8, 64} {
		b.Run(fmt.Sprintf("channels=%d", nc), func(b *testing.B) {
			keys, dataTypes, fr := benchFrame(nc, 100)
			cdec := codec.NewStatic(keys, dataTypes)
			t := frameWriterRequestTranslator{codec: cdec}
			req := framer.WriterRequest{
				Command: writer.CommandWrite,
				Frame:   fr,
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := t.Forward(b.Context(), req); err != nil {
					b.Fatalf("forward: %v", err)
				}
			}
		})
	}
}

// BenchmarkWriterRequestTranslator_Backward measures the steady-state hot path:
// Config is nil (set only on Open), Buffer carries the codec-encoded frame.
// This is what the server sees per Write after the initial handshake.
func BenchmarkWriterRequestTranslator_Backward(b *testing.B) {
	for _, nc := range []int{1, 8, 64} {
		b.Run(fmt.Sprintf("channels=%d", nc), func(b *testing.B) {
			keys, dataTypes, fr := benchFrame(nc, 100)
			cdec := codec.NewStatic(keys, dataTypes)
			buf, err := cdec.Encode(b.Context(), fr)
			if err != nil {
				b.Fatalf("encode: %v", err)
			}
			pb := &WriterRequest{
				Command: int32(writer.CommandWrite),
				Buffer:  buf,
			}
			t := frameWriterRequestTranslator{codec: cdec}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := t.Backward(b.Context(), pb); err != nil {
					b.Fatalf("backward: %v", err)
				}
			}
		})
	}
}

func BenchmarkIteratorRequestTranslator_RoundTrip(b *testing.B) {
	keys, dataTypes, _ := benchIteratorFrame(8, 1, 100)
	cdec := codec.NewStatic(keys, dataTypes)
	t := frameIteratorRequestTranslator{codec: cdec}
	req := framer.IteratorRequest{
		Command: iterator.CommandNext,
		Span:    telem.Second,
	}
	b.ReportAllocs()
	for b.Loop() {
		pb, err := t.Forward(b.Context(), req)
		if err != nil {
			b.Fatalf("forward: %v", err)
		}
		if _, err := t.Backward(b.Context(), pb); err != nil {
			b.Fatalf("backward: %v", err)
		}
	}
}

// BenchmarkIteratorResponseTranslator_Forward measures the new high-perf path
// (codec.Encode into Buffer). The fall-back path was the only option on rc;
// here we measure what SY-3556 introduces.
func BenchmarkIteratorResponseTranslator_Forward(b *testing.B) {
	cases := []struct {
		channels, domains, samples int
	}{
		{1, 1, 100},
		{8, 1, 100},
		{8, 10, 100},
		{64, 10, 100},
	}
	for _, c := range cases {
		name := fmt.Sprintf("channels=%d/domains=%d/samples=%d", c.channels, c.domains, c.samples)
		b.Run(name, func(b *testing.B) {
			keys, dataTypes, fr := benchIteratorFrame(c.channels, c.domains, c.samples)
			cdec := codec.NewStatic(keys, dataTypes)
			t := frameIteratorResponseTranslator{codec: cdec}
			res := framer.IteratorResponse{
				Variant: iterator.ResponseVariantData,
				Command: iterator.CommandNext,
				Frame:   fr,
			}
			ctx := context.Background()
			pb, err := t.Forward(ctx, res)
			if err != nil {
				b.Fatalf("warmup: %v", err)
			}
			b.ReportMetric(float64(len(pb.Buffer)), "wireB")
			b.ReportAllocs()
			for b.Loop() {
				if _, err := t.Forward(ctx, res); err != nil {
					b.Fatalf("forward: %v", err)
				}
			}
		})
	}
}

// BenchmarkIteratorResponseTranslator_Forward_NoCodec exercises the fall-back
// path that was the rc default: serialize the frame to protobuf. Comparing this
// against the codec path shows the speed-up SY-3556 buys.
func BenchmarkIteratorResponseTranslator_Forward_NoCodec(b *testing.B) {
	cases := []struct {
		channels, domains, samples int
	}{
		{1, 1, 100},
		{8, 1, 100},
		{8, 10, 100},
		{64, 10, 100},
	}
	for _, c := range cases {
		name := fmt.Sprintf("channels=%d/domains=%d/samples=%d", c.channels, c.domains, c.samples)
		b.Run(name, func(b *testing.B) {
			_, _, fr := benchIteratorFrame(c.channels, c.domains, c.samples)
			t := frameIteratorResponseTranslator{}
			res := framer.IteratorResponse{
				Variant: iterator.ResponseVariantData,
				Command: iterator.CommandNext,
				Frame:   fr,
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := t.Forward(b.Context(), res); err != nil {
					b.Fatalf("forward: %v", err)
				}
			}
		})
	}
}

func BenchmarkIteratorResponseTranslator_Backward(b *testing.B) {
	cases := []struct {
		channels, domains, samples int
	}{
		{1, 1, 100},
		{8, 10, 100},
		{64, 10, 100},
	}
	for _, c := range cases {
		name := fmt.Sprintf("channels=%d/domains=%d/samples=%d", c.channels, c.domains, c.samples)
		b.Run(name, func(b *testing.B) {
			keys, dataTypes, fr := benchIteratorFrame(c.channels, c.domains, c.samples)
			cdec := codec.NewStatic(keys, dataTypes)
			t := frameIteratorResponseTranslator{codec: cdec}
			ctx := context.Background()
			pb, err := t.Forward(ctx, framer.IteratorResponse{
				Variant: iterator.ResponseVariantData,
				Command: iterator.CommandNext,
				Frame:   fr,
			})
			if err != nil {
				b.Fatalf("forward: %v", err)
			}
			// reset to a fresh codec so each Backward goes through the same
			// decode work — Decode itself doesn't mutate codec state.
			b.ReportAllocs()
			for b.Loop() {
				if _, err := t.Backward(ctx, pb); err != nil {
					b.Fatalf("backward: %v", err)
				}
			}
		})
	}
}

func BenchmarkStreamerResponseTranslator_RoundTrip(b *testing.B) {
	for _, nc := range []int{1, 8, 64} {
		b.Run(fmt.Sprintf("channels=%d", nc), func(b *testing.B) {
			keys, dataTypes, fr := benchFrame(nc, 100)
			cdec := codec.NewStatic(keys, dataTypes)
			t := frameStreamerResponseTranslator{codec: cdec}
			res := framer.StreamerResponse{Frame: fr}
			ctx := context.Background()
			b.ReportAllocs()
			for b.Loop() {
				pb, err := t.Forward(ctx, res)
				if err != nil {
					b.Fatalf("forward: %v", err)
				}
				if _, err := t.Backward(ctx, pb); err != nil {
					b.Fatalf("backward: %v", err)
				}
			}
		})
	}
}
