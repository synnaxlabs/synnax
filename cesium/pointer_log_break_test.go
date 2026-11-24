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
	"context"
	"math/rand/v2"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var errBrkSync = errors.New("segment sync failed")

// brkSegFS fails the Sync of every segment file while fail is set.
type brkSegFS struct {
	xfs.FS
	fail *atomic.Bool
}

func (f brkSegFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".seg") {
		return file, err
	}
	return brkSegFile{File: file, fail: f.fail}, nil
}

func (f brkSegFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return brkSegFS{FS: sub, fail: f.fail}, nil
}

type brkSegFile struct {
	xfs.File
	fail *atomic.Bool
}

func (f brkSegFile) Sync() error {
	if f.fail.Load() {
		return errBrkSync
	}
	return f.File.Sync()
}

// brkSuffixFS fails the Sync of every file whose name ends with suffix, and every
// SyncDir of a directory other than the root, while the matching flag is set.
type brkSuffixFS struct {
	xfs.FS
	suffix  string
	fail    *atomic.Bool
	dirFail *atomic.Bool
	root    bool
}

func (f brkSuffixFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, f.suffix) {
		return file, err
	}
	return brkSegFile{File: file, fail: f.fail}, nil
}

func (f brkSuffixFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return brkSuffixFS{
		FS:      sub,
		suffix:  f.suffix,
		fail:    f.fail,
		dirFail: f.dirFail,
		root:    f.root && (name == "" || name == "."),
	}, nil
}

func (f brkSuffixFS) SyncDir(name string) error {
	if f.dirFail.Load() && (!f.root || (name != "" && name != ".")) {
		return errBrkSync
	}
	return f.FS.SyncDir(name)
}

var _ = Describe("Pointer log fatal path breaker", func() {
	var (
		fail, dirFail atomic.Bool
		db            *cesium.DB
		// done closes once Run returns, and runErr then holds its error.
		done      chan struct{}
		runErr    error
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
	)
	start := func(ctx context.Context, suffix string) {
		GinkgoHelper()
		fail.Store(false)
		dirFail.Store(false)
		fs := brkSuffixFS{
			FS:      xfs.NewMem(),
			suffix:  suffix,
			fail:    &fail,
			dirFail: &dirFail,
			root:    true,
		}
		db = MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		))
		runCtx, cancel := context.WithCancel(context.Background())
		done = make(chan struct{})
		go func() {
			runErr = db.Run(runCtx)
			close(done)
		}()
		DeferCleanup(func() {
			cancel()
			Eventually(done).WithTimeout(5 * time.Second).Should(BeClosed())
			// Close returns the failure that ended Run, if any.
			Expect(db.Close()).To(Or(Succeed(), MatchError(errBrkSync)))
		})
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		keys = []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
	}
	// syncOnce writes, commits, and syncs one sample, and returns the first error.
	syncOnce := func(ctx context.Context) (err error) {
		GinkgoHelper()
		w, err := db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    10 * telem.SecondTS,
		})
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, w.Close()) }()
		if _, _, err = w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(10),
			telem.NewSeriesV[int64](10),
		}), nil); err != nil {
			return err
		}
		if _, _, err = w.Commit(ctx, nil); err != nil {
			return err
		}
		return w.Sync()
	}

	It("Should end DB.Run after a failed sync of the log", func(ctx SpecContext) {
		start(ctx, ".log")
		fail.Store(true)
		Expect(syncOnce(ctx)).To(MatchError(errBrkSync))
		Eventually(done).WithTimeout(5 * time.Second).Should(BeClosed())
		Expect(runErr).To(MatchError(errBrkSync))
	})

	It("Should end DB.Run after a failed sync of a new segment's directory", func(
		ctx SpecContext,
	) {
		start(ctx, ".log")
		dirFail.Store(true)
		Expect(syncOnce(ctx)).To(MatchError(errBrkSync))
		Eventually(done).WithTimeout(5 * time.Second).Should(BeClosed())
		Expect(runErr).To(MatchError(errBrkSync))
	})
})

var _ = Describe("Pointer log breaker", func() {
	var (
		fail      atomic.Bool
		fs        xfs.FS
		db        *cesium.DB
		closeDB   func()
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
	)
	// open opens and runs a DB on fs. closeDB stops it once, checks that Run returned
	// no error, and closes it. The cleanup of the spec calls it for a DB still open.
	open := func(ctx context.Context) {
		GinkgoHelper()
		db = MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		))
		opened, run, closed := db, GoRun(GinkgoT().Context(), db.Run), false
		stopAndClose := func() {
			GinkgoHelper()
			if closed {
				return
			}
			closed = true
			Expect(run()).To(Succeed())
			Expect(opened.Close()).To(Succeed())
		}
		closeDB = stopAndClose
		DeferCleanup(stopAndClose)
	}
	BeforeEach(func(ctx SpecContext) {
		fail.Store(false)
		fs = brkSegFS{FS: xfs.NewMem(), fail: &fail}
		open(ctx)
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		keys = []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
	})
	frame := func(seconds ...telem.TimeStamp) cesium.Frame {
		values := make([]int64, len(seconds))
		for i, s := range seconds {
			values[i] = int64(s)
		}
		return telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(seconds...),
			telem.NewSeries(values),
		})
	}
	read := func(ctx context.Context) telem.MultiSeries {
		GinkgoHelper()
		return MustSucceed(db.Read(ctx, telem.TimeRangeMax, data)).Get(data)
	}
	write := func(ctx context.Context, w *cesium.Writer, s ...telem.TimeStamp) {
		GinkgoHelper()
		MustSucceed2(w.Write(ctx, frame(s...), nil))
		MustSucceed2(w.Commit(ctx, nil))
	}

	Describe("A clock write whose segment sync fails", func() {
		It("Should keep the first commit of a writer out of the state after a reopen",
			func(ctx SpecContext) {
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    10 * telem.SecondTS,
				}))
				write(ctx, w, 10, 11)
				fail.Store(true)
				Expect(w.Sync()).To(MatchError(errBrkSync))
				fail.Store(false)
				Expect(w.Close()).To(MatchError(errBrkSync))
				Expect(read(ctx).Series).To(BeEmpty())
				closeDB()
				open(ctx)
				Expect(read(ctx).Series).To(BeEmpty())
			})

		It("Should let the database go on and reopen after a later commit fails",
			func(ctx SpecContext) {
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    10 * telem.SecondTS,
				}))
				write(ctx, w, 10, 11)
				Expect(w.Sync()).To(Succeed())
				write(ctx, w, 12)
				fail.Store(true)
				Expect(w.Sync()).To(MatchError(errBrkSync))
				fail.Store(false)
				Expect(w.Close()).To(MatchError(errBrkSync))
				next := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    20 * telem.SecondTS,
				}))
				write(ctx, next, 20)
				Expect(next.Close()).To(Succeed())
				closeDB()
				open(ctx)
				values := read(ctx)
				Expect(values.Series).To(HaveLen(2))
				Expect(values.Series[0]).To(telem.MatchSeriesDataV[int64](10, 11))
				Expect(values.Series[1]).To(telem.MatchSeriesDataV[int64](20))
			})

		It("Should reopen after a later commit of a writer fails its segment sync",
			func(ctx SpecContext) {
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    10 * telem.SecondTS,
				}))
				write(ctx, w, 10, 11)
				Expect(w.Sync()).To(Succeed())
				write(ctx, w, 12)
				fail.Store(true)
				Expect(w.Sync()).To(MatchError(errBrkSync))
				fail.Store(false)
				Expect(w.Close()).To(MatchError(errBrkSync))
				closeDB()
				open(ctx)
				Expect(read(ctx).Series[0]).To(telem.MatchSeriesDataV[int64](10, 11))
			})
	})

	Describe("Drop", func() {
		It("Should drop an index and its data channel in one call, in either order",
			func(ctx SpecContext) {
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    10 * telem.SecondTS,
				}))
				write(ctx, w, 10, 11)
				Expect(w.Close()).To(Succeed())
				Expect(db.Drop(ctx, []cesium.ChannelKey{idx, data})).To(Succeed())
				Expect(db.RetrieveChannel(ctx, idx)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
				closeDB()
				open(ctx)
				Expect(db.RetrieveChannel(ctx, data)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
			})

		It("Should reopen after a drop of an index and its data channel in one call",
			func(ctx SpecContext) {
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    10 * telem.SecondTS,
				}))
				write(ctx, w, 10, 11)
				Expect(w.Close()).To(Succeed())
				Expect(db.Drop(ctx, []cesium.ChannelKey{idx, data})).To(Succeed())
				closeDB()
				open(ctx)
				Expect(db.RetrieveChannel(ctx, data)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
			})

		It("Should sweep an index and its data channel that both hold an intent",
			func(ctx SpecContext) {
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    10 * telem.SecondTS,
				}))
				write(ctx, w, 10, 11)
				Expect(w.Close()).To(Succeed())
				Expect(db.Intend(keys)).To(Succeed())
				closeDB()
				open(ctx)
				kept, _ := MustSucceed2(db.Sweep(ctx, nil))
				Expect(kept).To(BeEmpty())
				closeDB()
				open(ctx)
				Expect(db.RetrieveChannel(ctx, idx)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
			})
	})

	Describe("A writer that stores nothing", func() {
		It("Should stamp its series with an alignment number of unstored data",
			func(ctx SpecContext) {
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    10 * telem.SecondTS,
					Mode:     cesium.WriterModeStreamOnly,
				}))
				written, _ := MustSucceed2(w.Write(ctx, frame(10, 11), nil))
				Expect(written.Get(data).Series[0].Alignment.DomainIndex()).To(
					BeNumerically(">=", telem.LeadingDomain),
				)
				Expect(w.Close()).To(Succeed())
			})
	})

	Describe("A writer with many commits before a Sync", func() {
		It("Should sync its commits on a log with room", func(ctx SpecContext) {
			wide := make([]cesium.ChannelKey, 0, 101)
			wide = append(wide, idx)
			for range 100 {
				k := GenerateChannelKey()
				Expect(db.CreateChannel(ctx, cesium.Channel{
					Key: k, Name: "wide", Index: idx, DataType: telem.Uint8T,
				})).To(Succeed())
				wide = append(wide, k)
			}
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: wide,
				Start:    1 * telem.SecondTS,
			}))
			series := make([]telem.Series, len(wide))
			for i := range 100 {
				series[i+1] = telem.NewSeriesV[uint8](1)
			}
			// 11,000 commits of 101 slots hold more than 16 MiB of records.
			for i := range 11_000 {
				series[0] = telem.NewSeries([]telem.TimeStamp{
					telem.TimeStamp(i+1) * telem.SecondTS,
				})
				MustSucceed2(w.Write(ctx, telem.MultiFrame(wide, series), nil))
				MustSucceed2(w.Commit(ctx, nil))
			}
			Expect(w.Sync()).To(Succeed())
			Expect(w.Close()).To(Succeed())
		})

		It("Should switch logs and reopen after a writer whose last Sync failed",
			func(ctx SpecContext) {
				wide := make([]cesium.ChannelKey, 0, 101)
				wide = append(wide, idx)
				for range 100 {
					k := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key: k, Name: "wide", Index: idx, DataType: telem.Uint8T,
					})).To(Succeed())
					wide = append(wide, k)
				}
				series := make([]telem.Series, len(wide))
				for i := range 100 {
					series[i+1] = telem.NewSeriesV[uint8](1)
				}
				second := telem.TimeStamp(0)
				commits := func(w *cesium.Writer, n int) {
					GinkgoHelper()
					for range n {
						second++
						series[0] = telem.NewSeries([]telem.TimeStamp{
							second * telem.SecondTS,
						})
						MustSucceed2(w.Write(ctx, telem.MultiFrame(wide, series), nil))
						MustSucceed2(w.Commit(ctx, nil))
					}
				}
				failed := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: wide,
					Start:    1 * telem.SecondTS,
				}))
				commits(failed, 1)
				Expect(failed.Sync()).To(Succeed())
				commits(failed, 10_500)
				fail.Store(true)
				Expect(failed.Sync()).To(MatchError(errBrkSync))
				fail.Store(false)
				Expect(failed.Close()).To(MatchError(errBrkSync))
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: wide,
					Start:    (second + 1) * telem.SecondTS,
				}))
				// Four entries of about 15 MB pass three quarters of log 1.
				for range 5 {
					commits(w, 9_000)
					Expect(w.Sync()).To(Succeed())
				}
				Expect(w.Close()).To(Succeed())
				closeDB()
				open(ctx)
				Expect(read(ctx).Series).To(BeEmpty())
			})

		It("Should reopen after a switch that follows a writer whose last Sync failed",
			func(ctx SpecContext) {
				wide := make([]cesium.ChannelKey, 0, 101)
				wide = append(wide, idx)
				for range 100 {
					k := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key: k, Name: "wide", Index: idx, DataType: telem.Uint8T,
					})).To(Succeed())
					wide = append(wide, k)
				}
				series := make([]telem.Series, len(wide))
				for i := range 100 {
					series[i+1] = telem.NewSeriesV[uint8](1)
				}
				second := telem.TimeStamp(0)
				commits := func(w *cesium.Writer, n int) error {
					for range n {
						second++
						series[0] = telem.NewSeries([]telem.TimeStamp{
							second * telem.SecondTS,
						})
						if _, _, err := w.Write(
							ctx,
							telem.MultiFrame(wide, series),
							nil,
						); err != nil {
							return err
						}
						if _, _, err := w.Commit(ctx, nil); err != nil {
							return err
						}
					}
					return w.Sync()
				}
				failed := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: wide,
					Start:    1 * telem.SecondTS,
				}))
				Expect(commits(failed, 1)).To(Succeed())
				fail.Store(true)
				Expect(commits(failed, 10_500)).To(MatchError(errBrkSync))
				fail.Store(false)
				Expect(failed.Close()).To(MatchError(errBrkSync))
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: wide,
					Start:    (second + 1) * telem.SecondTS,
				}))
				for range 5 {
					Expect(commits(w, 9_000)).To(Succeed())
				}
				Expect(w.Close()).To(Succeed())
				closeDB()
				open(ctx)
				Expect(read(ctx).Series).To(BeEmpty())
			})
	})
})
