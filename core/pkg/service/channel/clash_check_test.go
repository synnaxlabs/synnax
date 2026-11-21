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
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// D-223: with no check before Allocate, the table's unique check is the one check, and
// a
// clash fails with a UniqueError that names the holder, at the position of the request.
var _ = Describe("Clash checks", func() {
	var (
		db       *gorp.DB
		svc      *channel.Service
		groupSvc *group.Service
		folder   group.Group
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		var cfg channel.ServiceConfig
		cfg, groupSvc = serviceConfig(ctx, node, node.DB)
		svc = MustOpen(channel.OpenService(ctx, cfg))
		folder = MustSucceed(
			groupSvc.CreateOrRetrieve(ctx, "Clashes", resource.TypeChannel),
		)
	})
	virtual := func(name string) channel.Channel {
		return channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
	}
	createMany := func(ctx SpecContext, chs []channel.Channel) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).CreateMany(ctx, &chs)
		})
	}
	clashOf := func(err error) gorp.UniqueError {
		GinkgoHelper()
		var clash gorp.UniqueError
		Expect(errors.As(err, &clash)).To(BeTrue(), "no UniqueError in %v", err)
		return clash
	}
	pathOf := func(err error) []string {
		GinkgoHelper()
		var p validate.PathError
		Expect(errors.As(err, &p)).To(BeTrue(), "no PathError in %v", err)
		return p.Path
	}

	It("Should refuse a stored channel's name with a UniqueError naming it", func(
		ctx SpecContext,
	) {
		first := virtual(UniqueChannelName())
		Expect(createMany(ctx, []channel.Channel{first})).To(Succeed())
		var stored channel.Channel
		Expect(svc.NewRetrieve().Where(channel.MatchNames(first.Name)).
			Entry(&stored).Exec(ctx, db)).To(Succeed())
		err := createMany(ctx, []channel.Channel{
			virtual(UniqueChannelName()),
			virtual(first.Name),
		})
		Expect(pathOf(err)).To(Equal([]string{"1", "name"}))
		Expect(clashOf(err)).To(Equal(gorp.UniqueError{
			Space:  "channel.name",
			Holder: gorp.NewTarget("channel", stored.Key()),
			Value:  first.Name,
		}))
	})

	It("Should refuse the later of two new channels of one name", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		err := createMany(ctx, []channel.Channel{
			virtual(name),
			virtual(UniqueChannelName()),
			virtual(name),
		})
		// The request's own check refuses it before Allocate.
		Expect(pathOf(err)).To(Equal([]string{"2", "name"}))
		Expect(err).To(MatchError(ContainSubstring("duplicate channel name")))
	})

	It("Should refuse a channel named as a folder made earlier in the transaction",
		func(ctx SpecContext) {
			name := UniqueChannelName()
			var made group.Group
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				made = group.Group{
					Name:   name,
					Scope:  resource.TypeChannel,
					Parent: new(group.ResourceID(folder.Key)),
				}
				if err := groupSvc.NewWriter(tx).Create(ctx, &made); err != nil {
					return err
				}
				ch := virtual(name)
				ch.Group = &folder.Key
				chs := []channel.Channel{ch}
				return svc.NewWriter(tx).CreateMany(ctx, &chs)
			})
			Expect(err).To(MatchError(ContainSubstring("already exists")))
			Expect(clashOf(err).Holder).To(Equal(gorp.Target{
				Table: string(resource.TypeGroup),
				Key:   made.Key.String(),
			}))
		},
	)
})
