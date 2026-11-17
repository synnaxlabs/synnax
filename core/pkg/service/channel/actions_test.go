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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", Ordered, func() {
	var (
		dist     testutil.Node
		db       *gorp.DB
		svc      *channel.Service
		groupSvc *group.Service
		rec      *Recorder[channel.Key, channel.Action]
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		dist = testutil.NewNode(ctx)
		db = dist.DB
		var cfg channel.ServiceConfig
		cfg, groupSvc = serviceConfig(ctx, dist, db)
		svc = MustOpen(channel.OpenService(ctx, cfg))
	})
	BeforeEach(func() {
		rec = &Recorder[channel.Key, channel.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
	})
	create := func(ctx context.Context, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		ch.Name = UniqueChannelName()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	apply := func(ctx context.Context, key channel.Key, acts ...channel.Action) (
		[]channel.Action,
		error,
	) {
		var computed []channel.Action
		err := db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			computed, err = svc.NewWriter(tx).
				WithDispatchKey("dk").
				Apply(ctx, key, acts...)
			return err
		})
		return computed, err
	}
	retrieve := func(ctx context.Context, key channel.Key) channel.Channel {
		GinkgoHelper()
		var res channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}
	set := func(path string, value any) channel.Action {
		return channel.NewSetAction(
			channel.SetPayload{Path: []string{path}, Value: value},
		)
	}
	index := channel.Channel{DataType: telem.TimestampT, IsIndex: true}
	calculated := channel.Channel{
		DataType:   telem.Float64T,
		Expression: "return 1 + 1",
		Virtual:    true,
	}

	It("Should attach a create that holds the stored channel", func(ctx SpecContext) {
		ch := create(ctx, index)
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Key).To(Equal(ch.Key()))
		Expect(frames[0].Actions).To(Equal([]channel.Action{
			channel.NewCreateAction(
				channel.CreatePayload{Channel: retrieve(ctx, ch.Key())},
			),
		}))
	})

	It(
		"Should not attach a create when it retrieves a calculated channel by name",
		func(ctx SpecContext) {
			ch := create(ctx, calculated)
			recorded := len(rec.Snapshot())
			chs := []channel.Channel{{
				Name:       ch.Name,
				DataType:   telem.Float64T,
				Expression: "return 1 + 1",
			}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					CreateMany(ctx, &chs, channel.RetrieveIfNameExists())
			})).To(Succeed())
			Expect(chs).To(HaveLen(1))
			Expect(chs[0]).To(Equal(retrieve(ctx, ch.Key())))
			Expect(rec.Snapshot()).To(HaveLen(recorded))
		},
	)

	It("Should rename the channel and its storage channel", func(ctx SpecContext) {
		ch := create(ctx, index)
		name := UniqueChannelName()
		rename := channel.NewRenameAction(channel.RenamePayload{Name: name})
		Expect(apply(ctx, ch.Key(), rename)).To(BeEmpty())
		Expect(retrieve(ctx, ch.Key()).Name).To(Equal(name))
		stored := MustSucceed(
			dist.Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()),
		)
		Expect(stored.Name).To(Equal(name))
		frames := rec.Snapshot()
		Expect(frames[len(frames)-1].DispatchKey).To(Equal("dk"))
		Expect(frames[len(frames)-1].Actions).To(Equal([]channel.Action{rename}))
	})

	It("Should attach a rename for each renamed channel", func(ctx SpecContext) {
		ch := create(ctx, index)
		name := UniqueChannelName()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Rename(ctx, ch.Key(), name, false)
		})).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames[len(frames)-1].Actions).To(Equal([]channel.Action{
			channel.NewRenameAction(channel.RenamePayload{Name: name}),
		}))
	})

	It("Should attach a set of the group for each channel", func(ctx SpecContext) {
		ch := create(ctx, index)
		g := group.Group{Name: "Channels", Scope: "channel"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).SetGroup(ctx, &g.Key, ch.Key())
		})).To(Succeed())
		Expect(retrieve(ctx, ch.Key()).Group).To(Equal(&g.Key))
		frames := rec.Snapshot()
		Expect(frames[len(frames)-1].Actions).To(Equal([]channel.Action{
			set("group", &g.Key),
		}))
	})

	It("Should infer the data type of an edited expression", func(ctx SpecContext) {
		ch := create(ctx, calculated)
		edit := set("expression", "return 1.5 * 2.0")
		computed := MustSucceed(apply(ctx, ch.Key(), edit))
		Expect(computed).To(Equal([]channel.Action{set("data_type", telem.Float64T)}))
		stored := retrieve(ctx, ch.Key())
		Expect(stored.Expression).To(Equal("return 1.5 * 2.0"))
		Expect(stored.DataType).To(Equal(telem.Float64T))
		frames := rec.Snapshot()
		Expect(frames[len(frames)-1].Actions).To(Equal(append(
			[]channel.Action{edit},
			computed...,
		)))
	})

	It("Should refuse an expression for a channel that is not calculated", func(
		ctx SpecContext,
	) {
		ch := create(ctx, index)
		Expect(apply(ctx, ch.Key(), set("expression", "return 1"))).
			Error().To(MatchError(ContainSubstring(
			"only a calculated channel can change its expression or operations",
		)))
		Expect(retrieve(ctx, ch.Key()).Expression).To(BeEmpty())
	})

	DescribeTable("Should refuse a path outside the editable fields", func(
		ctx SpecContext,
		path string,
		value any,
	) {
		ch := create(ctx, index)
		Expect(apply(ctx, ch.Key(), set(path, value))).
			Error().To(MatchError(validate.ErrValidation))
		Expect(retrieve(ctx, ch.Key())).To(Equal(ch))
	},
		Entry("name", "name", "renamed"),
		Entry("data_type", "data_type", "float64"),
		Entry("is_index", "is_index", false),
		Entry("virtual", "virtual", true),
	)

	It("Should reject a create", func(ctx SpecContext) {
		ch := create(ctx, index)
		Expect(apply(
			ctx,
			ch.Key(),
			channel.NewCreateAction(channel.CreatePayload{Channel: ch}),
		)).Error().To(MatchError(ContainSubstring(
			"only the Core dispatches a channel create",
		)))
	})

	It("Should attach a set of a data type that the Core changes", func(
		ctx SpecContext,
	) {
		ch := create(ctx, calculated)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).ChangeDataType(ctx, ch.Key(), telem.Int32T)
		})).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames[len(frames)-1].Actions).To(Equal([]channel.Action{
			set("data_type", telem.Int32T),
		}))
	})
})
