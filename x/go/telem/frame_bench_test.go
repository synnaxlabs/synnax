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

// filterCases size a frame and the subset of it a caller keeps. 100 entries exercise
// the mask path, 500 the copy path above the 128-entry mask limit.
var filterCases = []struct {
	name  string
	total int
	kept  int
}{
	{name: "100Total/50Kept", total: 100, kept: 50},
	{name: "500Total/50Kept", total: 500, kept: 50},
	{name: "500Total/500Kept", total: 500, kept: 500},
}

// newFilterFrame builds a frame of total series and the kept keys to filter it down
// to, spread evenly across the frame.
func newFilterFrame(total, kept int) (telem.Frame[int32], []int32) {
	keys := make([]int32, total)
	series := make([]telem.Series, total)
	for i := range keys {
		keys[i] = int32(i)
		series[i] = telem.NewSeriesV(float64(i))
	}
	keptKeys := make([]int32, kept)
	for i := range keptKeys {
		keptKeys[i] = keys[i*(total/kept)]
	}
	return telem.MultiFrame(keys, series), keptKeys
}

// BenchmarkKeepKeys benchmarks filtering a frame against a key slice.
func BenchmarkKeepKeys(b *testing.B) {
	for _, c := range filterCases {
		fr, keptKeys := newFilterFrame(c.total, c.kept)
		b.Run(c.name, func(b *testing.B) {
			for b.Loop() {
				_ = fr.KeepKeys(keptKeys)
			}
		})
	}
}

// BenchmarkKeepKeysSet benchmarks filtering a frame against a pre-built key set.
func BenchmarkKeepKeysSet(b *testing.B) {
	for _, c := range filterCases {
		fr, keptKeys := newFilterFrame(c.total, c.kept)
		keys := set.New(keptKeys...)
		b.Run(c.name, func(b *testing.B) {
			for b.Loop() {
				_ = fr.KeepKeysSet(keys)
			}
		})
	}
}
