// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Used channel checks", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
	)
	BeforeAll(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
	})

	It("Should leave a channel that only a folder and a status hold as unused", func(
		ctx SpecContext,
	) {
		g := group.Group{Name: "Used check", Scope: resource.TypeChannel}
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Group.NewWriter(tx).Create(ctx, &g); err != nil {
				return err
			}
			ch.Group = &g.Key
			if err := layer.Channel.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			return layer.Status.NewWriter(tx).Set(ctx, &status.Status[any]{
				Key:     "used-check",
				Name:    ch.Name,
				Variant: status.VariantInfo,
				Message: "Owned by a channel",
				Time:    telem.Now(),
				Owner:   new(channel.ResourceID(ch.Key())),
			})
		})).To(Succeed())
		retrieve := func(used bool) []channel.Key {
			GinkgoHelper()
			var found []channel.Channel
			Expect(layer.Channel.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Where(channel.MatchUsed(used)).
				Entries(&found).
				Exec(ctx, db)).To(Succeed())
			return channel.KeysFromChannels(found)
		}
		Expect(retrieve(true)).To(BeEmpty())
		Expect(retrieve(false)).To(ConsistOf(ch.Key()))
	})
})
