// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package time

import (
	"time"
)

// ScaledTicker is a ticker that scales the duration between ticks.
// It provides an identical interface to a time.Ticker.
type ScaledTicker struct {
	C     <-chan time.Duration
	Scale float64
	dur   time.Duration
	stop  chan struct{}
}

// NewScaledTicker returns a new ScaledTicker that ticks at the given duration and scale.
func NewScaledTicker(d time.Duration, scale float64) *ScaledTicker {
	c := make(chan time.Duration)
	t := &ScaledTicker{dur: d, Scale: scale, stop: make(chan struct{}), C: c}
	go t.tick(c)
	return t
}

// Stop stops the ticker.
func (s *ScaledTicker) Stop() { close(s.stop) }

func (s *ScaledTicker) tick(c chan time.Duration) {
	t := time.NewTicker(s.dur)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.dur = time.Duration(float64(s.dur) * s.Scale)
			select {
			case c <- s.dur:
			case <-s.stop:
				return
			}
			t.Reset(s.dur)
		}
	}
}
