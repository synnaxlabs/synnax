// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("UniqueChannelName", func() {
	It("Should generate a name with the test channel prefix", func() {
		Expect(UniqueChannelName()).To(HavePrefix("test_ch_"))
	})
	It("Should generate unique channel names", func() {
		count := 100
		existingNames := make(set.Set[string], count)
		for range count {
			nextName := UniqueChannelName()
			Expect(existingNames.Contains(nextName)).To(BeFalse())
			existingNames.Add(nextName)
		}
	})
})

var _ = Describe("Service helpers", func() {
	var (
		db  *gorp.DB
		svc *channel.Service
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: node.DB,
		}))
		svc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statusSvc,
		}))
	})

	Describe("CreateVirtual", func() {
		It("Should create virtual channels with unique names", func(ctx SpecContext) {
			keys := MustSucceed(CreateVirtual(ctx, db, svc, 3))
			Expect(keys).To(HaveLen(3))
			var channels []channel.Channel
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(keys...)).
				Entries(&channels).
				Exec(ctx, db)).To(Succeed())
			Expect(channel.KeysFromChannels(channels)).To(ConsistOf(keys))
			Expect(channels).To(HaveEach(HaveField("Virtual", BeTrue())))
			Expect(set.New(channel.Names(channels)...)).To(HaveLen(3))
		})

		It("Should create no channels for a zero count", func(ctx SpecContext) {
			Expect(CreateVirtual(ctx, db, svc, 0)).To(BeEmpty())
		})
	})

	Describe("Delete", func() {
		It("Should delete the channels with keys", func(ctx SpecContext) {
			keys := MustSucceed(CreateVirtual(ctx, db, svc, 2))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return Delete(ctx, svc, tx, keys...)
			})).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(keys...)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should refuse a built-in channel", func(ctx SpecContext) {
			builtIn := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.TimestampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Declare(ctx, &builtIn)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return Delete(ctx, svc, tx, builtIn.Key())
			})).To(MatchError(ContainSubstring("can't delete built-in channel")))
		})

		It("Should refuse a channel that another channel restricts", func(
			ctx SpecContext,
		) {
			idx := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &idx)
			})).To(Succeed())
			data := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				LocalIndex: idx.LocalKey,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &data)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return Delete(ctx, svc, tx, idx.Key())
			})).To(MatchError(gorp.RestrictError{
				Target:    gorp.NewTarget("channel", idx.Key()),
				Holder:    gorp.NewTarget("channel", data.Key()),
				Reference: "index",
			}))
		})
	})
})
