// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fault

import "github.com/synnaxlabs/x/validate"

// The kinds of draw. Each kind gives its own number for the same path and index.
const (
	drawBlock uint64 = iota + 1
	drawTorn
	drawTornSize
	drawCut
	drawEntry
	drawSync
	drawEvict
)

// draw returns a number from 0 up to, but not including, 1. Only the arguments fix it,
// so the order of calls on other files does not change a run.
func draw(runKey uint64, parts ...uint64) float64 {
	h := runKey
	for _, part := range parts {
		h = mix(h ^ part)
	}
	return float64(mix(h)>>11) / (1 << 53)
}

// mix is the finalizer of SplitMix64.
func mix(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// hashPath is the 64-bit FNV-1a hash of p.
func hashPath(p string) uint64 {
	h := uint64(14695981039346656037)
	for i := range len(p) {
		h = (h ^ uint64(p[i])) * 1099511628211
	}
	return h
}

func validateRate(v *validate.Validator, field string, rate float64) {
	if !v.GreaterThanEq(field, rate, 0) {
		v.LessThanEq(field, rate, 1)
	}
}
