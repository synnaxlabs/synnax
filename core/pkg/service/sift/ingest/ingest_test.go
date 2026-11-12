// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ingest_test

import (
	"context"
	"strings"
	"sync"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/sift/fake"
	"github.com/synnaxlabs/synnax/pkg/service/sift/ingest"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// recorder collects what an engine reports.
type recorder struct {
	mu          sync.Mutex
	checkpoints []ingest.Checkpoint
	retries     []error
}

func (r *recorder) onCheckpoint(_ context.Context, cp ingest.Checkpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkpoints = append(r.checkpoints, cp)
	return nil
}

func (r *recorder) onRetry(_ context.Context, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retries = append(r.retries, err)
}

func (r *recorder) last() ingest.Checkpoint {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.checkpoints) == 0 {
		return ingest.Checkpoint{}
	}
	return r.checkpoints[len(r.checkpoints)-1]
}

func (r *recorder) samples() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, cp := range r.checkpoints {
		n += cp.Samples
	}
	return n
}

func (r *recorder) retryErrors() []error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]error(nil), r.retries...)
}

// at returns the timestamp s seconds after the epoch.
func at(s int) telem.TimeStamp { return telem.TimeStamp(s) * telem.SecondTS }

// point returns the point Sift stores for a value written at ts.
func point(ts telem.TimeStamp, v any) fake.Point {
	return fake.Point{Time: time.Unix(0, int64(ts)).UTC(), Value: v}
}

var _ = Describe("Engine", func() {
	var (
		server *fake.Server
		conn   *grpc.ClientConn
		rec    *recorder
	)

	BeforeEach(func() {
		server = MustOpen(fake.Open(fake.Config{}))
		conn = MustOpen(grpc.NewClient(
			server.Address(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		))
		rec = &recorder{}
	})

	open := func(
		ctx context.Context,
		keys channel.Keys,
		cfgs ...ingest.Config,
	) *ingest.Engine {
		GinkgoHelper()
		return MustSucceed(ingest.New(ctx, append([]ingest.Config{{
			DB:           db,
			Framer:       framerSvc,
			Channel:      channelSvc,
			Conn:         conn,
			Asset:        "stand",
			Keys:         keys,
			OnCheckpoint: rec.onCheckpoint,
			OnRetry:      rec.onRetry,
			Retry: breaker.Config{
				BaseInterval: 5 * time.Millisecond,
				MaxRetries:   breaker.InfiniteRetries,
			},
		}}, cfgs...)...))
	}

	Describe("New", func() {
		It("Should reject a config without an asset", func(ctx SpecContext) {
			Expect(ingest.New(ctx, ingest.Config{
				DB:           db,
				Framer:       framerSvc,
				Channel:      channelSvc,
				Conn:         conn,
				Keys:         channel.Keys{1},
				OnCheckpoint: rec.onCheckpoint,
			})).Error().To(MatchError(ContainSubstring("asset: required")))
		})

		It("Should reject a virtual channel", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     "virtual_" + uuid.New().String()[:8],
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(ingest.New(ctx, ingest.Config{
				DB:           db,
				Framer:       framerSvc,
				Channel:      channelSvc,
				Conn:         conn,
				Asset:        "stand",
				Keys:         channel.Keys{ch.Key()},
				OnCheckpoint: rec.onCheckpoint,
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("channel "+ch.Name+" is virtual, and the Core stores no "+
					"data for virtual channels"),
			))
		})

		It("Should reject a calculated channel", func(ctx SpecContext) {
			data := createData(ctx, createIndex(ctx), telem.Float64T)
			ch := channel.Channel{
				Name:       "calculated_" + uuid.New().String()[:8],
				DataType:   telem.Float64T,
				Virtual:    true,
				Expression: "return " + data.Name + " * 2",
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(ingest.New(ctx, ingest.Config{
				DB:           db,
				Framer:       framerSvc,
				Channel:      channelSvc,
				Conn:         conn,
				Asset:        "stand",
				Keys:         channel.Keys{ch.Key()},
				OnCheckpoint: rec.onCheckpoint,
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("channel "+ch.Name+" is calculated, and Sift can only "+
					"take stored channels"),
			))
		})

		DescribeTable("Should reject a channel with a name Sift cannot take",
			func(ctx SpecContext, longIndex bool) {
				name := strings.Repeat("a", 248) + uuid.New().String()[:8]
				idx := channel.Channel{
					Name:     "time_" + uuid.New().String()[:8],
					DataType: telem.TimestampT,
					IsIndex:  true,
				}
				data := channel.Channel{
					Name:     "data_" + uuid.New().String()[:8],
					DataType: telem.Float64T,
				}
				if longIndex {
					idx.Name = name
				} else {
					data.Name = name
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					w := channelSvc.NewWriter(tx)
					if err := w.Create(ctx, &idx); err != nil {
						return err
					}
					data.LocalIndex = idx.LocalKey
					return w.Create(ctx, &data)
				})).To(Succeed())
				Expect(ingest.New(ctx, ingest.Config{
					DB:           db,
					Framer:       framerSvc,
					Channel:      channelSvc,
					Conn:         conn,
					Asset:        "stand",
					Keys:         channel.Keys{data.Key()},
					OnCheckpoint: rec.onCheckpoint,
				})).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError("channel "+name+" has a name longer than the 255 "+
						"characters Sift allows"),
				))
			},
			Entry("a data channel", false),
			Entry("an index", true),
		)

		It("Should reject a set of only index channels", func(ctx SpecContext) {
			idx := createIndex(ctx)
			Expect(ingest.New(ctx, ingest.Config{
				DB:           db,
				Framer:       framerSvc,
				Channel:      channelSvc,
				Conn:         conn,
				Asset:        "stand",
				Keys:         channel.Keys{idx.Key()},
				OnCheckpoint: rec.onCheckpoint,
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("select at least one channel that is not an index"),
			))
		})

		It("Should return not found for a channel that does not exist",
			func(ctx SpecContext) {
				Expect(ingest.New(ctx, ingest.Config{
					DB:           db,
					Framer:       framerSvc,
					Channel:      channelSvc,
					Conn:         conn,
					Asset:        "stand",
					Keys:         channel.Keys{channel.Key(1<<16 | 60000)},
					OnCheckpoint: rec.onCheckpoint,
				})).Error().To(MatchError(query.ErrNotFound))
			})
	})

	Describe("Indexes", func() {
		It("Should return the index of each channel in key order",
			func(ctx SpecContext) {
				idxA, idxB := createIndex(ctx), createIndex(ctx)
				a := createData(ctx, idxA, telem.Float64T)
				b := createData(ctx, idxB, telem.Float64T)
				e := open(ctx, channel.Keys{b.Key(), a.Key(), idxA.Key()})
				Expect(e.Indexes()).To(Equal(channel.Keys{idxA.Key(), idxB.Key()}))
			})
	})

	Describe("Upload", func() {
		It("Should send the samples in the time range", func(ctx SpecContext) {
			idx := createIndex(ctx)
			ch := createData(ctx, idx, telem.Float64T)
			write(ctx, idx, []telem.TimeStamp{at(1), at(2), at(3), at(4)},
				map[channel.Key]telem.Series{
					ch.Key(): telem.NewSeriesV(1.0, 2.0, 3.0, 4.0),
				})
			e := open(ctx, channel.Keys{ch.Key()})
			Expect(e.Upload(ctx, telem.TimeRange{Start: at(2), End: at(4)}, nil)).
				To(Succeed())
			Expect(server.Points("stand", ch.Name)).To(Equal([]fake.Point{
				point(at(2), 2.0),
				point(at(3), 3.0),
			}))
			Expect(rec.last().Positions).To(Equal(ingest.Positions{idx.Key(): at(4)}))
			Expect(rec.samples()).To(Equal(int64(2)))
		})

		DescribeTable("Should convert each data type",
			func(ctx SpecContext, s telem.Series, expected any) {
				idx := createIndex(ctx)
				ch := createData(ctx, idx, s.DataType)
				write(ctx, idx, []telem.TimeStamp{at(1)},
					map[channel.Key]telem.Series{ch.Key(): s})
				e := open(ctx, channel.Keys{ch.Key()})
				Expect(e.Upload(ctx, telem.TimeRange{Start: at(0), End: at(2)}, nil)).
					To(Succeed())
				Expect(server.Points("stand", ch.Name)).To(Equal([]fake.Point{
					point(at(1), expected),
				}))
			},
			Entry("float64", telem.NewSeriesV(1.5), 1.5),
			Entry("float32", telem.NewSeriesV[float32](1.5), float32(1.5)),
			Entry("int8", telem.NewSeriesV[int8](-8), int32(-8)),
			Entry("int16", telem.NewSeriesV[int16](-16), int32(-16)),
			Entry("int32", telem.NewSeriesV[int32](-32), int32(-32)),
			Entry("int64", telem.NewSeriesV[int64](-64), int64(-64)),
			Entry("uint8", telem.NewSeriesV[uint8](8), uint32(8)),
			Entry("uint16", telem.NewSeriesV[uint16](16), uint32(16)),
			Entry("uint32", telem.NewSeriesV[uint32](32), uint32(32)),
			Entry("uint64", telem.NewSeriesV[uint64](64), uint64(64)),
			Entry("timestamp", telem.NewSeriesV(at(5)), int64(at(5))),
			Entry("boolean", telem.NewSeriesV(true), true),
			Entry("string", telem.NewSeriesV("open"), "open"),
			Entry("json", MustSucceed(telem.NewJSONSeriesV(map[string]int{"a": 1})),
				`{"a":1}`),
			Entry(
				"uuid",
				telem.NewSeriesV(
					uuid.MustParse("1b4e28ba-2fa1-11d2-883f-0016d3cca427"),
				),
				"1b4e28ba-2fa1-11d2-883f-0016d3cca427",
			),
			Entry("bytes", telem.NewSeriesV([]byte{1, 2}), []byte{1, 2}),
		)

		It("Should send every sample of a channel that starts after its index",
			func(ctx SpecContext) {
				const before, count = 150_000, 120_000
				idx := createIndex(ctx)
				stamps := make([]telem.TimeStamp, before)
				for i := range before {
					stamps[i] = at(1) + telem.TimeStamp(i)
				}
				write(ctx, idx, stamps, nil)
				ch := createData(ctx, idx, telem.Float64T)
				later := make([]telem.TimeStamp, count)
				values := make([]float64, count)
				for i := range count {
					later[i] = at(2) + telem.TimeStamp(i)
					values[i] = float64(i)
				}
				write(ctx, idx, later,
					map[channel.Key]telem.Series{ch.Key(): telem.NewSeries(values)})
				e := open(ctx, channel.Keys{ch.Key()})
				Expect(e.Upload(ctx, telem.TimeRange{Start: at(0), End: at(3)}, nil)).
					To(Succeed())
				points := server.Points("stand", ch.Name)
				Expect(points).To(HaveLen(count))
				Expect(
					points[count-1],
				).To(Equal(point(later[count-1], float64(count-1))))
			})

		It("Should send one flow per index", func(ctx SpecContext) {
			idxA, idxB := createIndex(ctx), createIndex(ctx)
			a := createData(ctx, idxA, telem.Float64T)
			b := createData(ctx, idxB, telem.Int32T)
			write(ctx, idxA, []telem.TimeStamp{at(1)},
				map[channel.Key]telem.Series{a.Key(): telem.NewSeriesV(1.0)})
			write(ctx, idxB, []telem.TimeStamp{at(2)},
				map[channel.Key]telem.Series{b.Key(): telem.NewSeriesV[int32](2)})
			e := open(ctx, channel.Keys{a.Key(), b.Key()})
			Expect(e.Upload(ctx, telem.TimeRange{Start: at(0), End: at(3)}, nil)).
				To(Succeed())
			Expect(server.Points("stand", a.Name)).To(Equal([]fake.Point{
				point(at(1), 1.0),
			}))
			Expect(server.Points("stand", b.Name)).To(Equal([]fake.Point{
				point(at(2), int32(2)),
			}))
			Expect(server.IngestionConfigs()).To(HaveLen(1))
		})

		It("Should start each index at its position", func(ctx SpecContext) {
			idx := createIndex(ctx)
			ch := createData(ctx, idx, telem.Float64T)
			write(ctx, idx, []telem.TimeStamp{at(1), at(2), at(3)},
				map[channel.Key]telem.Series{
					ch.Key(): telem.NewSeriesV(1.0, 2.0, 3.0),
				})
			e := open(ctx, channel.Keys{ch.Key()})
			Expect(e.Upload(
				ctx,
				telem.TimeRange{Start: at(0), End: at(4)},
				ingest.Positions{idx.Key(): at(2) + 1},
			)).To(Succeed())
			Expect(server.Points("stand", ch.Name)).To(Equal([]fake.Point{
				point(at(3), 3.0),
			}))
		})

		It("Should send an empty value where a channel has no sample",
			func(ctx SpecContext) {
				idx := createIndex(ctx)
				a := createData(ctx, idx, telem.Float64T)
				b := createData(ctx, idx, telem.Float64T)
				write(ctx, idx, []telem.TimeStamp{at(1)},
					map[channel.Key]telem.Series{a.Key(): telem.NewSeriesV(1.0)})
				write(ctx, idx, []telem.TimeStamp{at(2)},
					map[channel.Key]telem.Series{
						a.Key(): telem.NewSeriesV(2.0),
						b.Key(): telem.NewSeriesV(20.0),
					})
				e := open(ctx, channel.Keys{a.Key(), b.Key()})
				Expect(e.Upload(ctx, telem.TimeRange{Start: at(0), End: at(3)}, nil)).
					To(Succeed())
				Expect(server.Points("stand", a.Name)).To(HaveLen(2))
				Expect(server.Points("stand", b.Name)).To(Equal([]fake.Point{
					point(at(2), 20.0),
				}))
				Expect(server.Invalid()).To(BeEmpty())
			})

		It("Should reuse the ingestion config of an earlier engine",
			func(ctx SpecContext) {
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				write(ctx, idx, []telem.TimeStamp{at(1)},
					map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0)})
				tr := telem.TimeRange{Start: at(0), End: at(2)}
				Expect(open(ctx, channel.Keys{ch.Key()}).Upload(ctx, tr, nil)).
					To(Succeed())
				Expect(open(ctx, channel.Keys{ch.Key()}).Upload(ctx, tr, nil)).
					To(Succeed())
				Expect(server.IngestionConfigs()).To(HaveLen(1))
			})

		It("Should retry a temporary failure from the confirmed positions",
			func(ctx SpecContext) {
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				write(ctx, idx, []telem.TimeStamp{at(1), at(2)},
					map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0, 2.0)})
				server.Fail(status.Error(codes.Unavailable, "down"))
				e := open(ctx, channel.Keys{ch.Key()})
				Expect(e.Upload(ctx, telem.TimeRange{Start: at(0), End: at(3)}, nil)).
					To(Succeed())
				Expect(server.Points("stand", ch.Name)).To(HaveLen(2))
				Expect(rec.retryErrors()).To(HaveExactElements(
					MatchError(ContainSubstring("down")),
				))
			})

		It("Should return the last error once it runs out of retries",
			func(ctx SpecContext) {
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				write(ctx, idx, []telem.TimeStamp{at(1)},
					map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0)})
				server.Fail(
					status.Error(codes.Unavailable, "down"),
					status.Error(codes.Unavailable, "down"),
				)
				e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{
					Retry: breaker.Config{MaxRetries: 1},
				})
				Expect(e.Upload(ctx, telem.TimeRange{Start: at(0), End: at(2)}, nil)).
					To(MatchError(ContainSubstring("down")))
				Expect(server.Points("stand", ch.Name)).To(BeEmpty())
			})

		It("Should return an error that retrying cannot fix", func(ctx SpecContext) {
			secured := MustOpen(fake.Open(fake.Config{APIKey: "secret"}))
			securedConn := MustOpen(grpc.NewClient(
				secured.Address(),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			))
			idx := createIndex(ctx)
			ch := createData(ctx, idx, telem.Float64T)
			e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{Conn: securedConn})
			Expect(e.Upload(ctx, telem.TimeRange{Start: at(0), End: at(1)}, nil)).
				To(MatchError(ContainSubstring("Sift rejected the API key")))
			Expect(rec.retryErrors()).To(BeEmpty())
		})

		It("Should return the context error when cancelled", func(ctx SpecContext) {
			idx := createIndex(ctx)
			ch := createData(ctx, idx, telem.Float64T)
			server.Fail(
				status.Error(codes.Unavailable, "down"),
				status.Error(codes.Unavailable, "down"),
			)
			cancelCtx, cancel := context.WithCancel(ctx)
			e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{
				OnRetry: func(context.Context, error) { cancel() },
			})
			write(ctx, idx, []telem.TimeStamp{at(1)},
				map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0)})
			Expect(e.Upload(cancelCtx, telem.TimeRange{Start: at(0), End: at(2)}, nil)).
				To(MatchError(context.Canceled))
		})

		It("Should stop sending and confirm what it sent when cancelled",
			func(ctx SpecContext) {
				const count = 200_000
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				stamps := make([]telem.TimeStamp, count)
				values := make([]float64, count)
				for i := range count {
					stamps[i] = at(1) + telem.TimeStamp(i)
					values[i] = float64(i)
				}
				write(ctx, idx, stamps,
					map[channel.Key]telem.Series{ch.Key(): telem.NewSeries(values)})
				e := open(ctx, channel.Keys{ch.Key()})
				cancelCtx, cancel := context.WithCancel(ctx)
				done := make(chan error, 1)
				go func() {
					done <- e.Upload(
						cancelCtx,
						telem.TimeRange{Start: at(0), End: at(2)},
						nil,
					)
				}()
				Eventually(server.Received).Should(BeNumerically(">", 1000))
				cancel()
				Eventually(done).Should(Receive(MatchError(context.Canceled)))
				points := server.Points("stand", ch.Name)
				Expect(len(points)).To(BeNumerically("<", count))
				Expect(rec.samples()).To(BeEquivalentTo(len(points)))
				Expect(rec.last().Positions).To(Equal(ingest.Positions{
					idx.Key(): stamps[len(points)-1] + 1,
				}))
			})
	})

	Describe("Stream", func() {
		stream := func(
			ctx context.Context,
			e *ingest.Engine,
			from ingest.Positions,
		) (stop func() error) {
			GinkgoHelper()
			sCtx, cancel := context.WithCancel(ctx)
			done := make(chan error, 1)
			go func() { done <- e.Stream(sCtx, from) }()
			return func() error {
				cancel()
				var err error
				Eventually(done).Should(Receive(&err))
				return err
			}
		}

		It("Should send stored samples after the positions, then live samples",
			func(ctx SpecContext) {
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				write(ctx, idx, []telem.TimeStamp{at(1), at(2)},
					map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0, 2.0)})
				e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{
					CheckpointInterval: 10 * telem.Millisecond,
				})
				stop := stream(ctx, e, ingest.Positions{idx.Key(): at(2)})
				Eventually(func() []fake.Point {
					return server.Points("stand", ch.Name)
				}).Should(Equal([]fake.Point{point(at(2), 2.0)}))
				write(ctx, idx, []telem.TimeStamp{at(3)},
					map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(3.0)})
				Eventually(func() []fake.Point {
					return server.Points("stand", ch.Name)
				}).Should(Equal([]fake.Point{
					point(at(2), 2.0),
					point(at(3), 3.0),
				}))
				Expect(stop()).To(Succeed())
				Expect(rec.last().Positions).To(Equal(ingest.Positions{
					idx.Key(): at(3) + 1,
				}))
			})

		It("Should start an index without a position at the current time",
			func(ctx SpecContext) {
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				write(ctx, idx, []telem.TimeStamp{at(1)},
					map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0)})
				e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{
					CheckpointInterval: 10 * telem.Millisecond,
				})
				stop := stream(ctx, e, nil)
				// Stream reads the current time on its own goroutine, so write later.
				later := telem.Now() + telem.SecondTS
				write(ctx, idx, []telem.TimeStamp{later},
					map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(2.0)})
				Eventually(func() []fake.Point {
					return server.Points("stand", ch.Name)
				}).Should(Equal([]fake.Point{point(later, 2.0)}))
				Expect(stop()).To(Succeed())
			})

		It("Should report the latest live position, though an index gets no samples",
			func(ctx SpecContext) {
				idleIdx := createIndex(ctx)
				idle := createData(ctx, idleIdx, telem.Float64T)
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				e := open(ctx, channel.Keys{idle.Key(), ch.Key()}, ingest.Config{
					CheckpointInterval: 10 * telem.Millisecond,
				})
				stop := stream(ctx, e, ingest.Positions{
					idleIdx.Key(): at(0),
					idx.Key():     at(0),
				})
				// A sample written before the live stream opens arrives in a catch-up,
				// where the idle index holds the position back, so write until one
				// arrives live.
				i := 0
				Eventually(func() telem.TimeStamp {
					i++
					write(ctx, idx, []telem.TimeStamp{at(i)},
						map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0)})
					return rec.last().Through
				}).Should(BeNumerically(">", at(0)))
				Expect(stop()).To(Succeed())
				last := rec.last()
				Expect(last.Through).To(BeNumerically(">=", last.Positions[idx.Key()]))
				Expect(last.Positions[idleIdx.Key()]).To(Equal(at(0)))
			})

		It("Should report the time it caught up to while it gets no samples",
			func(ctx SpecContext) {
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{
					CheckpointInterval: 10 * telem.Millisecond,
				})
				start := telem.Now()
				stop := stream(ctx, e, ingest.Positions{idx.Key(): at(0)})
				Eventually(func() telem.TimeStamp { return rec.last().Through }).
					Should(BeNumerically(">", start))
				Expect(stop()).To(Succeed())
				Expect(
					rec.last().Positions,
				).To(Equal(ingest.Positions{idx.Key(): at(0)}))
				Expect(rec.samples()).To(BeZero())
			})

		It("Should confirm what it sent when stopped", func(ctx SpecContext) {
			idx := createIndex(ctx)
			ch := createData(ctx, idx, telem.Float64T)
			e := open(ctx, channel.Keys{ch.Key()})
			stop := stream(ctx, e, ingest.Positions{idx.Key(): at(0)})
			write(ctx, idx, []telem.TimeStamp{at(1)},
				map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0)})
			Eventually(server.Received).Should(Equal(1))
			Expect(server.Points("stand", ch.Name)).To(BeEmpty())
			Expect(stop()).To(Succeed())
			Expect(server.Points("stand", ch.Name)).To(Equal([]fake.Point{
				point(at(1), 1.0),
			}))
			Expect(rec.last().Positions).To(Equal(ingest.Positions{
				idx.Key(): at(1) + 1,
			}))
		})

		It("Should not report what Sift did not confirm when stopped during a confirm",
			func(ctx SpecContext) {
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{
					CheckpointInterval: telem.Millisecond,
				})
				server.Pause()
				server.Fail(status.Error(codes.Unavailable, "down"))
				sCtx, cancel := context.WithCancel(ctx)
				done := make(chan error, 1)
				go func() { done <- e.Stream(sCtx, ingest.Positions{idx.Key(): at(0)}) }()
				write(ctx, idx, []telem.TimeStamp{at(1)},
					map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0)})
				Eventually(server.Streams).Should(Equal(1))
				// Gives the engine time to start waiting for the confirm.
				Consistently(done, 50*time.Millisecond).ShouldNot(Receive())
				cancel()
				server.Resume()
				Eventually(done).Should(Receive())
				// An idle tick may report the start, but never the unconfirmed sample.
				Expect(rec.last().Positions[idx.Key()]).To(Equal(at(0)))
				Expect(server.Points("stand", ch.Name)).To(BeEmpty())
			})

		It("Should stop within the confirm timeout when Sift stops reading",
			func(ctx SpecContext) {
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				const count = 50_000
				stamps := make([]telem.TimeStamp, count)
				for i := range stamps {
					stamps[i] = at(1) + telem.TimeStamp(i)*telem.MicrosecondTS
				}
				write(ctx, idx, stamps, map[channel.Key]telem.Series{
					ch.Key(): telem.NewSeries(make([]float64, count)),
				})
				e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{
					ConfirmTimeout: 100 * telem.Millisecond,
				})
				server.Pause()
				DeferCleanup(server.Resume)
				sCtx, cancel := context.WithCancel(ctx)
				done := make(chan error, 1)
				go func() { done <- e.Stream(sCtx, ingest.Positions{idx.Key(): at(0)}) }()
				Eventually(server.Streams).Should(Equal(1))
				// Gives the sends time to fill the stream that Sift does not read.
				Consistently(done, 200*time.Millisecond).ShouldNot(Receive())
				cancel()
				Eventually(done).WithTimeout(2 * time.Second).Should(Receive(Succeed()))
				Expect(rec.last().Positions[idx.Key()]).To(Equal(at(0)))
			})

		It("Should read from storage when it falls behind the live samples",
			func(ctx SpecContext) {
				idx := createIndex(ctx)
				ch := createData(ctx, idx, telem.Float64T)
				e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{
					CheckpointInterval: 10 * telem.Millisecond,
				})
				server.Pause()
				stop := stream(ctx, e, ingest.Positions{idx.Key(): at(0)})
				const count = 1500
				for i := range count {
					write(ctx, idx, []telem.TimeStamp{at(i + 1)},
						map[channel.Key]telem.Series{
							ch.Key(): telem.NewSeriesV(float64(i)),
						})
				}
				server.Resume()
				Eventually(func() []fake.Point {
					return server.Points("stand", ch.Name)
				}).WithTimeout(10 * time.Second).Should(HaveLen(count))
				Expect(stop()).To(Succeed())
			})

		Describe("Samples missing from the live stream", func() {
			// live waits until the engine sends live samples, which it reports as a
			// position at the current time while no samples arrive.
			live := func() {
				GinkgoHelper()
				start := telem.Now()
				Eventually(func() telem.TimeStamp {
					return rec.last().Through
				}).Should(BeNumerically(">", start))
			}

			openWriter := func(
				ctx context.Context,
				keys channel.Keys,
				cfg framer.WriterConfig,
			) func(...telem.Series) {
				GinkgoHelper()
				cfg.Keys = keys
				cfg.EnableAutoCommit = new(true)
				w := MustSucceed(framerSvc.OpenWriter(ctx, cfg))
				DeferCleanup(func() { Expect(w.Close()).To(Succeed()) })
				return func(series ...telem.Series) {
					GinkgoHelper()
					Expect(MustSucceed(w.Write(frame.NewMulti(keys, series)))).
						To(BeTrue())
				}
			}

			It("Should send a stored frame that the live stream did not carry",
				func(ctx SpecContext) {
					idx := createIndex(ctx)
					ch := createData(ctx, idx, telem.Float64T)
					e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{
						CheckpointInterval: 10 * telem.Millisecond,
					})
					stop := stream(ctx, e, ingest.Positions{idx.Key(): at(0)})
					live()
					write(ctx, idx, []telem.TimeStamp{at(1)},
						map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0)})
					Eventually(func() []fake.Point {
						return server.Points("stand", ch.Name)
					}).Should(HaveLen(1))
					keys := channel.Keys{idx.Key(), ch.Key()}
					stored := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
						Keys:  keys,
						Start: at(2),
						Mode:  framer.WriterModePersistOnly,
					}))
					Expect(MustSucceed(stored.Write(frame.NewMulti(keys, []telem.Series{
						telem.NewSeries([]telem.TimeStamp{at(2)}),
						telem.NewSeriesV(2.0),
					})))).To(BeTrue())
					Expect(stored.Close()).To(Succeed())
					write(ctx, idx, []telem.TimeStamp{at(3)},
						map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(3.0)})
					Eventually(func() []fake.Point {
						return server.Points("stand", ch.Name)
					}).Should(Equal([]fake.Point{
						point(at(1), 1.0),
						point(at(2), 2.0),
						point(at(3), 3.0),
					}))
					Expect(stop()).To(Succeed())
					Expect(server.Invalid()).To(BeEmpty())
				})

			It("Should send the values of a channel written without its index",
				func(ctx SpecContext) {
					idx := createIndex(ctx)
					ch := createData(ctx, idx, telem.Float64T)
					other := createData(ctx, idx, telem.Float64T)
					e := open(ctx, channel.Keys{ch.Key(), other.Key()}, ingest.Config{
						CheckpointInterval: 10 * telem.Millisecond,
					})
					stop := stream(ctx, e, ingest.Positions{idx.Key(): at(0)})
					live()
					openWriter(
						ctx,
						channel.Keys{idx.Key(), other.Key()},
						framer.WriterConfig{Start: at(1)},
					)(
						telem.NewSeries([]telem.TimeStamp{at(1), at(2)}),
						telem.NewSeriesV(10.0, 20.0),
					)
					Eventually(func() []fake.Point {
						return server.Points("stand", other.Name)
					}).Should(HaveLen(2))
					openWriter(
						ctx,
						channel.Keys{ch.Key()},
						framer.WriterConfig{Start: at(1)},
					)(telem.NewSeriesV(1.0, 2.0))
					Eventually(func() []fake.Point {
						return server.Points("stand", ch.Name)
					}).Should(Equal([]fake.Point{
						point(at(1), 1.0),
						point(at(2), 2.0),
					}))
					Expect(stop()).To(Succeed())
					Expect(server.Points("stand", other.Name)).To(HaveLen(2))
					Expect(server.Invalid()).To(BeEmpty())
				})
		})

		It("Should retry a temporary failure", func(ctx SpecContext) {
			idx := createIndex(ctx)
			ch := createData(ctx, idx, telem.Float64T)
			write(ctx, idx, []telem.TimeStamp{at(1)},
				map[channel.Key]telem.Series{ch.Key(): telem.NewSeriesV(1.0)})
			server.Fail(status.Error(codes.Unavailable, "down"))
			e := open(ctx, channel.Keys{ch.Key()}, ingest.Config{
				CheckpointInterval: 10 * telem.Millisecond,
			})
			stop := stream(ctx, e, ingest.Positions{idx.Key(): at(0)})
			Eventually(func() []fake.Point {
				return server.Points("stand", ch.Name)
			}).Should(Equal([]fake.Point{point(at(1), 1.0)}))
			Expect(stop()).To(Succeed())
			Expect(rec.retryErrors()).To(HaveExactElements(
				MatchError(ContainSubstring("down")),
			))
		})
	})
})
