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
	"context"
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
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	xfstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Sweep", func() {
	open := func(ctx context.Context, node testutil.Node) *status.Service {
		GinkgoHelper()
		cfg, _ := serviceConfig(ctx, node, node.DB)
		MustOpen(channel.OpenService(
			ctx,
			cfg,
			channel.ServiceConfig{DropExecutor: func(work func()) { work() }},
		))
		return cfg.Status
	}
	// start opens a channel service with cfg that drops storage in line. The caller
	// closes it.
	start := func(ctx context.Context, cfg channel.ServiceConfig) *channel.Service {
		GinkgoHelper()
		return MustSucceed(channel.OpenService(
			ctx,
			cfg,
			channel.ServiceConfig{DropExecutor: func(work func()) { work() }},
		))
	}
	// nodeOn returns a node whose Cesium DB runs on fsys.
	nodeOn := func(ctx SpecContext, fsys xfs.FS) testutil.Node {
		GinkgoHelper()
		base := DeferClose(storagetestutil.OpenLayer(ctx))
		tsDB := DeferClose(MustSucceed(cesium.Open(ctx, "", cesium.WithFS(fsys))))
		StartRun(tsDB.Run)
		return testutil.NewNode(ctx, distribution.LayerConfig{
			Storage: &storage.Layer{KV: base.KV, TS: tsDB, Gorp: base.Gorp},
		})
	}
	// orphan writes samples to a Cesium channel that has no record.
	orphan := func(ctx SpecContext, node testutil.Node, local uint16) cesium.ChannelKey {
		GinkgoHelper()
		key := uint32(distchannel.NewKey(1, distchannel.LocalKey(local)))
		Expect(node.Storage.TS.CreateChannel(ctx, cesium.Channel{
			Key: key, Name: "orphan", IsIndex: true, DataType: telem.TimestampT,
		})).To(Succeed())
		Expect(node.Storage.TS.WriteSeries(
			ctx,
			key,
			10*telem.SecondTS,
			telem.NewSeriesSecondsTSV(10, 11, 12),
		)).To(Succeed())
		return key
	}
	// keptStatus returns the status that names kept data.
	keptStatus := func(
		ctx SpecContext,
		node testutil.Node,
		svc *status.Service,
	) (status.Status[any], error) {
		var s status.Status[any]
		err := svc.NewRetrieve[any]().
			Where(status.MatchKeys[any]("channel_kept_data")).
			Entry(&s).
			Exec(ctx, node.DB)
		return s, err
	}

	It(
		"Should raise a warning status that names the data with no channel record",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			first, second := orphan(ctx, node, 60_000), orphan(ctx, node, 60_001)
			s := MustSucceed(keptStatus(ctx, node, open(ctx, node)))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Found data with no channel"))
			Expect(s.Description).To(MatchRegexp(fmt.Sprintf(
				`^Channels %d and %d hold [0-9]+ B of data but have no channel record `+
					`and no delete\. To keep the data, restore the records from a `+
					"backup\\. To remove it, stop the Core and run `synnax purge %d %d`\\.$",
				first, second, first, second,
			)))
			Expect(node.Storage.TS.Read(ctx, telem.TimeRangeMax, first)).Error().
				To(MatchError(cesium.ErrChannelNotFound))
		},
	)

	It("Should clear the warning status when no data is kept", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		key := orphan(ctx, node, 60_000)
		cfg, _ := serviceConfig(ctx, node, node.DB)
		svc := start(ctx, cfg)
		MustSucceed(keptStatus(ctx, node, cfg.Status))
		Expect(svc.Close()).To(Succeed())
		Expect(channel.Purge(
			ctx,
			node.DB,
			node.Storage.TS,
			[]channel.Key{channel.Key(key)},
		)).To(BeEmpty())
		DeferClose(start(ctx, cfg))
		Expect(keptStatus(ctx, node, cfg.Status)).Error().
			To(MatchError(query.ErrNotFound))
	})

	It(
		"Should remove at the next start the storage that a failed drop leaves",
		func(ctx SpecContext) {
			fsys := xfstestutil.WrapFaultyFS(xfs.NewMem())
			node := nodeOn(ctx, fsys)
			cfg, _ := serviceConfig(ctx, node, node.DB)
			svc := start(ctx, cfg)
			ch := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.TimestampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			dir := fmt.Sprint(ch.Key())
			fsys.SetOptions(xfstestutil.WithFailRename(dir))
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
			Expect(fsys.Exists(dir)).To(BeTrue())
			fsys.SetOptions()
			Expect(svc.Close()).To(Succeed())
			DeferClose(start(ctx, cfg))
			Expect(fsys.Exists(dir)).To(BeFalse())
			Expect(keptStatus(ctx, node, cfg.Status)).Error().
				To(MatchError(query.ErrNotFound))
		},
	)

	It("Should keep the data of a channel with no record that a delete names", func(
		ctx SpecContext,
	) {
		node := testutil.NewNode(ctx)
		key := orphan(ctx, node, 60_000)
		cfg, _ := serviceConfig(ctx, node, node.DB)
		svc := start(ctx, cfg)
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, channel.Key(key), false)
		})).To(Succeed())
		Expect(svc.Close()).To(Succeed())
		DeferClose(start(ctx, cfg))
		s := MustSucceed(keptStatus(ctx, node, cfg.Status))
		Expect(s.Description).To(ContainSubstring(fmt.Sprint(key)))
	})

	It("Should give a new channel no key that kept data holds", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		key := orphan(ctx, node, 60_000)
		cfg, _ := serviceConfig(ctx, node, node.DB)
		svc := DeferClose(start(ctx, cfg))
		ch := channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Leaseholder: 1,
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		Expect(ch.Key().LocalKey()).To(BeNumerically(">", channel.Key(key).LocalKey()))
	})

	It(
		"Should refuse a delete whose intent cannot be made durable",
		func(ctx SpecContext) {
			fsys := xfstestutil.WrapFaultyFS(xfs.NewMem())
			node := nodeOn(ctx, fsys)
			cfg, _ := serviceConfig(ctx, node, node.DB)
			svc := MustOpen(channel.OpenService(ctx, cfg))
			ch := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.TimestampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			fsys.SetOptions(xfstestutil.WithFailWriteAt("delete-intents"))
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(MatchError(xfstestutil.ErrFault))
			fsys.SetOptions()
			var found channel.Channel
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&found).
				Exec(ctx, node.DB)).To(Succeed())
			// The state of the intents is unknown after a failed append, so no delete
			// runs until a restart.
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(MatchError(ContainSubstring("until a restart")))
		},
	)
})
