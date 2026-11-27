// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	"encoding/binary"
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// randomStamp returns a stamp from 0 to 60 seconds, on a whole second or between two.
func randomStamp(rng *rand.Rand) telem.TimeStamp {
	ts := telem.TimeStamp(rng.IntN(61)) * telem.SecondTS
	if rng.IntN(3) == 0 {
		ts += telem.TimeStamp(rng.IntN(1000)) * telem.MillisecondTS
	}
	return ts
}

// randomBounds returns a range of at least one millisecond from 0 to 60 seconds.
func randomBounds(rng *rand.Rand) telem.TimeRange {
	a, b := randomStamp(rng), randomStamp(rng)
	if b < a {
		a, b = b, a
	}
	return a.Range(max(b, a+telem.MillisecondTS))
}

var _ = Describe("Seek bounds check", func() {
	It("Should keep the view and each read in the bounds and on the seek's side", func(
		ctx SpecContext,
	) {
		spans := []telem.TimeSpan{
			unary.AutoSpan,
			telem.Second,
			3 * telem.Second,
			10 * telem.Second,
		}
		for runKey := range 300 {
			rng := rand.New(rand.NewPCG(uint64(runKey), 0))
			db := MustSucceed(unary.Open(ctx, unary.Config{
				FS:        xfs.NewMem(),
				MetaCodec: json.Codec,
				Channel: channel.Channel{
					Name:     "index",
					Key:      1,
					DataType: telem.TimestampT,
					IsIndex:  true,
					Index:    1,
				},
				Instrumentation: PanicLogger(),
			}))
			// One to four domains of whole-second samples, with gaps of 0 to 5 s.
			t := telem.TimeStamp(1+rng.IntN(10)) * telem.SecondTS
			for range 1 + rng.IntN(4) {
				seconds := make([]telem.TimeStamp, 1+rng.IntN(8))
				for k := range seconds {
					seconds[k] = t/telem.SecondTS + telem.TimeStamp(k)
				}
				Expect(unary.Write(ctx, db, t, telem.NewSeriesSecondsTSV(seconds...))).
					To(Succeed(), "run key %d", runKey)
				t += telem.TimeStamp(len(seconds)+rng.IntN(6)) * telem.SecondTS
			}
			iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
				Bounds: telem.TimeRangeMax,
			}))
			iter.SetBounds(randomBounds(rng))
			var (
				seekedTo telem.TimeStamp
				seeked   bool
			)
			for step := range 30 {
				bounds := iter.Bounds()
				var op string
				switch r := rng.IntN(10); {
				case r == 0:
					op = "set bounds"
					iter.SetBounds(randomBounds(rng))
					seeked = false
					continue
				case r < 3:
					seekedTo, seeked = randomStamp(rng), true
					op = "seek LE"
					iter.SeekLE(ctx, seekedTo)
				case r < 5:
					seekedTo, seeked = randomStamp(rng), true
					op = "seek GE"
					iter.SeekGE(ctx, seekedTo)
				case r == 5:
					op = "seek first"
					iter.SeekFirst(ctx)
					seeked = false
				default:
					span := spans[rng.IntN(len(spans))]
					forward := rng.IntN(2) == 0
					if forward {
						op = "next"
						iter.Next(ctx, span)
					} else {
						op = "prev"
						iter.Prev(ctx, span)
					}
					for s := range iter.Value().Series() {
						for k := range int(s.Len()) {
							ts := telem.TimeStamp(binary.LittleEndian.Uint64(s.At(k)))
							Expect(bounds.ContainsStamp(ts)).To(BeTrue(),
								"run key %d step %d: %s read %v outside %v",
								runKey, step, op, ts, bounds,
							)
							if !seeked {
								continue
							}
							if forward {
								Expect(ts).To(BeNumerically(">=", seekedTo),
									"run key %d step %d: next read %v before seek %v",
									runKey, step, ts, seekedTo,
								)
							} else {
								Expect(ts).To(BeNumerically("<", seekedTo),
									"run key %d step %d: prev read %v after seek %v",
									runKey, step, ts, seekedTo,
								)
							}
						}
					}
					seeked = false
				}
				Expect(iter.Error()).ToNot(HaveOccurred(),
					"run key %d step %d: %s", runKey, step, op)
				view := iter.View()
				Expect(view.Start >= bounds.Start && view.End <= bounds.End).To(
					BeTrue(),
					"run key %d step %d: %s left view %v outside %v",
					runKey, step, op, view, bounds,
				)
			}
			Expect(iter.Close()).To(Succeed())
			Expect(db.Close()).To(Succeed())
		}
	})
})
