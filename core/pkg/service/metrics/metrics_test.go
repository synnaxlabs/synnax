// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/metrics"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Metrics", func() {
	Describe("Service Creation", func() {
		It("Should create a service with valid configuration", func(ctx SpecContext) {
			svc := MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              groupSvc,
				Framer:             framerSvc,
				DB:                 dist.DB,
				HostKey:            dist.Identity.HostKey,
				Storage:            dist.Storage,
				CollectionInterval: 5 * time.Second,
			}))
			Expect(svc.Close()).To(Succeed())
		})
		It("Should fail with missing Channel service", func(ctx SpecContext) {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Framer:  framerSvc,
				HostKey: dist.Identity.HostKey,
				Storage: dist.Storage,
			})).Error().To(MatchError(ContainSubstring("channel: must be non-nil")))
		})
		It("Should fail with missing Framer service", func(ctx SpecContext) {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel: channelSvc,
				HostKey: dist.Identity.HostKey,
				Storage: dist.Storage,
			})).Error().To(MatchError(ContainSubstring("framer: must be non-nil")))
		})
		It("Should fail with a missing host key", func(ctx SpecContext) {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel: channelSvc,
				Framer:  framerSvc,
				Storage: dist.Storage,
			})).Error().To(MatchError(ContainSubstring("host_key: must be non-zero")))
		})
		It("Should fail with missing Storage", func(ctx SpecContext) {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel: channelSvc,
				Framer:  framerSvc,
				HostKey: dist.Identity.HostKey,
			})).Error().To(MatchError(ContainSubstring("storage: must be non-nil")))
		})
		It("Should apply default collection interval", func(ctx SpecContext) {
			cfg := metrics.DefaultServiceConfig.Override(metrics.ServiceConfig{
				Channel: channelSvc,
				Framer:  framerSvc,
				HostKey: dist.Identity.HostKey,
				Storage: dist.Storage,
			})
			Expect(cfg.CollectionInterval).To(Equal(2 * time.Second))
		})
	})
	Describe("Channel Creation", func() {
		var names []string
		// context.Background() is used because the service must outlive
		// JustBeforeEach.
		JustBeforeEach(func() {
			MustOpen(
				metrics.OpenService(context.Background(), metrics.ServiceConfig{
					Channel:            channelSvc,
					Group:              groupSvc,
					Framer:             framerSvc,
					HostKey:            dist.Identity.HostKey,
					DB:                 dist.DB,
					Storage:            dist.Storage,
					CollectionInterval: 100 * time.Millisecond,
				}),
			)
			names = getNames(dist.Identity.HostKey)
		})
		It("Should create index channel with correct naming", func(ctx SpecContext) {
			expectedName := names[0]
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(expectedName)).
				Entry(&ch).
				Exec(ctx, dist.DB),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.TimestampT))
			Expect(ch.IsIndex).To(BeTrue())
		})
		It("Should create CPU metric channel", func(ctx SpecContext) {
			expectedName := names[1]
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(expectedName)).
				Entry(&ch).
				Exec(ctx, dist.DB),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should create memory metric channel", func(ctx SpecContext) {
			expectedName := names[2]
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(expectedName)).
				Entry(&ch).
				Exec(ctx, dist.DB),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It(
			"Should create total disk size metric channel as calculated",
			func(ctx SpecContext) {
				expectedName := names[3]
				var ch channel.Channel
				Expect(channelSvc.NewRetrieve().
					Where(channel.MatchNames(expectedName)).
					Entry(&ch).
					Exec(ctx, dist.DB),
				).To(Succeed())
				Expect(ch.Name).To(Equal(expectedName))
				Expect(ch.DataType).To(Equal(telem.Float32T))
				Expect(ch.IsCalculated()).To(BeTrue())
			},
		)
		It("Should create ts size metric channel", func(ctx SpecContext) {
			expectedName := names[4]
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(expectedName)).
				Entry(&ch).
				Exec(ctx, dist.DB),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should create kv (pebble) size metric channel", func(ctx SpecContext) {
			expectedName := names[5]
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(expectedName)).
				Entry(&ch).
				Exec(ctx, dist.DB),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should reuse existing channels", func(ctx SpecContext) {
			svc2 := MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              groupSvc,
				Framer:             framerSvc,
				HostKey:            dist.Identity.HostKey,
				DB:                 dist.DB,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))
			var channels []channel.Channel
			Expect(channelSvc.
				NewRetrieve().
				Where(channel.MatchNames(names...)).
				Entries(&channels).
				Exec(ctx, dist.DB),
			).To(Succeed())
			Expect(channels).To(HaveLen(len(names)))
			Expect(svc2.Close()).To(Succeed())
		})
	})
	Describe("Group", func() {
		open := func(ctx context.Context) *metrics.Service {
			GinkgoHelper()
			return MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              groupSvc,
				Framer:             framerSvc,
				HostKey:            dist.Identity.HostKey,
				DB:                 dist.DB,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))
		}
		retrieve := func(ctx context.Context, name string) channel.Channel {
			GinkgoHelper()
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(name)).
				Entry(&ch).
				Exec(ctx, dist.DB)).To(Succeed())
			return ch
		}

		It(
			"Should create its channels in a top-level Metrics channel group",
			func(ctx SpecContext) {
				svc := open(ctx)
				metricsGroup := MustSucceed(
					groupSvc.CreateOrRetrieve(ctx, "Metrics", resource.TypeChannel),
				)
				Expect(retrieve(ctx, getNames(dist.Identity.HostKey)[1]).Group).
					To(Equal(&metricsGroup.Key))
				Expect(svc.Close()).To(Succeed())
			},
		)

		It(
			"Should create its index channels in the Metrics group",
			func(ctx SpecContext) {
				DeferCleanup(open(ctx).Close)
				metricsGroup := MustSucceed(
					groupSvc.CreateOrRetrieve(ctx, "Metrics", resource.TypeChannel),
				)
				names := getNames(dist.Identity.HostKey)
				for _, name := range []string{names[0], names[3] + "_time"} {
					Expect(retrieve(ctx, name).Group).To(Equal(&metricsGroup.Key), name)
				}
			},
		)

		It("Should keep a channel in the group it was moved to", func(ctx SpecContext) {
			names := getNames(dist.Identity.HostKey)
			svc := open(ctx)
			mem := retrieve(ctx, names[2])
			custom := MustSucceed(groupSvc.CreateOrRetrieve(
				ctx, "Custom Metrics Group", resource.TypeChannel,
			))
			Expect(dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).SetGroup(ctx, &custom.Key, mem.Key())
			})).
				To(Succeed())
			Expect(svc.Close()).To(Succeed())
			svc = open(ctx)
			Expect(retrieve(ctx, names[2]).Group).To(Equal(&custom.Key))
			Expect(svc.Close()).To(Succeed())
		})
	})
	Describe("Writer Control Subject", func() {
		It("Should name the writer after the host node", func(ctx SpecContext) {
			MustOpen(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              groupSvc,
				Framer:             framerSvc,
				HostKey:            dist.Identity.HostKey,
				DB:                 dist.DB,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))
			Eventually(func() []string {
				states := dist.Storage.TS.ControlStates()
				holders := make([]string, 0, len(states.Transfers))
				for _, t := range states.Transfers {
					if t.To != nil {
						holders = append(holders, t.To.Subject.Name)
					}
				}
				return holders
			}).Should(ContainElement(fmt.Sprintf(
				"Node %v Metrics Writer", dist.Identity.HostKey,
			)))
		})
	})
	Describe("Metric Collection", func() {
		var (
			streamer  framer.Streamer
			requests  confluence.Inlet[framer.StreamerRequest]
			responses confluence.Outlet[framer.StreamerResponse]
		)
		// context.Background() is used because the service and streamer are used by It
		// blocks, so the context must outlive BeforeEach.
		BeforeEach(func() {
			ctx := context.Background()
			// Write some data to time-series database so disk size metrics are non-zero
			indexCh := &channel.Channel{
				Name:     "metrics_test_index",
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(
					ctx, indexCh, channel.RetrieveIfNameExists(),
				)
			})).To(Succeed())
			dataCh := &channel.Channel{
				Name:       "metrics_test_data",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(
					ctx, dataCh, channel.RetrieveIfNameExists(),
				)
			})).To(Succeed())
			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{indexCh.Key(), dataCh.Key()},
			}))
			now := telem.Now()
			fr := frame.NewUnary(indexCh.Key(), telem.NewSeriesV(now, now+telem.MillisecondTS, now+2*telem.MillisecondTS)).
				Append(dataCh.Key(), telem.NewSeriesV[float32](1.0, 2.0, 3.0))
			MustSucceed(w.Write(fr))
			Expect(w.Close()).To(Succeed())

			MustOpen(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              groupSvc,
				DB:                 dist.DB,
				Framer:             framerSvc,
				HostKey:            dist.Identity.HostKey,
				Storage:            dist.Storage,
				CollectionInterval: 50 * time.Millisecond,
			}))
			var channels []channel.Channel
			names := getNames(dist.Identity.HostKey)
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(names...)).
				Entries(&channels).
				Exec(ctx, dist.DB),
			).To(Succeed())
			Expect(channels).To(HaveLen(len(names)))
			streamer = MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
				Keys: channel.KeysFromChannels(channels),
			}))
			sCtx := signal.Wrap(ctx)
			requests, responses = confluence.Attach(streamer)
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		})
		AfterEach(func(ctx SpecContext) {
			requests.Close()
			Eventually(responses.Outlet()).Should(BeClosed())
		})
		It("Should write metrics at configured interval", func(ctx SpecContext) {
			names := getNames(dist.Identity.HostKey)
			var channels []channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(names...)).
				Entries(&channels).
				Exec(ctx, dist.DB),
			).To(Succeed())
			keyByName := make(map[string]channel.Key, len(channels))
			for _, ch := range channels {
				keyByName[ch.Name] = ch.Key()
			}

			// The raw metrics and the reactive total_size_gb calculation are
			// separate writes the streamer may batch or reorder, so accumulate the
			// first series seen per channel rather than asserting on frame shape.
			collected := make(map[channel.Key]telem.Series, len(names))
			series := func(name string) telem.Series { return collected[keyByName[name]] }
			Eventually(func(g Gomega) {
				var res framer.StreamerResponse
				g.Eventually(responses.Outlet()).Should(Receive(&res))
				for key, s := range res.Frame.Entries() {
					if _, ok := collected[key]; !ok {
						collected[key] = s
					}
				}
				for _, name := range names {
					g.Expect(collected).To(HaveKey(keyByName[name]))
				}
			}).Should(Succeed())

			timeSeries := series(names[0])
			Expect(timeSeries.DataType).To(Equal(telem.TimestampT))
			Expect(timeSeries.Len()).To(Equal(int64(1)))

			cpuSeries := series(names[1])
			Expect(cpuSeries.DataType).To(Equal(telem.Float32T))
			Expect(cpuSeries.Len()).To(Equal(int64(1)))
			cpuVal := cpuSeries.ValueAt[float32](0)
			Expect(cpuVal).To(BeNumerically(">=", 0))
			Expect(cpuVal).To(BeNumerically("<=", 100))

			memSeries := series(names[2])
			Expect(memSeries.DataType).To(Equal(telem.Float32T))
			Expect(memSeries.Len()).To(Equal(int64(1)))
			memVal := memSeries.ValueAt[float32](0)
			Expect(memVal).To(BeNumerically(">=", 0))
			Expect(memVal).To(BeNumerically("<=", 100))

			tsSizeSeries := series(names[4])
			Expect(tsSizeSeries.DataType).To(Equal(telem.Float32T))
			Expect(tsSizeSeries.Len()).To(Equal(int64(1)))
			tsSize := tsSizeSeries.ValueAt[float32](0)
			Expect(tsSize).To(BeNumerically(">", 0))

			kvSizeSeries := series(names[5])
			Expect(kvSizeSeries.DataType).To(Equal(telem.Float32T))
			Expect(kvSizeSeries.Len()).To(Equal(int64(1)))
			kvSize := kvSizeSeries.ValueAt[float32](0)
			Expect(kvSize).To(BeNumerically(">", 0))

			totalSizeSeries := series(names[3])
			Expect(totalSizeSeries.DataType).To(Equal(telem.Float32T))
			Expect(totalSizeSeries.Len()).To(Equal(int64(1)))
			totalSize := totalSizeSeries.ValueAt[float32](0)
			Expect(totalSize).To(BeNumerically("~", tsSize+kvSize, 0.0001))
		})
	})
})

func getNames(hostKey node.Key) []string {
	prefix := "sy_node_" + hostKey.String() + "_metrics_"
	return []string{
		prefix + "time",
		prefix + "cpu_percentage",
		prefix + "mem_percentage",
		prefix + "total_size_gb",
		prefix + "ts_size_gb",
		prefix + "kv_size_gb",
	}
}
