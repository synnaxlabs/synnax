// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"testing"

	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// filterCases size a frame and the key list a caller demands from it. Frames of 100
// entries or fewer exercise the mask path, larger ones the copy path. A demand longer
// than the frame is the common relay shape: one writer sends a few channels to a
// subscriber that reads many.
var filterCases = []struct {
	name     string
	total    int
	demanded int
}{
	{name: "10Frame/5Demanded", total: 10, demanded: 5},
	{name: "10Frame/100Demanded", total: 10, demanded: 100},
	{name: "25Frame/25Demanded", total: 25, demanded: 25},
	{name: "50Frame/50Demanded", total: 50, demanded: 50},
	{name: "100Frame/50Demanded", total: 100, demanded: 50},
	{name: "100Frame/500Demanded", total: 100, demanded: 500},
	{name: "500Frame/50Demanded", total: 500, demanded: 50},
	{name: "500Frame/500Demanded", total: 500, demanded: 500},
}

// newFilterFrame builds a frame of total series and the key list a caller demands from
// it. Demanded keys the frame does not carry are interleaved with the ones it does, so
// a match lands at an average position in the list instead of at its front.
func newFilterFrame(total, demanded int) (telem.Frame[int32], []int32) {
	keys := make([]int32, total)
	series := make([]telem.Series, total)
	for i := range keys {
		keys[i] = int32(i)
		series[i] = telem.NewSeriesV(float64(i))
	}
	var (
		demandedKeys = make([]int32, demanded)
		present      = min(demanded, total)
		stride       = demanded / present
		absent       = int32(total)
	)
	for i := range demandedKeys {
		if i%stride == 0 && i/stride < present {
			demandedKeys[i] = keys[(i/stride)*(total/present)]
		} else {
			demandedKeys[i] = absent
			absent++
		}
	}
	return telem.MultiFrame(keys, series), demandedKeys
}

// BenchmarkKeepKeys benchmarks filtering a frame against a key slice.
func BenchmarkKeepKeys(b *testing.B) {
	for _, c := range filterCases {
		fr, demandedKeys := newFilterFrame(c.total, c.demanded)
		b.Run(c.name, func(b *testing.B) {
			for b.Loop() {
				_ = fr.KeepKeys(demandedKeys)
			}
		})
	}
}

// BenchmarkKeepKeysSet benchmarks filtering a frame against a pre-built key set.
func BenchmarkKeepKeysSet(b *testing.B) {
	for _, c := range filterCases {
		fr, demandedKeys := newFilterFrame(c.total, c.demanded)
		keys := set.New(demandedKeys...)
		b.Run(c.name, func(b *testing.B) {
			for b.Loop() {
				_ = fr.KeepKeysSet(keys)
			}
		})
	}
}
