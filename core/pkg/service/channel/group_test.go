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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Group", Ordered, func() {
	var (
		groupSvc *group.Service
		svc      *channel.Service
		w        channel.Writer
		g        group.Group
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		cfg, gs := serviceConfig(ctx, testutil.NewNode(ctx))
		groupSvc = gs
		svc = MustOpen(channel.OpenService(ctx, cfg))
		w = svc.NewWriter(nil)
		g = MustSucceed(groupSvc.CreateOrRetrieve(ctx, "Sensors", resource.TypeChannel))
	})
	create := func(ctx context.Context, grp *group.Key) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
			Group:    grp,
		}
		Expect(w.Create(ctx, &ch)).To(Succeed())
		return ch
	}
	retrieve := func(ctx context.Context, key channel.Key) channel.Channel {
		GinkgoHelper()
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(key)).
			Entry(&ch).
			Exec(ctx, nil)).To(Succeed())
		return ch
	}

	It("Should create a channel in a group", func(ctx SpecContext) {
		ch := create(ctx, &g.Key)
		Expect(retrieve(ctx, ch.Key()).Group).To(Equal(&g.Key))
	})

	It("Should reject a group that does not exist", func(ctx SpecContext) {
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
			Group:    new(uuid.New()),
		}
		Expect(w.Create(ctx, &ch)).To(MatchError(ContainSubstring("does not exist")))
	})

	Describe("SetGroup", func() {
		It("Should move channels into a group and back out", func(ctx SpecContext) {
			a, b := create(ctx, nil), create(ctx, nil)
			Expect(w.SetGroup(ctx, &g.Key, a.Key(), b.Key())).To(Succeed())
			Expect(retrieve(ctx, a.Key()).Group).To(Equal(&g.Key))
			Expect(retrieve(ctx, b.Key()).Group).To(Equal(&g.Key))
			Expect(w.SetGroup(ctx, nil, a.Key())).To(Succeed())
			Expect(retrieve(ctx, a.Key()).Group).To(BeNil())
		})
	})

	It(
		"Should keep the stored group when a channel is created again",
		func(ctx SpecContext) {
			ch := create(ctx, &g.Key)
			again := channel.Channel{
				Name:     ch.Name,
				DataType: ch.DataType,
				IsIndex:  true,
			}
			Expect(w.Create(ctx, &again, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(retrieve(ctx, ch.Key()).Group).To(Equal(&g.Key))
		},
	)

	It("Should filter channels by group", func(ctx SpecContext) {
		in, out := create(ctx, &g.Key), create(ctx, nil)
		var grouped, ungrouped []channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(in.Key(), out.Key())).
			Where(channel.MatchGroups(g.Key)).
			Entries(&grouped).
			Exec(ctx, nil)).To(Succeed())
		Expect(channel.KeysFromChannels(grouped)).To(ConsistOf(in.Key()))
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(in.Key(), out.Key())).
			Where(channel.MatchHasGroup(false)).
			Entries(&ungrouped).
			Exec(ctx, nil)).To(Succeed())
		Expect(channel.KeysFromChannels(ungrouped)).To(ConsistOf(out.Key()))
	})

	It("Should refuse to delete a group that holds a channel", func(ctx SpecContext) {
		full := MustSucceed(
			groupSvc.CreateOrRetrieve(ctx, "Full", resource.TypeChannel),
		)
		create(ctx, &full.Key)
		Expect(groupSvc.NewWriter(nil).Delete(ctx, full.Key)).To(MatchError(
			ContainSubstring("cannot delete group Full because it is not empty"),
		))
	})
})
