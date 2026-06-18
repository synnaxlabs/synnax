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
	distFramer "github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelmock "github.com/synnaxlabs/synnax/pkg/service/channel/mock"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/metrics"
	"github.com/synnaxlabs/synnax/pkg/service/node"
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
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				DB:                 dist.DB,
				HostProvider:       dist.Cluster,
				Storage:            dist.Storage,
				CollectionInterval: 5 * time.Second,
			}))
			Expect(svc.Close()).To(Succeed())
		})
		It("Should fail with missing Channel service", func(ctx SpecContext) {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Framer:       framerSvc,
				HostProvider: dist.Cluster,
				Storage:      dist.Storage,
			})).Error().To(MatchError(ContainSubstring("channel: must be non-nil")))
		})
		It("Should fail with missing Framer service", func(ctx SpecContext) {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:      channelSvc,
				HostProvider: dist.Cluster,
				Storage:      dist.Storage,
			})).Error().To(MatchError(ContainSubstring("framer: must be non-nil")))
		})
		It("Should fail with missing HostProvider", func(ctx SpecContext) {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel: channelSvc,
				Framer:  framerSvc,
				Storage: dist.Storage,
			})).Error().To(MatchError(ContainSubstring("host_provider: must be non-nil")))
		})
		It("Should fail with missing Storage", func(ctx SpecContext) {
			Expect(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:      channelSvc,
				Framer:       framerSvc,
				HostProvider: dist.Cluster,
			})).Error().To(MatchError(ContainSubstring("storage: must be non-nil")))
		})
		It("Should apply default collection interval", func(ctx SpecContext) {
			cfg := metrics.DefaultServiceConfig.Override(metrics.ServiceConfig{
				Channel:      channelSvc,
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
		// context.Background() is used because svc is stored in a shared var
		// used by It blocks, so the context must outlive JustBeforeEach.
		JustBeforeEach(func() {
			svc = MustSucceed(metrics.OpenService(context.Background(), metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				DB:                 dist.DB,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))
			names = getNames(dist.Cluster.HostKey())
		})
		JustAfterEach(func(ctx SpecContext) {
			Expect(svc.Close()).To(Succeed())
		})
		It("Should create index channel with correct naming", func(ctx SpecContext) {
			expectedName := names[0]
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(expectedName)).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.TimeStampT))
			Expect(ch.IsIndex).To(BeTrue())
		})
		It("Should create CPU metric channel", func(ctx SpecContext) {
			expectedName := names[1]
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(expectedName)).
				Entry(&ch).
				Exec(ctx, nil),
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
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should create total disk size metric channel as calculated", func(ctx SpecContext) {
			expectedName := names[3]
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(expectedName)).
				Entry(&ch).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.IsCalculated()).To(BeTrue())
		})
		It("Should create ts size metric channel", func(ctx SpecContext) {
			expectedName := names[4]
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(expectedName)).
				Entry(&ch).
				Exec(ctx, nil),
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
				Exec(ctx, nil),
			).To(Succeed())
			Expect(ch.Name).To(Equal(expectedName))
			Expect(ch.DataType).To(Equal(telem.Float32T))
			Expect(ch.LocalIndex).ToNot(BeZero())
		})
		It("Should reuse existing channels", func(ctx SpecContext) {
			svc2 := MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				DB:                 dist.DB,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))
			var channels []channel.Channel
			Expect(channelSvc.
				NewRetrieve().
				Where(channel.MatchNames(names...)).
				Entries(&channels).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(channels).To(HaveLen(len(names)))
			Expect(svc2.Close()).To(Succeed())
		})
	})
	Describe("Group Relationship Persistence", func() {
		It("Should not re-attach a channel to the metrics group if it was moved to a different group", func(ctx SpecContext) {
			names := getNames(dist.Cluster.HostKey())
			svc := MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				DB:                 dist.DB,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))

			var cpuChannel channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(names[1])).
				Entry(&cpuChannel).
				Exec(ctx, nil),
			).To(Succeed())

			newGroup := MustSucceed(dist.Group.CreateOrRetrieve(
				ctx, "Custom Metrics Group", channelSvc.Group().OntologyID(),
			))

			metricsGroup := MustSucceed(dist.Group.CreateOrRetrieve(
				ctx, "Metrics", channelSvc.Group().OntologyID(),
			))
			otgWriter := dist.Ontology.NewWriter(nil)
			Expect(otgWriter.DeleteRelationship(
				ctx,
				metricsGroup.OntologyID(),
				ontology.RelationshipTypeParentOf,
				cpuChannel.OntologyID(),
			)).To(Succeed())

			Expect(otgWriter.DefineRelationship(
				ctx,
				newGroup.OntologyID(),
				ontology.RelationshipTypeParentOf,
				cpuChannel.OntologyID(),
			)).To(Succeed())

			var parents []ontology.Resource
			Expect(dist.Ontology.NewRetrieve().
				WhereIDs(cpuChannel.OntologyID()).
				TraverseTo(ontology.ParentsTraverser).
				WhereTypes(ontology.ResourceTypeGroup).
				Entries(&parents).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(parents).To(HaveLen(1))
			Expect(parents[0].ID).To(Equal(newGroup.OntologyID()))

			Expect(svc.Close()).To(Succeed())

			svc = MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				DB:                 dist.DB,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))

			var parentsAfterReopen []ontology.Resource
			Expect(dist.Ontology.NewRetrieve().
				WhereIDs(cpuChannel.OntologyID()).
				TraverseTo(ontology.ParentsTraverser).
				WhereTypes(ontology.ResourceTypeGroup).
				Entries(&parentsAfterReopen).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(parentsAfterReopen).To(HaveLen(1))
			Expect(parentsAfterReopen[0].ID).To(Equal(newGroup.OntologyID()))

			Expect(svc.Close()).To(Succeed())
		})
		It("Should attach channels to the metrics group if they have no group relationship", func(ctx SpecContext) {
			names := getNames(dist.Cluster.HostKey())
			svc := MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				DB:                 dist.DB,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))

			var memChannel channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(names[2])).
				Entry(&memChannel).
				Exec(ctx, nil),
			).To(Succeed())

			metricsGroup := MustSucceed(dist.Group.CreateOrRetrieve(
				ctx, "Metrics", channelSvc.Group().OntologyID(),
			))

			otgWriter := dist.Ontology.NewWriter(nil)
			Expect(otgWriter.DeleteRelationship(
				ctx,
				metricsGroup.OntologyID(),
				ontology.RelationshipTypeParentOf,
				memChannel.OntologyID(),
			)).To(Succeed())

			var parentsBefore []ontology.Resource
			Expect(dist.Ontology.NewRetrieve().
				WhereIDs(memChannel.OntologyID()).
				TraverseTo(ontology.ParentsTraverser).
				WhereTypes(ontology.ResourceTypeGroup).
				Entries(&parentsBefore).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(parentsBefore).To(BeEmpty())

			Expect(svc.Close()).To(Succeed())

			svc = MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				DB:                 dist.DB,
				Storage:            dist.Storage,
				CollectionInterval: 100 * time.Millisecond,
			}))

			var parentsAfterReopen []ontology.Resource
			Expect(dist.Ontology.NewRetrieve().
				WhereIDs(memChannel.OntologyID()).
				TraverseTo(ontology.ParentsTraverser).
				WhereTypes(ontology.ResourceTypeGroup).
				Entries(&parentsAfterReopen).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(parentsAfterReopen).To(HaveLen(1))
			Expect(parentsAfterReopen[0].ID).To(Equal(metricsGroup.OntologyID()))

			Expect(svc.Close()).To(Succeed())
		})
	})
	Describe("Metric Collection", func() {
		var (
			svc       *metrics.Service
			streamer  framer.Streamer
			requests  confluence.Inlet[framer.StreamerRequest]
			responses confluence.Outlet[framer.StreamerResponse]
		)
		// context.Background() is used because svc, streamer, etc. are stored in
		// shared vars used by It blocks, so the context must outlive BeforeEach.
		BeforeEach(func() {
			ctx := context.Background()
			// Write some data to time-series database so disk size metrics are non-zero
			indexCh := &channel.Channel{
				Name:     "metrics_test_index",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(channelSvc.Create(ctx, indexCh, channel.RetrieveIfNameExists())).To(Succeed())
			dataCh := &channel.Channel{
				Name:       "metrics_test_data",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(channelSvc.Create(ctx, dataCh, channel.RetrieveIfNameExists())).To(Succeed())
			w := MustSucceed(channelmock.OpenWriter(ctx, dist, channelSvc, distFramer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{indexCh.Key(), dataCh.Key()},
			}))
			now := telem.Now()
			fr := frame.NewUnary(indexCh.Key(), telem.NewSeriesV(now, now+telem.MillisecondTS, now+2*telem.MillisecondTS)).
				Append(dataCh.Key(), telem.NewSeriesV[float32](1.0, 2.0, 3.0))
			MustSucceed(w.Write(fr))
			Expect(w.Close()).To(Succeed())

			svc = MustSucceed(metrics.OpenService(ctx, metrics.ServiceConfig{
				Channel:            channelSvc,
				Group:              dist.Group,
				Ontology:           dist.Ontology,
				DB:                 dist.DB,
				Framer:             framerSvc,
				HostProvider:       dist.Cluster,
				Storage:            dist.Storage,
				CollectionInterval: 50 * time.Millisecond,
			}))
			var channels []channel.Channel
			names := getNames(dist.Cluster.HostKey())
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(names...)).
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
		AfterEach(func(ctx SpecContext) {
			requests.Close()
			Eventually(responses.Outlet()).Should(BeClosed())
			Expect(svc.Close()).To(Succeed())
		})
		It("Should write metrics at configured interval", func(ctx SpecContext) {
			names := getNames(dist.Cluster.HostKey())
			var channels []channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(names...)).
				Entries(&channels).
				Exec(ctx, nil),
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
			Expect(timeSeries.DataType).To(Equal(telem.TimeStampT))
			Expect(timeSeries.Len()).To(Equal(int64(1)))

			cpuSeries := series(names[1])
			Expect(cpuSeries.DataType).To(Equal(telem.Float32T))
			Expect(cpuSeries.Len()).To(Equal(int64(1)))
			cpuVal := telem.ValueAt[float32](cpuSeries, 0)
			Expect(cpuVal).To(BeNumerically(">=", 0))
			Expect(cpuVal).To(BeNumerically("<=", 100))

			memSeries := series(names[2])
			Expect(memSeries.DataType).To(Equal(telem.Float32T))
			Expect(memSeries.Len()).To(Equal(int64(1)))
			memVal := telem.ValueAt[float32](memSeries, 0)
			Expect(memVal).To(BeNumerically(">=", 0))
			Expect(memVal).To(BeNumerically("<=", 100))

			tsSizeSeries := series(names[4])
			Expect(tsSizeSeries.DataType).To(Equal(telem.Float32T))
			Expect(tsSizeSeries.Len()).To(Equal(int64(1)))
			tsSize := telem.ValueAt[float32](tsSizeSeries, 0)
			Expect(tsSize).To(BeNumerically(">", 0))

			kvSizeSeries := series(names[5])
			Expect(kvSizeSeries.DataType).To(Equal(telem.Float32T))
			Expect(kvSizeSeries.Len()).To(Equal(int64(1)))
			kvSize := telem.ValueAt[float32](kvSizeSeries, 0)
			Expect(kvSize).To(BeNumerically(">", 0))

			totalSizeSeries := series(names[3])
			Expect(totalSizeSeries.DataType).To(Equal(telem.Float32T))
			Expect(totalSizeSeries.Len()).To(Equal(int64(1)))
			totalSize := telem.ValueAt[float32](totalSizeSeries, 0)
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
