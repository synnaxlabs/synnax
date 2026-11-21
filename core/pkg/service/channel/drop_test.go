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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	xfstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete Order", func() {
	inline := func(work func()) { work() }
	// open opens a channel service on node that drops storage in line, and returns it
	// with its status service.
	open := func(
		ctx context.Context,
		node testutil.Node,
	) (*channel.Service, *status.Service) {
		GinkgoHelper()
		cfg, _ := serviceConfig(ctx, node, node.DB)
		svc := MustOpen(channel.OpenService(
			ctx,
			cfg,
			channel.ServiceConfig{DropExecutor: inline},
		))
		return svc, cfg.Status
	}
	create := func(
		ctx context.Context,
		node testutil.Node,
		svc *channel.Service,
	) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Leaseholder: 1,
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}

	It(
		"Should refuse a writer on a channel that a delete claims until the delete rolls back",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			svc, _ := open(ctx, node)
			ch := create(ctx, node, svc)
			rollback := errors.New("rollback")
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				if err := svc.NewWriter(tx).Delete(ctx, ch.Key(), false); err != nil {
					return err
				}
				Expect(node.Framer.OpenWriter(ctx, framer.WriterConfig{
					Keys:  []channel.Key{ch.Key()},
					Start: 10 * telem.SecondTS,
				})).Error().To(MatchError(framer.ErrChannelDeleting))
				return rollback
			})).To(MatchError(rollback))
			Expect(MustSucceed(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  []channel.Key{ch.Key()},
				Start: 10 * telem.SecondTS,
			})).Close()).To(Succeed())
		},
	)

	It(
		"Should drop the storage before the commit returns with an inline executor",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			svc, _ := open(ctx, node)
			ch := create(ctx, node, svc)
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
			Expect(node.Storage.TS.RetrieveChannel(ctx, uint32(ch.Key()))).
				Error().To(MatchError(cesium.ErrChannelNotFound))
		},
	)

	It(
		"Should raise an error status when the storage of a deleted channel cannot be removed",
		func(ctx SpecContext) {
			base := DeferClose(storagetestutil.OpenLayer(ctx))
			fsys := xfstestutil.WrapFaultyFS(xfs.NewMem())
			tsDB := DeferClose(MustSucceed(cesium.Open(ctx, "", cesium.WithFS(fsys))))
			StartRun(tsDB.Run)
			node := testutil.NewNode(ctx, distribution.LayerConfig{
				Storage: &storage.Layer{KV: base.KV, TS: tsDB, Gorp: base.Gorp},
			})
			svc, statuses := open(ctx, node)
			ch := create(ctx, node, svc)
			fsys.SetOptions(xfstestutil.WithFailRename(
				strconv.FormatUint(uint64(ch.Key()), 10),
			))
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
			var errs []status.Status[any]
			Expect(statuses.NewRetrieve[any]().
				Where(status.MatchVariants[any](status.VariantError)).
				Entries(&errs).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(errs).To(ContainElement(SatisfyAll(
				HaveField("Message", "Failed to remove the data of deleted channels"),
				HaveField("Description", Not(BeEmpty())),
			)))
		},
	)
})
