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
	dtestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("RetrieveIfNameExistsInGroup", func() {
	var (
		db     *gorp.DB
		svc    *channel.Service
		groups *group.Service
	)
	BeforeEach(func(ctx SpecContext) {
		dist := dtestutil.NewNode(ctx)
		cfg, g := serviceConfig(ctx, dist, dist.DB)
		db, groups, svc = dist.DB, g, MustOpen(channel.OpenService(ctx, cfg))
	})
	// holders stores a channel named name in each of two new folders, as a v0.62 store
	// can hold, and returns them in key order.
	holders := func(ctx SpecContext, name string) []channel.Channel {
		GinkgoHelper()
		chs := make([]channel.Channel, 2)
		for i := range chs {
			f := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
			chs[i] = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    &f.Key,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := groups.NewWriter(tx).Create(ctx, &f); err != nil {
					return err
				}
				return svc.NewWriter(tx).Create(ctx, &chs[i])
			})).To(Succeed())
			chs[i].Name = name
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, chs[i])).
				To(Succeed())
		}
		Expect(chs[1].Key()).To(BeNumerically(">", chs[0].Key()))
		return chs
	}
	create := func(ctx SpecContext, ch *channel.Channel) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).
				Create(ctx, ch, channel.RetrieveIfNameExistsInGroup())
		})).To(Succeed())
	}

	It("Should take the highest key of a name that two other folders hold, and warn",
		func(ctx SpecContext) {
			name := UniqueChannelName()
			held := holders(ctx, name)
			ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
			create(ctx, &ch)
			Expect(ch.Key()).To(Equal(held[1].Key()))
			var warnings []status.Status[any]
			Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
				Where(gorp.Match(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
					return s.Name == name, nil
				})).
				Entries(&warnings).
				Exec(ctx, db)).To(Succeed())
			Expect(warnings).To(ConsistOf(SatisfyAll(
				HaveField("Variant", status.VariantWarning),
				HaveField("Description", SatisfyAll(
					ContainSubstring(held[0].Key().String()),
					ContainSubstring(held[1].Key().String()),
				)),
			)))
		})

	It("Should take the channel in its own folder before ones elsewhere", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		holders(ctx, name)
		own := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &own)
		})).To(Succeed())
		own.Name = name
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, own)).
			To(Succeed())
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		create(ctx, &ch)
		Expect(ch.Key()).To(Equal(own.Key()))
	})
})
