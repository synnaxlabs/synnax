// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Purge check", func() {
	nodeOn := func(ctx SpecContext, fsys xfs.FS) testutil.Node {
		GinkgoHelper()
		base := DeferClose(storagetestutil.OpenLayer(ctx))
		tsDB := DeferClose(MustSucceed(cesium.Open(ctx, "", cesium.WithFS(fsys))))
		StartRun(tsDB.Run)
		return testutil.NewNode(ctx, distribution.LayerConfig{
			Storage: &storage.Layer{KV: base.KV, TS: tsDB, Gorp: base.Gorp},
		})
	}
	// orphans writes samples to an index and a data channel of it, neither with a
	// record. The index has the higher key.
	orphans := func(ctx SpecContext, ts *cesium.DB) (idx, data cesium.ChannelKey) {
		GinkgoHelper()
		idx = uint32(distchannel.NewKey(1, 60_001))
		data = uint32(distchannel.NewKey(1, 60_000))
		Expect(ts.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		Expect(commitFrame(ctx, ts, idx, 10*telem.SecondTS, telem.MultiFrame(
			[]cesium.ChannelKey{idx, data},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(10, 11, 12),
				telem.NewSeriesV[int64](1, 2, 3),
			},
		))).To(Succeed())
		return idx, data
	}

	It("Should refuse a purge of an index that kept data uses", func(
		ctx SpecContext,
	) {
		fsys := xfs.NewMem()
		node := nodeOn(ctx, fsys)
		idx, data := orphans(ctx, node.Storage.TS)
		listed := MustSucceed(channel.Purge(ctx, node.DB, node.Storage.TS, nil))
		Expect(listed).To(HaveLen(2))
		Expect(channel.Purge(
			ctx,
			node.DB,
			node.Storage.TS,
			[]channel.Key{channel.Key(idx)},
		)).Error().To(MatchError(ContainSubstring(fmt.Sprintf(
			"channel %d is the index of channel %d, which also holds data with no "+
				"channel record. Purge both channels together",
			idx, data,
		))))
		Expect(fsys.Exists(fmt.Sprint(idx))).To(BeTrue())
		Expect(MustSucceed(channel.Purge(ctx, node.DB, node.Storage.TS, nil))).
			To(HaveLen(2))
	})

	It("Should give a new channel no key of an index that kept data uses", func(
		ctx SpecContext,
	) {
		fsys := xfs.NewMem()
		node := nodeOn(ctx, fsys)
		idx, _ := orphans(ctx, node.Storage.TS)
		Expect(node.Storage.TS.Intend([]cesium.ChannelKey{idx})).To(Succeed())
		cfg, _ := serviceConfig(ctx, node, node.DB)
		svc := DeferClose(MustSucceed(channel.OpenService(
			ctx,
			cfg,
			channel.ServiceConfig{DropExecutor: func(work func()) { work() }},
		)))
		for {
			ch := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.TimestampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			if uint32(ch.Key()) >= idx {
				break
			}
		}
	})
})
