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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// Names stored before the NFC rule may be in another form. "pé" in NFC holds U+00E9;
// in NFD it holds "e" and U+0301.
const (
	nfcName = "p\u00e9"
	nfdName = "pe\u0301"
)

var _ = Describe("Ambiguity breaker", func() {
	var (
		db       *gorp.DB
		svc      *channel.Service
		a, b, c  group.Group
		groupSvc *group.Service
		cfg      channel.ServiceConfig
	)
	stored := func(name string, key channel.LocalKey, g *group.Key) channel.Channel {
		return channel.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			LocalKey:    key,
			DataType:    telem.Float32T,
			Virtual:     true,
			Group:       g,
		}
	}
	BeforeEach(func(ctx SpecContext) {
		fresh := testutil.NewNode(ctx)
		db = fresh.DB
		cfg, groupSvc = serviceConfig(ctx, fresh, fresh.DB)
		a = group.Group{Name: "a", Scope: resource.TypeChannel}
		b = group.Group{Name: "b", Scope: resource.TypeChannel}
		c = group.Group{Name: "c", Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			for _, g := range []*group.Group{&a, &b, &c} {
				if err := groupSvc.NewWriter(tx).Create(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
	})
	open := func(ctx SpecContext, chs ...channel.Channel) {
		createStored(ctx, db, chs...)
		svc = MustOpen(channel.OpenService(ctx, cfg))
	}
	retrieveIfExists := func(ctx SpecContext, ch *channel.Channel) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, ch, channel.RetrieveIfNameExists())
		})
	}

	Describe("RetrieveIfNameExists", func() {
		It("Should take the channel of the name in the folder of the request over an "+
			"exact form in another folder", func(ctx SpecContext) {
			elsewhere := stored(nfdName, 7, &a.Key)
			inFolder := stored(nfcName, 8, &b.Key)
			open(ctx, elsewhere, inFolder)
			ch := channel.Channel{
				Name:     nfdName,
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    &b.Key,
			}
			Expect(retrieveIfExists(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).To(Equal(inFolder.Key()))
		})

		It("Should take the one channel of the name in the folder of the request when "+
			"two exact forms sit in other folders", func(ctx SpecContext) {
			inA := stored(nfdName, 7, &a.Key)
			inC := stored(nfdName, 9, &c.Key)
			inFolder := stored(nfcName, 8, &b.Key)
			open(ctx, inA, inC, inFolder)
			ch := channel.Channel{
				Name:     nfdName,
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    &b.Key,
			}
			Expect(retrieveIfExists(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).To(Equal(inFolder.Key()))
		})
	})

	Describe("DeleteManyByNames", func() {
		It("Should refuse a name that two stored channels hold after NFC", func(
			ctx SpecContext,
		) {
			inA := stored(nfcName, 7, &a.Key)
			inB := stored(nfdName, 8, &b.Key)
			open(ctx, inA, inB)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).DeleteManyByNames(ctx, []string{nfcName}, false)
			})).To(MatchError(validate.ErrValidation))
			Expect(MustSucceed(svc.NewRetrieve().
				Where(channel.MatchKeys(inA.Key())).
				Exists(ctx, db))).To(BeTrue())
		})
	})

	Describe("The Arc resolver", func() {
		It("Should refuse a name that two stored channels hold after NFC", func(
			ctx SpecContext,
		) {
			inA := stored(nfcName, 7, &a.Key)
			inB := stored(nfdName, 8, &b.Key)
			open(ctx, inA, inB)
			Expect(svc.NewArcSymbolResolver(db).Resolve(ctx, nfcName)).Error().
				To(MatchError(validate.ErrValidation))
		})

		It("Should refuse a path that two stored channels in one folder hold", func(
			ctx SpecContext,
		) {
			open(ctx, stored("pt", 7, &a.Key), stored("pt", 8, &a.Key))
			r := svc.NewArcSymbolResolver(db)
			Expect(r.Resolve(ctx, "pt")).Error().To(MatchError(ContainSubstring(
				"2 channels are named 'pt': a.pt, a.pt. Name one by its path",
			)))
			Expect(r.Resolve(ctx, "a.pt")).Error().
				To(MatchError(validate.ErrValidation))
		})
	})
})
