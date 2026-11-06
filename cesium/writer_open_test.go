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
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/resource"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const gateTimeout = 30 * time.Second

var domainDataFile = regexp.MustCompile(`^\d+\.domain$`)

// fileGate holds opens of domain data files while it is armed.
type fileGate struct {
	armed   atomic.Bool
	entered chan struct{}
	release chan struct{}
	open    func()
}

func newFileGate() *fileGate {
	g := &fileGate{entered: make(chan struct{}, 1), release: make(chan struct{})}
	g.open = sync.OnceFunc(func() { close(g.release) })
	return g
}

// gatedFS gives each channel directory its own fileGate.
type gatedFS struct {
	fs.FS
	gate  *fileGate
	gates *sync.Map
}

func newGatedFS(base fs.FS) *gatedFS { return &gatedFS{FS: base, gates: &sync.Map{}} }

func (g *gatedFS) gateFor(key cesium.ChannelKey) *fileGate {
	gate, _ := g.gates.LoadOrStore(strconv.Itoa(int(key)), newFileGate())
	return gate.(*fileGate)
}

func (g *gatedFS) Sub(name string) (fs.FS, error) {
	sub, err := g.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	gate, _ := g.gates.LoadOrStore(name, newFileGate())
	return &gatedFS{FS: sub, gate: gate.(*fileGate), gates: g.gates}, nil
}

func (g *gatedFS) park(name string) error {
	if g.gate == nil || !g.gate.armed.Load() || !domainDataFile.MatchString(name) {
		return nil
	}
	select {
	case g.gate.entered <- struct{}{}:
	default:
	}
	select {
	case <-g.gate.release:
		return nil
	case <-time.After(gateTimeout):
		return errors.New("file gate was not released")
	}
}

func (g *gatedFS) Open(name string, flag int) (fs.File, error) {
	if err := g.park(name); err != nil {
		return nil, err
	}
	return g.FS.Open(name, flag)
}

func (g *gatedFS) Stat(name string) (fs.FileInfo, error) {
	if err := g.park(name); err != nil {
		return nil, err
	}
	return g.FS.Stat(name)
}

var _ = Describe("Writer open", func() {
	Describe("Blocked in the file system", func() {
		const stallTimeout = 5 * time.Second
		var (
			db     *cesium.DB
			gate   *fileGate
			idxKey cesium.ChannelKey
			opened chan error
		)

		BeforeEach(func(ctx SpecContext) {
			gfs := newGatedFS(FileSystems["memFS"]())
			db = mustOpenDBOnFS(ctx, gfs)
			idxKey = GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      idxKey,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			})).To(Succeed())
			gate = gfs.gateFor(idxKey)
			gate.armed.Store(true)
			DeferCleanup(gate.open)
			opened = make(chan error, 1)
			go func() {
				w, err := db.OpenWriter(context.Background(), cesium.WriterConfig{
					Channels:       []cesium.ChannelKey{idxKey},
					Start:          telem.SecondTS,
					Mode:           cesium.WriterModePersistOnly,
					ControlSubject: control.Subject{Key: "blocked"},
				})
				if err == nil {
					err = w.Close()
				}
				opened <- err
			}()
			Eventually(gate.entered).Should(Receive())
			DeferCleanup(func() {
				gate.open()
				Eventually(opened).Should(Receive(Succeed()))
			})
		})

		It("Should let a channel be created", func(ctx SpecContext) {
			created := make(chan error, 1)
			go func() {
				created <- db.CreateChannel(ctx, cesium.Channel{
					Key:      GenerateChannelKey(),
					Name:     "created",
					IsIndex:  true,
					DataType: telem.TimestampT,
				})
			}()
			Eventually(created, stallTimeout).Should(Receive(Succeed()))
		})

		It("Should make a time range delete wait with no lock held", func(
			ctx SpecContext,
		) {
			deleted := make(chan error, 1)
			go func() {
				deleted <- db.DeleteTimeRange(
					ctx,
					[]cesium.ChannelKey{idxKey},
					telem.TimeRangeMax,
				)
			}()
			Consistently(deleted, 200*time.Millisecond).ShouldNot(Receive())
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      GenerateChannelKey(),
				Name:     "created",
				IsIndex:  true,
				DataType: telem.TimestampT,
			})).To(Succeed())
			gate.open()
			Eventually(deleted, stallTimeout).Should(Receive(SatisfyAny(
				Succeed(),
				MatchError(control.ErrUnauthorized),
			)))
		})

		It("Should report control states", func() {
			states := make(chan cesium.ControlUpdate, 1)
			go func() { states <- db.ControlStates() }()
			Eventually(states, stallTimeout).Should(Receive())
		})

		It("Should fail to delete the channel", func() {
			deleted := make(chan error, 1)
			go func() { deleted <- db.DeleteChannel(idxKey) }()
			Eventually(deleted, stallTimeout).Should(
				Receive(MatchError(resource.ErrOpen)),
			)
		})
	})

	Describe("During close", func() {
		It("Should make Close wait for the open", func(ctx SpecContext) {
			var (
				gfs = newGatedFS(FileSystems["memFS"]())
				db  = openDBOnFS(ctx, gfs)
				key = GenerateChannelKey()
			)
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      key,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			})).To(Succeed())
			gate := gfs.gateFor(key)
			gate.armed.Store(true)
			DeferCleanup(gate.open)
			opened := make(chan *cesium.Writer, 1)
			go func() {
				defer GinkgoRecover()
				opened <- MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{key},
					Start:    telem.SecondTS,
					Mode:     cesium.WriterModePersistOnly,
				}))
			}()
			Eventually(gate.entered).Should(Receive())
			closed := make(chan error, 1)
			go func() { closed <- db.Close() }()
			Consistently(closed, 200*time.Millisecond).ShouldNot(Receive())
			gate.open()
			var w *cesium.Writer
			Eventually(opened).Should(Receive(&w))
			Eventually(closed).Should(Receive(MatchError(resource.ErrOpen)))
			Expect(w.Close()).To(Succeed())
		})

		It("Should fail an open that starts after Close", func(ctx SpecContext) {
			db := openDBOnFS(ctx, FileSystems["memFS"]())
			Expect(db.Close()).To(Succeed())
			Expect(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{GenerateChannelKey()},
				Start:    telem.SecondTS,
			})).Error().To(MatchError(cesium.ErrDBClosed))
		})
	})

	Describe("Next to a delete", func() {
		const stallTimeout = 10 * time.Second
		for _, auth := range []control.Authority{1, control.AuthorityAbsolute} {
			for _, concurrency := range []control.Concurrency{
				control.ConcurrencyExclusive,
				control.ConcurrencyShared,
			} {
				Context(
					fmt.Sprintf("Authority %d, concurrency %d", auth, concurrency),
					func() {
						var (
							db      *cesium.DB
							gfs     *gatedFS
							passKey cesium.ChannelKey
							delKey  cesium.ChannelKey
							deleted chan error
							// The delete removes [12 s, 15 s) from a domain of ten
							// samples. It parks on the read that resolves its offsets.
							tr = (12 * telem.SecondTS).Range(15 * telem.SecondTS)
						)
						stamp := func(s telem.TimeStamp) telem.TimeStamp {
							return s * telem.SecondTS
						}
						writerConfig := func() cesium.WriterConfig {
							return cesium.WriterConfig{
								Channels:    []cesium.ChannelKey{passKey, delKey},
								Start:       tr.Start,
								Authorities: []control.Authority{auth},
							}
						}
						writeThroughClose := func(w *cesium.Writer) {
							GinkgoHelper()
							MustSucceed(w.Write(telem.MultiFrame(
								[]cesium.ChannelKey{passKey, delKey},
								[]telem.Series{
									telem.NewSeriesV(tr.Start),
									telem.NewSeriesV(tr.Start),
								},
							)))
							MustSucceed(w.Commit())
							Expect(w.Close()).To(Succeed())
						}
						parkDelete := func(ctx context.Context) *fileGate {
							GinkgoHelper()
							gate := gfs.gateFor(delKey)
							gate.armed.Store(true)
							DeferCleanup(gate.open)
							deleted = make(chan error, 1)
							go func() {
								deleted <- db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{delKey},
									tr,
								)
							}()
							Eventually(gate.entered, stallTimeout).Should(Receive())
							return gate
						}
						expectStoredWrite := func(ctx context.Context) {
							GinkgoHelper()
							frame := MustSucceed(db.Read(ctx, tr, delKey))
							Expect(frame.SeriesAt(0).Data).To(Equal(
								telem.NewSeriesV(tr.Start).Data,
							))
						}

						BeforeEach(func(ctx SpecContext) {
							gfs = newGatedFS(FileSystems["memFS"]())
							db = mustOpenDBOnFS(ctx, gfs)
							passKey, delKey = GenerateChannelKey(), GenerateChannelKey()
							for _, key := range []cesium.ChannelKey{passKey, delKey} {
								Expect(db.CreateChannel(ctx, cesium.Channel{
									Key:         key,
									Name:        "time",
									IsIndex:     true,
									DataType:    telem.TimestampT,
									Concurrency: concurrency,
								})).To(Succeed())
							}
							samples := make([]telem.TimeStamp, 0, 10)
							for s := telem.TimeStamp(10); s < 20; s++ {
								samples = append(samples, stamp(s))
							}
							Expect(db.WriteSeries(
								ctx,
								delKey,
								samples[0],
								telem.NewSeries(samples),
							)).To(Succeed())
						})

						It(
							"Should make a delete wait for a writer that is opening",
							func(
								ctx SpecContext,
							) {
								passGate := gfs.gateFor(passKey)
								passGate.armed.Store(true)
								DeferCleanup(passGate.open)
								cfg := writerConfig()
								cfg.Start = stamp(5)
								opened := make(chan *cesium.Writer, 1)
								go func() {
									defer GinkgoRecover()
									opened <- MustSucceed(db.OpenWriter(ctx, cfg))
								}()
								Eventually(
									passGate.entered,
									stallTimeout,
								).Should(Receive())
								deleted := make(chan error, 1)
								go func() {
									deleted <- db.DeleteTimeRange(
										ctx,
										[]cesium.ChannelKey{delKey},
										tr,
									)
								}()
								Consistently(
									deleted,
									200*time.Millisecond,
								).ShouldNot(Receive())
								passGate.open()
								var w *cesium.Writer
								Eventually(opened, stallTimeout).Should(Receive(&w))
								Eventually(deleted, stallTimeout).Should(
									Receive(MatchError(control.ErrUnauthorized)),
								)
								Expect(w.Close()).To(Succeed())
							},
						)

						It("Should open a writer after a parked delete", func(
							ctx SpecContext,
						) {
							delGate := parkDelete(ctx)
							opened := make(chan *cesium.Writer, 1)
							go func() {
								defer GinkgoRecover()
								opened <- MustSucceed(db.OpenWriter(ctx, writerConfig()))
							}()
							Consistently(
								opened,
								200*time.Millisecond,
							).ShouldNot(Receive())
							delGate.open()
							Eventually(deleted, stallTimeout).Should(Receive(Succeed()))
							var w *cesium.Writer
							Eventually(opened, stallTimeout).Should(Receive(&w))
							writeThroughClose(w)
							expectStoredWrite(ctx)
						})

						It("Should fail a delete of a region a writer holds", func(
							ctx SpecContext,
						) {
							cfg := writerConfig()
							cfg.Start = stamp(5)
							w := MustSucceed(db.OpenWriter(ctx, cfg))
							Expect(db.DeleteTimeRange(
								ctx,
								[]cesium.ChannelKey{delKey},
								tr,
							)).To(MatchError(control.ErrUnauthorized))
							Expect(w.Close()).To(Succeed())
						})
					},
				)
			}
		}
	})

	Describe("Next to an index delete", func() {
		Context("With a dependent writer opening", func() {
			const stallTimeout = 10 * time.Second
			var (
				gfs         *gatedFS
				db          *cesium.DB
				idxKey      cesium.ChannelKey
				key, other  cesium.ChannelKey
				oldData     = (10 * telem.SecondTS).Range(20 * telem.SecondTS)
				newStart    = 1000 * telem.SecondTS
				sampleCount = func(ctx context.Context, key cesium.ChannelKey) int64 {
					GinkgoHelper()
					return MustSucceed(
						db.Read(ctx, telem.TimeRangeMax, key),
					).Get(key).
						Len()
				}
			)
			BeforeEach(func(ctx SpecContext) {
				gfs = newGatedFS(FileSystems["memFS"]())
				db = mustOpenDBOnFS(ctx, gfs)
				idxKey, key, other = GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
				Expect(db.CreateChannel(
					ctx,
					cesium.Channel{
						Key:      idxKey,
						Name:     "time",
						IsIndex:  true,
						DataType: telem.TimestampT,
					},
					cesium.Channel{
						Key:      key,
						Name:     "data",
						Index:    idxKey,
						DataType: telem.Int64T,
					},
					cesium.Channel{
						Key:      other,
						Name:     "other",
						Index:    idxKey,
						DataType: telem.Int64T,
					},
				)).To(Succeed())
				var (
					stamps []telem.TimeStamp
					values []int64
				)
				for s := range 10 {
					stamps = append(
						stamps,
						oldData.Start+telem.TimeStamp(s)*telem.SecondTS,
					)
					values = append(values, int64(s))
				}
				start := oldData.Start
				Expect(
					db.WriteSeries(ctx, idxKey, start, telem.NewSeries(stamps)),
				).To(Succeed())
				Expect(
					db.WriteSeries(ctx, key, start, telem.NewSeries(values)),
				).To(Succeed())
				Expect(
					db.WriteSeries(ctx, other, start, telem.NewSeries(values)),
				).To(Succeed())
			})

			parkOpen := func(
				ctx context.Context,
				parked cesium.ChannelKey,
				channels ...cesium.ChannelKey,
			) (*fileGate, chan *cesium.Writer) {
				GinkgoHelper()
				gate := gfs.gateFor(parked)
				gate.armed.Store(true)
				DeferCleanup(gate.open)
				opened := make(chan *cesium.Writer, 1)
				go func() {
					defer GinkgoRecover()
					opened <- MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: channels,
						Start:    newStart,
					}))
				}()
				Eventually(gate.entered, stallTimeout).Should(Receive())
				gate.armed.Store(false)
				return gate, opened
			}
			deleteAsync := func(
				ctx context.Context,
				tr telem.TimeRange,
				channels ...cesium.ChannelKey,
			) chan error {
				deleted := make(chan error, 1)
				go func() { deleted <- db.DeleteTimeRange(ctx, channels, tr) }()
				return deleted
			}

			It(
				"Should delete old index data once the open ends",
				func(ctx SpecContext) {
					Expect(db.DeleteTimeRange(
						ctx,
						[]cesium.ChannelKey{key, other},
						oldData,
					)).To(Succeed())
					gate, opened := parkOpen(ctx, key, idxKey, key)
					deleted := deleteAsync(ctx, oldData, idxKey)
					Consistently(deleted, 200*time.Millisecond).ShouldNot(Receive())
					gate.open()
					var w *cesium.Writer
					Eventually(opened, stallTimeout).Should(Receive(&w))
					Eventually(deleted, stallTimeout).Should(Receive(Succeed()))
					Expect(w.Close()).To(Succeed())
					Expect(sampleCount(ctx, idxKey)).To(BeZero())
				},
			)

			It("Should delete nothing if the writer then holds the range", func(
				ctx SpecContext,
			) {
				gate, opened := parkOpen(ctx, idxKey, idxKey, key, other)
				deleted := deleteAsync(ctx, telem.TimeRangeMax, key, other, idxKey)
				Consistently(deleted, 200*time.Millisecond).ShouldNot(Receive())
				gate.open()
				var w *cesium.Writer
				Eventually(opened, stallTimeout).Should(Receive(&w))
				Eventually(deleted, stallTimeout).Should(
					Receive(MatchError(control.ErrUnauthorized)),
				)
				Expect(w.Close()).To(Succeed())
				Expect(sampleCount(ctx, idxKey)).To(Equal(int64(10)))
				Expect(sampleCount(ctx, key)).To(Equal(int64(10)))
				Expect(sampleCount(ctx, other)).To(Equal(int64(10)))
			})
		})

		It("Should not join the region HasDataFor holds", func(ctx SpecContext) {
			const rounds = 256
			var (
				db     = mustOpenDBOnFS(ctx, FileSystems["memFS"]())
				idxKey = GenerateChannelKey()
				key    = GenerateChannelKey()
				start  = 2 * telem.SecondTS
			)
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key:      idxKey,
					Name:     "time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      key,
					Name:     "data",
					Index:    idxKey,
					DataType: telem.Int64T,
				},
			)).To(Succeed())
			Expect(db.WriteSeries(
				ctx,
				idxKey,
				telem.SecondTS,
				telem.NewSeriesV(telem.SecondTS, start),
			)).To(Succeed())
			Expect(db.WriteSeries(
				ctx,
				key,
				telem.SecondTS,
				telem.NewSeriesV[int64](1),
			)).To(Succeed())
			var (
				wg   sync.WaitGroup
				done atomic.Bool
			)
			wg.Go(func() {
				defer GinkgoRecover()
				defer done.Store(true)
				for range rounds {
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{key},
						Start:    start,
					}))
					Expect(w.Close()).To(Succeed())
				}
			})
			wg.Go(func() {
				defer GinkgoRecover()
				for !done.Load() {
					Expect(db.DeleteTimeRange(
						ctx,
						[]cesium.ChannelKey{idxKey},
						telem.TimeRangeMax,
					)).To(MatchError(ContainSubstring("cannot delete index channel")))
				}
			})
			wg.Wait()
		})
	})
})
