// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/telem"
)

func newHTTPFramerCodec(keys channel.Keys, dts []telem.DataType) *framer.Codec {
	return &framer.Codec{
		Codec:          codec.NewStatic(keys, dts),
		LowerPerfCodec: json.Codec,
	}
}

func httpBenchFrame(numChannels, samples int) (channel.Keys, []telem.DataType, frame.Frame) {
	keys := make(channel.Keys, numChannels)
	dts := make([]telem.DataType, numChannels)
	for i := range numChannels {
		keys[i] = channel.Key(i + 1)
		dts[i] = telem.Float32T
	}
	data := make([]float32, samples)
	for i := range samples {
		data[i] = float32(i)
	}
	series := make([]telem.Series, numChannels)
	frameKeys := make(channel.Keys, numChannels)
	for i := range numChannels {
		series[i] = telem.NewSeries(data)
		frameKeys[i] = keys[i]
	}
	return keys, dts, frame.NewMulti(frameKeys, series)
}

func httpIteratorFrame(numChannels, numDomains, samples int) (channel.Keys, []telem.DataType, frame.Frame) {
	keys := make(channel.Keys, numChannels)
	dts := make([]telem.DataType, numChannels)
	for i := range numChannels {
		keys[i] = channel.Key(i + 1)
		dts[i] = telem.Float32T
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
	return keys, dts, frame.NewMulti(frameKeys, series)
}

func BenchmarkHTTPCodec_WriteRequest_Encode(b *testing.B) {
	for _, nc := range []int{1, 8, 64} {
		b.Run(fmt.Sprintf("channels=%d", nc), func(b *testing.B) {
			keys, dts, fr := httpBenchFrame(nc, 100)
			c := newHTTPFramerCodec(keys, dts)
			msg := http.WSMessage[framer.WriterRequest]{
				Type: "data",
				Payload: framer.WriterRequest{
					Command: writer.CommandWrite,
					Frame:   fr,
				},
			}
			w := bytes.NewBuffer(nil)
			b.ReportAllocs()
			for b.Loop() {
				w.Reset()
				if err := c.EncodeStream(b.Context(), w, msg); err != nil {
					b.Fatalf("encode: %v", err)
				}
			}
		})
	}
}

func BenchmarkHTTPCodec_WriteRequest_Decode(b *testing.B) {
	for _, nc := range []int{1, 8, 64} {
		b.Run(fmt.Sprintf("channels=%d", nc), func(b *testing.B) {
			keys, dts, fr := httpBenchFrame(nc, 100)
			c := newHTTPFramerCodec(keys, dts)
			msg := http.WSMessage[framer.WriterRequest]{
				Type: "data",
				Payload: framer.WriterRequest{
					Command: writer.CommandWrite,
					Frame:   fr,
				},
			}
			encoded, err := c.Encode(b.Context(), msg)
			if err != nil {
				b.Fatalf("encode: %v", err)
			}
			var dec http.WSMessage[framer.WriterRequest]
			b.ReportAllocs()
			for b.Loop() {
				dec = http.WSMessage[framer.WriterRequest]{}
				if err := c.Decode(b.Context(), encoded, &dec); err != nil {
					b.Fatalf("decode: %v", err)
				}
			}
		})
	}
}

func BenchmarkHTTPCodec_StreamerResponse_Encode(b *testing.B) {
	for _, nc := range []int{1, 8, 64} {
		b.Run(fmt.Sprintf("channels=%d", nc), func(b *testing.B) {
			keys, dts, fr := httpBenchFrame(nc, 100)
			c := newHTTPFramerCodec(keys, dts)
			msg := http.WSMessage[framer.StreamerResponse]{
				Type:    "data",
				Payload: framer.StreamerResponse{Frame: fr},
			}
			w := bytes.NewBuffer(nil)
			b.ReportAllocs()
			for b.Loop() {
				w.Reset()
				if err := c.EncodeStream(b.Context(), w, msg); err != nil {
					b.Fatalf("encode: %v", err)
				}
			}
		})
	}
}

func BenchmarkHTTPCodec_StreamerResponse_Decode(b *testing.B) {
	for _, nc := range []int{1, 8, 64} {
		b.Run(fmt.Sprintf("channels=%d", nc), func(b *testing.B) {
			keys, dts, fr := httpBenchFrame(nc, 100)
			c := newHTTPFramerCodec(keys, dts)
			msg := http.WSMessage[framer.StreamerResponse]{
				Type:    "data",
				Payload: framer.StreamerResponse{Frame: fr},
			}
			encoded, err := c.Encode(b.Context(), msg)
			if err != nil {
				b.Fatalf("encode: %v", err)
			}
			var dec http.WSMessage[framer.StreamerResponse]
			b.ReportAllocs()
			for b.Loop() {
				dec = http.WSMessage[framer.StreamerResponse]{}
				if err := c.Decode(b.Context(), encoded, &dec); err != nil {
					b.Fatalf("decode: %v", err)
				}
			}
		})
	}
}

// BenchmarkHTTPCodec_IteratorResponse_Encode covers the new path SY-3556 adds to the
// HTTP codec: an iterator data response is encoded via the high-perf codec instead of
// JSON. Cases mirror the iterator's typical "across-domains" shape where each channel
// appears once per time domain.
func BenchmarkHTTPCodec_IteratorResponse_Encode(b *testing.B) {
	cases := []struct {
		channels, domains, samples int
	}{
		{1, 1, 100},
		{8, 1, 100},
		{8, 10, 100},
		{64, 10, 100},
	}
	for _, cs := range cases {
		name := fmt.Sprintf("channels=%d/domains=%d/samples=%d", cs.channels, cs.domains, cs.samples)
		b.Run(name, func(b *testing.B) {
			keys, dts, fr := httpIteratorFrame(cs.channels, cs.domains, cs.samples)
			c := newHTTPFramerCodec(keys, dts)
			msg := http.WSMessage[framer.IteratorResponse]{
				Type: "data",
				Payload: framer.IteratorResponse{
					Variant: iterator.ResponseVariantData,
					Command: iterator.CommandNext,
					Frame:   fr,
				},
			}
			w := bytes.NewBuffer(nil)
			b.ReportAllocs()
			for b.Loop() {
				w.Reset()
				if err := c.EncodeStream(b.Context(), w, msg); err != nil {
					b.Fatalf("encode: %v", err)
				}
			}
		})
	}
}

func BenchmarkHTTPCodec_IteratorResponse_Decode(b *testing.B) {
	cases := []struct {
		channels, domains, samples int
	}{
		{1, 1, 100},
		{8, 10, 100},
		{64, 10, 100},
	}
	for _, cs := range cases {
		name := fmt.Sprintf("channels=%d/domains=%d/samples=%d", cs.channels, cs.domains, cs.samples)
		b.Run(name, func(b *testing.B) {
			keys, dts, fr := httpIteratorFrame(cs.channels, cs.domains, cs.samples)
			c := newHTTPFramerCodec(keys, dts)
			msg := http.WSMessage[framer.IteratorResponse]{
				Type: "data",
				Payload: framer.IteratorResponse{
					Variant: iterator.ResponseVariantData,
					Command: iterator.CommandNext,
					Frame:   fr,
				},
			}
			encoded, err := c.Encode(b.Context(), msg)
			if err != nil {
				b.Fatalf("encode: %v", err)
			}
			var dec http.WSMessage[framer.IteratorResponse]
			b.ReportAllocs()
			for b.Loop() {
				dec = http.WSMessage[framer.IteratorResponse]{}
				if err := c.Decode(b.Context(), encoded, &dec); err != nil {
					b.Fatalf("decode: %v", err)
				}
			}
		})
	}
}

// BenchmarkHTTPCodec_IteratorRequest_Encode measures the request side of the new
// iterator HTTP codec wiring. Iterator requests stay on the JSON path (lowPerfEncode) —
// this is what we pay for the lowest-frequency control messages (Open / Next /
// SeekFirst).
func BenchmarkHTTPCodec_IteratorRequest_Encode(b *testing.B) {
	keys, dts, _ := httpIteratorFrame(8, 1, 1)
	c := newHTTPFramerCodec(keys, dts)
	msg := http.WSMessage[framer.IteratorRequest]{
		Type: "data",
		Payload: framer.IteratorRequest{
			Command: iterator.CommandNext,
			Span:    telem.Second,
			Keys:    keys,
		},
	}
	w := bytes.NewBuffer(nil)
	b.ReportAllocs()
	for b.Loop() {
		w.Reset()
		if err := c.EncodeStream(b.Context(), w, msg); err != nil {
			b.Fatalf("encode: %v", err)
		}
	}
}
