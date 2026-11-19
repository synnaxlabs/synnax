// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// breakPayload returns 1,400 bytes for the record id, so its clock fills 11,200 bytes:
// three blocks of 4 KiB.
func breakPayload(id uint64) []byte {
	rng := rand.New(rand.NewPCG(id, 7))
	out := make([]byte, 1400)
	for i := range out {
		out[i] = byte(rng.Uint32())
	}
	return out
}

// breakClock returns the bytes that Commit writes on the clock for n samples of id.
func breakClock(id uint64, n int) []byte {
	start := telem.TimeStamp(id) * telem.SecondTS
	stamps := make([]telem.TimeStamp, n)
	for i := range stamps {
		stamps[i] = start + telem.TimeStamp(i)
	}
	return telem.NewSeries(stamps).Data
}

// breakClocks reads the clock of db and groups it by second, as Recovered does.
func breakClocks(ctx context.Context, db *cesium.DB) map[uint64][]byte {
	frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, crashClock))
	out := map[uint64][]byte{}
	for _, s := range frame.SeriesSlice() {
		id := uint64(s.TimeRange.Start / telem.SecondTS)
		out[id] = append(out[id], s.Data...)
	}
	return out
}

// breakBase returns a disk on which the three channels of the adapter and their meta
// files are durable, and an engine open on it that committed records 1 to commits.
func breakBase(ctx context.Context, commits uint64) (*fault.FS, crash.Engine) {
	open := openCrashEngine(crash.Visible)
	disk := MustSucceed(fault.New(fault.Config{}))
	Expect(MustSucceed(open(ctx, disk)).Close()).To(Succeed())
	base := MustSucceed(disk.Crash(fault.CrashConfig{
		DataSurvivalRate:  1,
		EntrySurvivalRate: 1,
	}))
	e := MustSucceed(open(ctx, base))
	for id := uint64(1); id <= commits; id++ {
		ack := MustSucceed(e.Commit(
			ctx,
			crash.Record{ID: id, Payload: breakPayload(id)},
			crash.Visible,
		))
		Expect(ack).To(Equal(crash.Visible))
	}
	return base, e
}

var _ = Describe("Crash adapter break", func() {
	It("gives no payload for a commit whose clock a crash left as zeros",
		func(ctx SpecContext) {
			base, live := breakBase(ctx, 4)
			open := openCrashEngine(crash.Visible)
			var (
				hidden []string
				rates  = []float64{0.25, 0.5, 0.75}
				images []fault.CrashConfig
			)
			// The rates that the Partial profile draws from, with 32 crash keys each.
			for _, data := range rates {
				for _, torn := range rates {
					for _, entry := range rates {
						for key := uint64(0); key < 32; key++ {
							images = append(images, fault.CrashConfig{
								RunKey:            key,
								DataSurvivalRate:  data,
								TornRate:          torn,
								EntrySurvivalRate: entry,
							})
						}
					}
				}
			}
			for _, cfg := range images {
				key := fmt.Sprintf(
					"crash key %d, rates %.2f %.2f %.2f",
					cfg.RunKey,
					cfg.DataSurvivalRate,
					cfg.TornRate,
					cfg.EntrySurvivalRate,
				)
				image := MustSucceed(base.Crash(cfg))
				e, err := open(ctx, image)
				if err != nil {
					continue
				}
				records, err := e.Recovered(ctx)
				if err == nil {
					clocks := breakClocks(ctx, e.(*crashEngine).db)
					for _, r := range records {
						if !bytes.Equal(r.Payload, breakPayload(r.ID)) {
							continue
						}
						got, want := clocks[r.ID], breakClock(r.ID, len(r.Payload))
						wrong, zero := 0, 0
						for i := 0; i+8 <= len(got) && i+8 <= len(want); i += 8 {
							if !bytes.Equal(got[i:i+8], want[i:i+8]) {
								wrong++
								if bytes.Equal(got[i:i+8], make([]byte, 8)) {
									zero++
								}
							}
						}
						if wrong > 0 || len(got) != len(want) {
							hidden = append(hidden, fmt.Sprintf(
								"%s: commit %d has its whole payload, but %d of "+
									"its %d clock samples are wrong (%d are zero)",
								key, r.ID, wrong, len(want)/8, zero,
							))
						}
					}
				}
				Expect(e.Close()).To(Succeed())
			}
			Expect(live.Close()).To(Succeed())
			Expect(hidden).To(BeEmpty())
		})

	// The table puts Mixed under "a channel directory with a torn or missing meta
	// file". A torn or missing file comes from a crash, so its recovery reads no fault.
	DescribeTable("Mixed at run key 0 breaks R4 on a torn or missing meta file",
		func(ctx SpecContext, ack crash.Ack) {
			_, err := crash.Run(ctx, crash.Config{
				Open:    openCrashEngine(ack),
				Commits: 8,
				Stride:  1,
				Profile: crash.Mixed,
			})
			var v *crash.Violation
			Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
			Expect(v.Rule).To(Equal(crash.R4))
			Expect(v.ReadFaults).To(BeZero(), v.Error())
		},
		Entry("Durable", crash.Durable),
		Entry("Visible", crash.Visible),
	)
})
