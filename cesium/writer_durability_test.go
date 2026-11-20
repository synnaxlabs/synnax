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
	"fmt"
	"math/rand/v2"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/domain"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer errors", func() {
	var (
		db               *cesium.DB
		idx1, d1         cesium.ChannelKey
		idx2, d2         cesium.ChannelKey
		createTwoIndexes func(ctx SpecContext)
	)
	BeforeEach(func(ctx SpecContext) {
		db = mustOpenDBOnFS(ctx, xfs.NewMem())
		idx1, d1 = GenerateChannelKey(), GenerateChannelKey()
		idx2, d2 = GenerateChannelKey(), GenerateChannelKey()
		createTwoIndexes = func(ctx SpecContext) {
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx1, Name: "i1", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: d1, Name: "d1", Index: idx1, DataType: telem.Int64T,
				},
				cesium.Channel{
					Key: idx2, Name: "i2", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: d2, Name: "d2", Index: idx2, DataType: telem.Int64T,
				},
			)).To(Succeed())
		}
	})

	It(
		"Should return the commit error of DB.Write when another writer controls part of the frame",
		func(ctx SpecContext) {
			createTwoIndexes(ctx)
			Expect(db.Write(ctx, 12*telem.SecondTS, telem.MultiFrame(
				[]cesium.ChannelKey{idx2, d2},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(12, 20),
					telem.NewSeriesV[int64](1, 2),
				},
			))).To(Succeed())
			MustOpen(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{idx1, d1},
				Start:    5 * telem.SecondTS,
			}))
			// Group 2 overlaps its stored data, so its commit fails.
			err := db.Write(ctx, 5*telem.SecondTS, telem.MultiFrame(
				[]cesium.ChannelKey{idx1, d1, idx2, d2},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(5, 6),
					telem.NewSeriesV[int64](3, 4),
					telem.NewSeriesSecondsTSV(5, 15),
					telem.NewSeriesV[int64](3, 4),
				},
			))
			Expect(err).To(MatchError(domain.ErrWriteConflict))
		},
	)

	It(
		"Should join a persist error of Close with the error that broke the writer",
		func(ctx SpecContext) {
			var (
				errPersist  = errors.New("persist failed")
				failPersist bool
			)
			hooked := mustOpenDBOnFS(ctx, persistHookFS{
				FS: xfs.NewMem(),
				onPersist: func() error {
					if failPersist {
						return errPersist
					}
					return nil
				},
			})
			Expect(hooked.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx1, Name: "i1", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: d1, Name: "d1", Index: idx1, DataType: telem.Int64T,
				},
			)).To(Succeed())
			keys := []cesium.ChannelKey{idx1, d1}
			w := MustSucceed(hooked.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Start:    10 * telem.SecondTS,
			}))
			MustSucceed(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(10, 11),
				telem.NewSeriesV[int64](1, 2),
			})))
			MustSucceed(w.Commit(ctx))
			// A frame with uneven series breaks the writer.
			Expect(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(12, 13),
				telem.NewSeriesV[int64](3),
			}))).Error().To(MatchError(validate.ErrValidation))
			failPersist = true
			err := w.Close()
			failPersist = false
			Expect(err).To(MatchError(validate.ErrValidation))
			// The commit at 10s is not durable, so Close must say so.
			Expect(err).To(MatchError(errPersist))
		},
	)
})

var _ = Describe("Writer sync durability", func() {
	// snapshot opens a DB on a copy of mem, as a restart after a crash would see it,
	// and returns what it reads for keys.
	snapshot := func(
		ctx SpecContext,
		mem xfs.FS,
		keys []cesium.ChannelKey,
	) map[cesium.ChannelKey]telem.MultiSeries {
		GinkgoHelper()
		cp := xfs.NewMem()
		Expect(xfs.CopyDir(mem, cp, "")).To(Succeed())
		probe := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(cp)))
		defer func() { Expect(probe.Close()).To(Succeed()) }()
		out := make(map[cesium.ChannelKey]telem.MultiSeries, len(keys))
		for _, k := range keys {
			out[k] = MustSucceed(probe.Read(ctx, telem.TimeRangeMax, k)).Get(k)
		}
		return out
	}
	for seed := uint64(1); seed <= 32; seed++ {
		It(fmt.Sprintf(
			"Should make every commit durable at Sync and Close, seed %d",
			seed,
		), func(ctx SpecContext) {
			rng := rand.New(rand.NewPCG(seed, 0x5EED))
			mem := xfs.NewMem()
			db := mustOpenDBOnFS(ctx, mem, cesium.WithFileSizeCap(96*telem.Byte))
			var (
				i1, d1 = GenerateChannelKey(), GenerateChannelKey()
				i2, d2 = GenerateChannelKey(), GenerateChannelKey()
				keys   = []cesium.ChannelKey{i1, d1, i2, d2}
				next   = map[cesium.ChannelKey]telem.TimeStamp{
					i1: telem.SecondTS,
					i2: telem.SecondTS,
				}
				log []string
				// pending and committed count the samples of each index.
				pending   = map[cesium.ChannelKey]int64{}
				committed = map[cesium.ChannelKey]int64{}
			)
			commitAll := func() {
				for k, n := range pending {
					committed[k] += n
					pending[k] = 0
				}
			}
			expectVisible := func(when string) {
				GinkgoHelper()
				for idx, data := range map[cesium.ChannelKey]cesium.ChannelKey{
					i1: d1,
					i2: d2,
				} {
					for _, k := range []cesium.ChannelKey{idx, data} {
						Expect(
							MustSucceed(
								db.Read(ctx, telem.TimeRangeMax, k),
							).Get(k).
								Len(),
						).To(Equal(committed[idx]), "%s: channel %d after %v", when, k, log)
					}
				}
			}
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: i1, Name: "i1", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{Key: d1, Name: "d1", Index: i1, DataType: telem.Int64T},
				cesium.Channel{
					Key: i2, Name: "i2", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{Key: d2, Name: "d2", Index: i2, DataType: telem.StringT},
			)).To(Succeed())
			open := func() *cesium.Writer {
				start := max(next[i1], next[i2])
				next[i1], next[i2] = start, start
				return MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    start,
				}))
			}
			expectDurable := func(when string, of ...cesium.ChannelKey) {
				GinkgoHelper()
				if len(of) == 0 {
					of = keys
				}
				snap := snapshot(ctx, mem, keys)
				for _, k := range of {
					live := MustSucceed(db.Read(ctx, telem.TimeRangeMax, k)).Get(k)
					Expect(snap[k].Len()).To(
						Equal(live.Len()),
						"%s: channel %d after %v",
						when,
						k,
						log,
					)
					Expect(snap[k].Data()).To(Equal(live.Data()))
				}
			}
			series := func(idx cesium.ChannelKey, n int) []telem.Series {
				pending[idx] += int64(n)
				stamps := make([]telem.TimeStamp, n)
				for j := range stamps {
					stamps[j] = next[idx]
					next[idx] += telem.SecondTS
				}
				if idx == i1 {
					data := make([]int64, n)
					for j := range data {
						data[j] = rng.Int64()
					}
					return []telem.Series{
						telem.NewSeries(stamps),
						telem.NewSeries(data),
					}
				}
				data := make([]string, n)
				for j := range data {
					data[j] = strings.Repeat("x", 1+rng.IntN(9))
				}
				return []telem.Series{telem.NewSeries(stamps), telem.NewSeries(data)}
			}
			w := open()
			for range 80 {
				switch op := rng.IntN(100); {
				case op < 50:
					var (
						fKeys   []cesium.ChannelKey
						fSeries []telem.Series
						groups  = rng.IntN(3)
						n       = 1 + rng.IntN(4)
					)
					if groups != 2 {
						fKeys = append(fKeys, i1, d1)
						fSeries = append(fSeries, series(i1, n)...)
					}
					if groups != 1 {
						fKeys = append(fKeys, i2, d2)
						fSeries = append(fSeries, series(i2, n)...)
					}
					log = append(log, fmt.Sprintf("write(%d,%d)", groups, n))
					MustSucceed(w.Write(ctx, telem.MultiFrame(fKeys, fSeries)))
					expectVisible("write")
				case op < 70:
					log = append(log, "commit")
					MustSucceed(w.Commit(ctx))
					commitAll()
					expectVisible("commit")
				case op < 82:
					log = append(log, "sync")
					Expect(w.Sync()).To(Succeed())
					expectDurable("sync")
				case op < 92:
					idx := []cesium.ChannelKey{i1, i2}[rng.IntN(2)]
					log = append(log, fmt.Sprintf("end(%d)", idx))
					Expect(w.EndDomain(ctx, idx)).To(Succeed())
					// EndDomain commits only the groups of the indexes it names.
					committed[idx] += pending[idx]
					pending[idx] = 0
					expectVisible("end domain")
				case op < 96:
					log = append(log, "handoff")
					MustSucceed(w.Commit(ctx))
					commitAll()
					high := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{i1, d1},
						Start:    next[i1],
					}))
					n := 1 + rng.IntN(4)
					MustSucceed(high.Write(ctx, telem.MultiFrame(
						[]cesium.ChannelKey{i1, d1},
						series(i1, n),
					)))
					MustSucceed(high.Commit(ctx))
					commitAll()
					expectVisible("handoff commit")
					if rng.IntN(2) == 0 {
						Expect(w.Sync()).To(Succeed())
						expectDurable("sync during handoff")
					}
					Expect(high.Close()).To(Succeed())
					// The index persists for the whole channel, so the Close of high
					// also makes the earlier commits of w on i1 durable.
					expectDurable("handoff close", i1, d1)
				default:
					log = append(log, "close")
					Expect(w.Close()).To(Succeed())
					pending = map[cesium.ChannelKey]int64{}
					expectVisible("close")
					expectDurable("close")
					w = open()
				}
			}
			Expect(w.Close()).To(Succeed())
			expectDurable("final close")
		})
	}
})

var _ = Describe("Writer persist faults", func() {
	It("Should persist at Close what a failed Sync left out", func(ctx SpecContext) {
		var (
			errPersist = errors.New("persist failed")
			fails      int
			mem        = xfs.NewMem()
			i, d       = GenerateChannelKey(), GenerateChannelKey()
			keys       = []cesium.ChannelKey{i, d}
		)
		db := mustOpenDBOnFS(ctx, persistHookFS{
			FS: mem,
			onPersist: func() error {
				if fails > 0 {
					fails--
					return errPersist
				}
				return nil
			},
		})
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      i,
				Name:     "i",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{Key: d, Name: "d", Index: i, DataType: telem.Int64T},
		)).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    10 * telem.SecondTS,
		}))
		MustSucceed(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(10, 11),
			telem.NewSeriesV[int64](1, 2),
		})))
		MustSucceed(w.Commit(ctx))
		fails = 1
		Expect(w.Sync()).To(MatchError(errPersist))
		Expect(w.Close()).To(MatchError(errPersist))
		cp := xfs.NewMem()
		Expect(xfs.CopyDir(mem, cp, "")).To(Succeed())
		probe := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(cp)))
		for _, k := range keys {
			Expect(MustSucceed(probe.Read(ctx, telem.TimeRangeMax, k)).Get(k).Len()).
				To(Equal(int64(2)))
		}
		Expect(probe.Close()).To(Succeed())
	})
})
