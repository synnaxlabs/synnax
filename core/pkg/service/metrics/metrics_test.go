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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	distFramer "github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/metrics"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Metrics", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})
	Describe("Service Creation", func() {
		It("Should create a service with valid configuration", func() {
			svc := MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            dist.Channel,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				Storage:            dist.Storage,
				CollectionInterval: 5 * time.Second,
			}))
			Expect(svc.Close()).To(Succeed())
		})
		It("Should fail with missing Channel service", func() {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Framer:       framerSvc,
				HostProvider: dist.Cluster,
				Storage:      dist.Storage,
			})).Error().To(MatchError(ContainSubstring("channel: must be non-nil")))
		})
		It("Should fail with missing Framer service", func() {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:      dist.Channel,
				HostProvider: dist.Cluster,
				Storage:      dist.Storage,
			})).Error().To(MatchError(ContainSubstring("framer: must be non-nil")))
		})
		It("Should fail with missing HostProvider", func() {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel: dist.Channel,
				Framer:  framerSvc,
				Storage: dist.Storage,
			})).Error().To(MatchError(ContainSubstring("host_provider: must be non-nil")))
		})
		It("Should fail with missing Storage", func() {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:      dist.Channel,
				Framer:       framerSvc,
				HostProvider: dist.Cluster,
			})).Error().To(MatchError(ContainSubstring("storage: must be non-nil")))
		})
		It("Should apply default collection interval", func() {
			cfg := metrics.DefaultConfig.Override(metrics.ServiceConfig{
				Channel:      dist.Channel,
				Framer:       framerSvc,
				HostProvider: dist.Cluster,
				Storage:      dist.Storage,
			})
			Expect(cfg.CollectionInterval).To(Equal(2 * time.Second))
		})
	})
	Describe("Channel Creation", func() {
		var (
			svc   *metrics.Service
			names []string
		)
		JustBeforeEach(func() {
			svc = MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            dist.Channel,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))
			names = getNames(dist.Cluster.HostKey())
		})
		JustAfterEach(func() {
			Expect(svc.Close()).To(Succeed())
		})
		It("Should create index channel with correct naming", func() {
			expectedName := names[0]
			var ch channel.Channel
			Expect(dist.Channel.NewRetrieve().
				WhereNames(expectedName).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.TimeStampT))
			Expect(ch.IsIndex).To(BeTrue())
		})
		It("Should create CPU metric channel", func() {
			expectedName := names[1]
			var ch channel.Channel
			Expect(dist.Channel.NewRetrieve().
				WhereNames(expectedName).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should create memory metric channel", func() {
			expectedName := names[2]
			var ch channel.Channel
			Expect(dist.Channel.NewRetrieve().
				WhereNames(expectedName).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should create total disk size metric channel", func() {
			expectedName := names[3]
			var ch channel.Channel
			Expect(dist.Channel.NewRetrieve().
				WhereNames(expectedName).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should create ts (cesium) size metric channel", func() {
			expectedName := names[4]
			var ch channel.Channel
			Expect(dist.Channel.NewRetrieve().
				WhereNames(expectedName).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should create kv (pebble) size metric channel", func() {
			expectedName := names[5]
			var ch channel.Channel
			Expect(dist.Channel.NewRetrieve().
				WhereNames(expectedName).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should create channel count metric channel", func() {
			expectedName := names[6]
			var ch channel.Channel
			Expect(dist.Channel.NewRetrieve().
				WhereNames(expectedName).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Int32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should reuse existing channels", func() {
			svc2 := MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            dist.Channel,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))
			var channels []channel.Channel
			Expect(dist.
				Channel.
				NewRetrieve().
				WhereNames(names...).
				Entries(&channels).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(channels).To(HaveLen(len(names)))
			Expect(svc2.Close()).To(Succeed())
		})
	})
	Describe("Metric Collection", func() {
		var (
			svc       *metrics.Service
			streamer  framer.Streamer
			requests  confluence.Inlet[framer.StreamerRequest]
			responses confluence.Outlet[framer.StreamerResponse]
		)
		BeforeEach(func() {
			// Write some data to cesium so disk size metrics are non-zero
			indexCh := &channel.Channel{
				Name:     "metrics_test_index",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(dist.Channel.Create(ctx, indexCh, channel.RetrieveIfNameExists())).To(Succeed())
			dataCh := &channel.Channel{
				Name:       "metrics_test_data",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh, channel.RetrieveIfNameExists())).To(Succeed())
			w := MustSucceed(dist.Framer.OpenWriter(ctx, distFramer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{indexCh.Key(), dataCh.Key()},
			}))
			now := telem.Now()
			fr := frame.NewUnary(indexCh.Key(), telem.NewSeriesV(now, now+telem.MillisecondTS, now+2*telem.MillisecondTS)).
				Append(dataCh.Key(), telem.NewSeriesV[float32](1.0, 2.0, 3.0))
			MustSucceed(w.Write(fr))
			Expect(w.Close()).To(Succeed())

			svc = MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            dist.Channel,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				Storage:            dist.Storage,
				CollectionInterval: 50 * time.Millisecond,
			}))
			var channels []channel.Channel
			names := getNames(dist.Cluster.HostKey())
			Expect(dist.Channel.NewRetrieve().
				WhereNames(names...).
				Entries(&channels).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(channels).To(HaveLen(len(names)))
			streamer = MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
				Keys: channel.KeysFromChannels(channels),
			}))
			sCtx := signal.Wrap(ctx)
			requests, responses = confluence.Attach(streamer)
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		})
		AfterEach(func() {
			requests.Close()
			Eventually(responses.Outlet()).Should(BeClosed())
			Expect(svc.Close()).To(Succeed())
		})
		It("Should write metrics at configured interval", func() {
			var res framer.StreamerResponse
			Eventually(responses.Outlet()).Should(Receive(&res))
			Expect(res.Frame.Count()).To(Equal(7))

			timeSeries := res.Frame.SeriesAt(0)
			Expect(timeSeries.DataType).To(Equal(telem.TimeStampT))
			Expect(timeSeries.Len()).To(Equal(int64(1)))
			latestTime := telem.ValueAt[telem.TimeStamp](res.Frame.SeriesAt(0), -1)

			cpuSeries := res.Frame.SeriesAt(1)
			Expect(cpuSeries.DataType).To(Equal(telem.Float32T))
			Expect(cpuSeries.Len()).To(Equal(int64(1)))
			cpuVal := telem.ValueAt[float32](cpuSeries, 0)
			Expect(cpuVal).To(BeNumerically(">", 0))
			Expect(cpuVal).To(BeNumerically("<", 100))

			memSeries := res.Frame.SeriesAt(2)
			Expect(memSeries.DataType).To(Equal(telem.Float32T))
			Expect(memSeries.Len()).To(Equal(int64(1)))
			memVal := telem.ValueAt[float32](memSeries, 0)
			Expect(memVal).To(BeNumerically(">", 0))
			Expect(memVal).To(BeNumerically("<", 100))

			totalSizeSeries := res.Frame.SeriesAt(3)
			Expect(totalSizeSeries.DataType).To(Equal(telem.Float32T))
			Expect(totalSizeSeries.Len()).To(Equal(int64(1)))
			totalSize := telem.ValueAt[float32](totalSizeSeries, 0)
			Expect(totalSize).To(BeNumerically(">", 0))

			tsSizeSeries := res.Frame.SeriesAt(4)
			Expect(tsSizeSeries.DataType).To(Equal(telem.Float32T))
			Expect(tsSizeSeries.Len()).To(Equal(int64(1)))
			tsSize := telem.ValueAt[float32](tsSizeSeries, 0)
			Expect(tsSize).To(BeNumerically(">", 0))

			kvSizeSeries := res.Frame.SeriesAt(5)
			Expect(kvSizeSeries.DataType).To(Equal(telem.Float32T))
			Expect(kvSizeSeries.Len()).To(Equal(int64(1)))
			kvSize := telem.ValueAt[float32](kvSizeSeries, 0)
			Expect(kvSize).To(BeNumerically(">", 0))

			Expect(totalSize).To(BeNumerically("~", tsSize+kvSize, 0.0001))

			// Verify channel count metric (index 6)
			channelCountSeries := res.Frame.SeriesAt(6)
			Expect(channelCountSeries.DataType).To(Equal(telem.Int32T))
			Expect(channelCountSeries.Len()).To(Equal(int64(1)))
			channelCount := telem.ValueAt[int32](channelCountSeries, 0)
			Expect(channelCount).To(BeNumerically(">", 0))
			Expect(channelCount).To(BeEquivalentTo(dist.Channel.CountExternalNonVirtual()))

			Eventually(responses.Outlet()).Should(Receive(&res))
			Expect(res.Frame.Count()).To(Equal(7))
			timeSeries = res.Frame.SeriesAt(0)
			Expect(timeSeries.DataType).To(Equal(telem.TimeStampT))
			Expect(timeSeries.Len()).To(Equal(int64(1)))
			nextTime := telem.ValueAt[telem.TimeStamp](res.Frame.SeriesAt(0), -1)
			Expect(nextTime).To(BeNumerically(">", latestTime))
		})
	})
})

func getNames(hostKey cluster.NodeKey) []string {
	prefix := "sy_node_" + hostKey.String() + "_metrics_"
	return []string{
		prefix + "time",
		prefix + "cpu_percentage",
		prefix + "mem_percentage",
		prefix + "total_size_gb",
		prefix + "ts_size_gb",
		prefix + "kv_size_gb",
		prefix + "channel_count",
	}
}
