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

func (g *gatedFS) Open(name string, flag int) (fs.File, error) {
	if g.gate != nil && g.gate.armed.Load() && domainDataFile.MatchString(name) {
		select {
		case g.gate.entered <- struct{}{}:
		default:
		}
		select {
		case <-g.gate.release:
		case <-time.After(gateTimeout):
			return nil, errors.New("file gate was not released")
		}
	}
	return g.FS.Open(name, flag)
}

var _ = Describe("Writer open", func() {
	Describe("Blocked in the file system", func() {
		const stallTimeout = 5 * time.Second
		var (
			db     *cesium.DB
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
			gate := gfs.gateFor(idxKey)
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

		It("Should fail to delete a time range it covers", func(ctx SpecContext) {
			deleted := make(chan error, 1)
			go func() {
				deleted <- db.DeleteTimeRange(
					ctx,
					[]cesium.ChannelKey{idxKey},
					telem.TimeRangeMax,
				)
			}()
			Eventually(deleted, stallTimeout).Should(
				Receive(MatchError(control.ErrUnauthorized)),
			)
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
							// samples.
							// It parks on the read that resolves its offsets.
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
							"Should keep a writer that opened first out of the region",
							func(
								ctx SpecContext,
							) {
								passGate := gfs.gateFor(passKey)
								passGate.armed.Store(true)
								DeferCleanup(passGate.open)
								opened := make(chan *cesium.Writer, 1)
								go func() {
									defer GinkgoRecover()
									opened <- MustSucceed(db.OpenWriter(ctx, writerConfig()))
								}()
								Eventually(
									passGate.entered,
									stallTimeout,
								).Should(Receive())
								delGate := parkDelete(ctx)
								passGate.open()
								Consistently(
									opened,
									200*time.Millisecond,
								).ShouldNot(Receive())
								delGate.open()
								Eventually(
									deleted,
									stallTimeout,
								).Should(Receive(Succeed()))
								var w *cesium.Writer
								Eventually(opened, stallTimeout).Should(Receive(&w))
								writeThroughClose(w)
								expectStoredWrite(ctx)
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
		It("Should not join the region HasDataFor holds", func(ctx SpecContext) {
			const rounds = 1024
			var (
				db     = mustOpenDBOnFS(ctx, FileSystems["memFS"]())
				idxKey = GenerateChannelKey()
				key    = GenerateChannelKey()
				stamps = make([]telem.TimeStamp, rounds+1)
			)
			for i := range stamps {
				stamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
			}
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
				stamps[0],
				telem.NewSeries(stamps),
			)).To(Succeed())
			Expect(db.WriteSeries(
				ctx,
				key,
				stamps[0],
				telem.NewSeriesV[int64](1),
			)).To(Succeed())
			var (
				wg   sync.WaitGroup
				done atomic.Bool
			)
			wg.Go(func() {
				defer GinkgoRecover()
				defer done.Store(true)
				for _, start := range stamps[1 : rounds+1] {
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{key},
						Start:    start,
					}))
					MustSucceed(w.Write(telem.UnaryFrame(
						key,
						telem.NewSeriesV[int64](1),
					)))
					MustSucceed(w.Commit())
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
