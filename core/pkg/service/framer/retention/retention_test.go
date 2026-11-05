// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package retention_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer/retention"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", Ordered, func() {
	var (
		node       testutil.Node
		channelSvc *channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node = testutil.NewNode(ctx)
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
			Group:    groupSvc,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       node.DB,
			Group:    groupSvc,
			Ontology: otg,
			Label:    labelSvc,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel:      node.Channel,
			DB:           node.DB,
			HostProvider: node.Cluster,
			Ontology:     otg,
			Group:        groupSvc,
			Status:       statusSvc,
		}))
	})

	open := func() {
		GinkgoHelper()
		MustOpen(retention.Open(retention.Config{
			Channel:      channelSvc,
			Framer:       node.Framer,
			HostProvider: node.Cluster,
			Interval:     10 * telem.Millisecond,
		}))
	}

	// create creates an index channel with retention and a data channel on it, and
	// writes one sample to both at each of stamps.
	create := func(
		ctx SpecContext,
		retention telem.TimeSpan,
		stamps ...telem.TimeStamp,
	) (index, data channel.Channel) {
		GinkgoHelper()
		index = channel.Channel{
			Name:      UniqueChannelName(),
			DataType:  telem.TimestampT,
			IsIndex:   true,
			Retention: retention,
		}
		Expect(channelSvc.NewWriter(nil).Create(ctx, &index)).To(Succeed())
		data = channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			LocalIndex: index.LocalKey,
		}
		Expect(channelSvc.NewWriter(nil).Create(ctx, &data)).To(Succeed())
		keys := []channel.Key{index.Key(), data.Key()}
		w := MustSucceed(node.Framer.OpenWriter(ctx, framer.WriterConfig{
			Start: stamps[0],
			Keys:  keys,
		}))
		values := make([]float64, len(stamps))
		Expect(w.Write(frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesV(stamps...),
			telem.NewSeriesV(values...),
		}))).To(BeTrue())
		MustSucceed(w.Commit())
		Expect(w.Close()).To(Succeed())
		return index, data
	}

	read := func(ctx SpecContext, ch channel.Channel) telem.MultiSeries {
		GinkgoHelper()
		key := ch.Key().StorageKey()
		return MustSucceed(node.Storage.TS.Read(ctx, telem.TimeRangeMax, key)).Get(key)
	}

	It("Should delete the data older than the retention of an index", func(
		ctx SpecContext,
	) {
		now := telem.Now()
		old := now.Add(-2 * telem.Hour)
		index, data := create(ctx, telem.Hour, old, old.Add(telem.Second), now)
		open()
		Eventually(func(ctx SpecContext) []byte {
			return read(ctx, index).Data()
		}).WithContext(ctx).Should(Equal(telem.NewSeriesV(now).Data))
		Expect(read(ctx, data).Len()).To(Equal(int64(1)))
	})

	It("Should keep the data of an index with no retention", func(ctx SpecContext) {
		old := telem.Now().Add(-2 * telem.Hour)
		index, _ := create(ctx, 0, old, old.Add(telem.Second))
		open()
		Consistently(func(ctx SpecContext) int64 {
			return read(ctx, index).Len()
		}).WithContext(ctx).Should(Equal(int64(2)))
	})

	It("Should fail to open without a channel service", func() {
		Expect(retention.Open(retention.Config{
			Framer:       node.Framer,
			HostProvider: node.Cluster,
		})).Error().To(MatchError(ContainSubstring("channel: must be non-nil")))
	})
})
