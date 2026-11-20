// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"context"
	"fmt"
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Round 2 check", func() {
	DescribeTable(
		"Should refuse a header copy that passes the check of the caller's log but "+
			"holds another database or generation",
		func(change func(*log.Header)) {
			h := header
			change(&h)
			b := sealedHeader(h.Encode(0))
			Expect(log.DecodeHeader(b, db, generation, 0)).Error().
				To(MatchError(place.ErrCorrupt))
		},
		Entry("another database", func(h *log.Header) { h.DB = 1 }),
		Entry("another generation", func(h *log.Header) { h.Generation = 4 }),
	)
})

// entryCount returns the number of entries in the frames of the log file in fs.
func entryCount(fs xfs.FS) int {
	GinkgoHelper()
	n := 0
	for _, f := range walk(fileBytes(fs)) {
		n += int(f.Count)
	}
	return n
}

var _ = Describe("Round 2 check, random entries", func() {
	for k := range uint64(32) {
		It(fmt.Sprintf("Should fill a log on a full disk with no growth, and keep "+
			"every acked entry through a crash, run key %d", k), func() {
			fs := MustSucceed(fault.New(fault.Config{
				RunKey:        k,
				Capacity:      logSize,
				SyncErrorRate: []float64{0, 0.03}[k%2],
			}))
			// A run key whose sync error lands in the setup tests nothing.
			if log.Prepare(fs, generation, logSize) != nil {
				return
			}
			l, err := log.Start(log.Config{
				FS:         fs,
				Executor:   goExec,
				DB:         db,
				Generation: generation,
				Sequence:   1,
			})
			if err != nil {
				return
			}
			ran := make(chan error, 1)
			go func() { ran <- l.Run(context.Background()) }()
			rng := rand.New(rand.NewPCG(k, k^0x9E3779B97F4A7C15))
			var (
				acked, full int
				failed      error
				acks        []log.Ack
			)
			wait := func() {
				for _, a := range acks {
					if _, err := a.Wait(); err == nil {
						acked++
					} else if errors.Is(err, log.ErrFull) {
						full++
					} else {
						failed = err
					}
				}
				acks = acks[:0]
			}
			for i := range 400 {
				n := 48 + 8*rng.IntN(64)
				if rng.IntN(3) == 0 {
					n = 48 + 8*rng.IntN(32_000)
				}
				e := bigEntry(n)
				if rng.IntN(2) == 0 {
					e = commitEntry(n)
				}
				acks = append(acks, l.Append(e))
				if rng.IntN(3) == 0 {
					wait()
				}
				if i%50 == 49 {
					wait()
					img := MustSucceed(fs.Crash(fault.CrashConfig{
						RunKey:           k + uint64(i),
						DataSurvivalRate: 0.5,
						TornRate:         0.5,
					}))
					Expect(entryCount(img)).To(BeNumerically(">=", acked))
				}
			}
			wait()
			err = l.Close()
			Eventually(ran).Should(Receive(BeNil()))
			if failed == nil {
				Expect(err).ToNot(HaveOccurred())
				Expect(full).To(BeNumerically(">", 100), "the log fills")
				frames := walk(fileBytes(fs))
				Expect(frames[len(frames)-1].Clean).To(BeTrue())
			} else {
				Expect(err).To(MatchError(failed))
			}
			Expect(MustSucceed(fs.Stat(logName)).Size()).To(Equal(int64(logSize)))
			Expect(entryCount(fs)).To(BeNumerically(">=", acked))
		})
	}
})
