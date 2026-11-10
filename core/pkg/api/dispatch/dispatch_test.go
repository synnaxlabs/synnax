// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package dispatch_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/dispatch"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func run[K comparable, A any](
	ctx context.Context,
	handle dispatch.Handler[K, A],
	req actions.DispatchRequest[K, A],
) (res actions.DispatchResponse[A], err error) {
	err = db.WithTx(ctx, func(tx gorp.Tx) error {
		res, err = handle(ctx, tx, req)
		return err
	})
	return res, err
}

var _ = Describe("New", func() {
	Describe("Rack", func() {
		var (
			handle dispatch.Handler[rack.Key, rack.Action]
			rec    *Recorder[rack.Key, rack.Action]
			r      rack.Rack
		)
		BeforeEach(func(ctx SpecContext) {
			handle = dispatch.New(svc.Rack.NewWriter, rack.Writer.Apply)
			r = rack.Rack{Name: "Rack"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Rack.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
			rec = &Recorder[rack.Key, rack.Action]{}
			DeferCleanup(svc.Rack.Actions().OnChange(rec.Record))
		})

		It("Should reply with the actions it broadcasts", func(ctx SpecContext) {
			rename := rack.NewSetAction(
				rack.SetPayload{Path: []string{"name"}, Value: "Renamed"},
			)
			res := MustSucceed(
				run(ctx, handle, actions.DispatchRequest[rack.Key, rack.Action]{
					Key:         r.Key,
					DispatchKey: "dk",
					Actions:     []rack.Action{rename},
				}),
			)
			Expect(res.Actions).To(Equal([]rack.Action{rename}))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].Key).To(Equal(r.Key))
			Expect(frames[0].DispatchKey).To(Equal("dk"))
			Expect(frames[0].Actions).To(Equal(res.Actions))
		})

		It("Should reply with nothing when the apply fails", func(ctx SpecContext) {
			res, err := run(ctx, handle, actions.DispatchRequest[rack.Key, rack.Action]{
				Key:         r.Key + 1000,
				DispatchKey: "dk",
				Actions: []rack.Action{rack.NewSetAction(
					rack.SetPayload{Path: []string{"name"}, Value: "Renamed"},
				)},
			})
			Expect(err).To(MatchError(query.ErrNotFound))
			Expect(res).To(BeZero())
			Expect(rec.Snapshot()).To(BeEmpty())
		})
	})

	It("Should reply with the actions that only the Core computes", func(
		ctx SpecContext,
	) {
		handle := dispatch.New(svc.Channel.NewWriter, channel.Writer.Apply)
		ch := channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Expression: "return 1 + 1",
			Virtual:    true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		rec := &Recorder[channel.Key, channel.Action]{}
		DeferCleanup(svc.Channel.Actions().OnChange(rec.Record))
		edit := channel.NewSetAction(channel.SetPayload{
			Path:  []string{"expression"},
			Value: "return 1.5 * 2.0",
		})
		res := MustSucceed(run(
			ctx,
			handle,
			actions.DispatchRequest[channel.Key, channel.Action]{
				Key:         ch.Key(),
				DispatchKey: "dk",
				Actions:     []channel.Action{edit},
			},
		))
		Expect(res.Actions).To(Equal([]channel.Action{
			edit,
			channel.NewSetAction(channel.SetPayload{
				Path:  []string{"data_type"},
				Value: telem.Float64T,
			}),
		}))
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].DispatchKey).To(Equal("dk"))
		Expect(frames[0].Actions).To(Equal(res.Actions))
	})
})

var _ = Describe("Core-only", func() {
	var (
		ch   channel.Channel
		t    task.Task
		root user.User
	)
	BeforeEach(func(ctx SpecContext) {
		ch = channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		t = task.Task{
			Type: pagerduty.AlertTaskType,
			Rack: svc.Rack.EmbeddedKey,
			Name: "Alerts",
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := svc.Channel.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			return svc.Task.NewWriter(tx).Create(ctx, &t)
		})).To(Succeed())
		root = MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			svc.RBAC,
			svc.User,
			user.User{Name: "Root"},
		))
	})
	dispatchChannel := func(ctx SpecContext, a channel.Action) error {
		_, err := run(
			ctx,
			dispatch.New(svc.Channel.NewWriter, channel.Writer.Apply),
			actions.DispatchRequest[channel.Key, channel.Action]{
				Key:     ch.Key(),
				Actions: []channel.Action{a},
			},
		)
		return err
	}
	dispatchTask := func(ctx SpecContext, a task.Action) error {
		_, err := run(
			ctx,
			dispatch.New(svc.Task.NewWriter, task.Writer.Apply),
			actions.DispatchRequest[task.Key, task.Action]{
				Key:     t.Key,
				Actions: []task.Action{a},
			},
		)
		return err
	}
	dispatchUser := func(ctx SpecContext, a user.Action) error {
		_, err := run(
			ctx,
			dispatch.New(svc.User.NewWriter, user.Writer.Apply),
			actions.DispatchRequest[user.Key, user.Action]{
				Key:     root.Key,
				Actions: []user.Action{a},
			},
		)
		return err
	}
	retrieveTask := func(ctx SpecContext) task.Task {
		GinkgoHelper()
		var res task.Task
		Expect(svc.Task.NewRetrieve().
			Where(task.MatchKeys(t.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	DescribeTable(
		"Should refuse a client action that only the Core dispatches",
		func(ctx SpecContext, send func(SpecContext) error, message string) {
			err := send(ctx)
			Expect(errors.Is(err, actions.ErrCoreOnly)).To(BeTrue())
			Expect(err).To(MatchError(message))
			var stored channel.Channel
			Expect(svc.Channel.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.DataType).To(Equal(telem.TimestampT))
			Expect(retrieveTask(ctx).ConfigHash).To(Equal(t.ConfigHash))
		},
		Entry("a create", func(ctx SpecContext) error {
			return dispatchChannel(
				ctx,
				channel.NewCreateAction(channel.CreatePayload{Channel: ch}),
			)
		}, "only the Core dispatches the action: validation error"),
		Entry("an update of the config hash of a task", func(ctx SpecContext) error {
			return dispatchTask(ctx, task.NewUpdateAction(task.UpdatePayload{
				Value: task.Update{ConfigHash: new("0123456789abcdef")},
			}))
		}, "config_hash is set by the Core: validation error"),
		Entry("an update of the data type of a channel", func(ctx SpecContext) error {
			return dispatchChannel(ctx, channel.NewUpdateAction(channel.UpdatePayload{
				Value: channel.Update{DataType: new(telem.Float64T)},
			}))
		}, "data_type is set by the Core: validation error"),
		Entry("a set of the data type of a channel", func(ctx SpecContext) error {
			return dispatchChannel(ctx, channel.NewSetAction(channel.SetPayload{
				Path:  []string{"data_type"},
				Value: telem.Float64T,
			}))
		}, "data_type is set by the Core: validation error"),
		Entry("a remove of the config hash of a task", func(ctx SpecContext) error {
			return dispatchTask(ctx, task.NewRemoveAction(task.RemovePayload{
				Key: "config_hash",
			}))
		}, "config_hash is set by the Core: validation error"),
		Entry("a set of the root flag of a user", func(ctx SpecContext) error {
			return dispatchUser(ctx, user.NewSetAction(user.SetPayload{
				Path:  []string{"root_user"},
				Value: true,
			}))
		}, "root_user is set by the Core: validation error"),
	)

	It("Should let the Core write the config hash of a task", func(ctx SpecContext) {
		update := task.NewUpdateAction(task.UpdatePayload{
			Value: task.Update{ConfigHash: new("0123456789abcdef")},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.Task.NewWriter(tx).Apply(ctx, t.Key, update)
			return err
		})).To(Succeed())
		Expect(retrieveTask(ctx).ConfigHash).To(Equal("0123456789abcdef"))
	})
})
