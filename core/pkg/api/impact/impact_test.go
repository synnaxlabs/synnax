// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiimpact "github.com/synnaxlabs/synnax/pkg/api/impact"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Routes", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
		plan  freighter.UnaryHandler[apiimpact.PlanRequest, impact.Plan]
		apply freighter.UnaryHandler[apiimpact.ApplyRequest, impact.Plan]
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
		svc := MustSucceed(apiimpact.NewService(apiimpact.ServiceConfig{
			RBAC:   layer.RBAC,
			Impact: layer.Impact,
		}))
		binder := MustSucceed(bind.NewService(bind.ServiceConfig{
			DB:   db,
			RBAC: layer.RBAC,
			Held: apiauth.HeldTables(),
		}))
		plan = bind.Read(binder, svc.Plan)
		apply = svc.ApplyHandler(binder)
	})

	createChannel := func(ctx SpecContext) channel.Key {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch.Key()
	}
	createTask := func(ctx SpecContext, ch channel.Key) task.Task {
		GinkgoHelper()
		dev := device.Device{
			Key:      UniqueChannelName(),
			Rack:     layer.Rack.EmbeddedKey,
			Name:     "plc",
			Make:     "modbus",
			Model:    "modbus",
			Location: "127.0.0.1:502",
		}
		tsk := task.Task{
			Name: "modbus_read",
			Type: "modbus_read",
			Config: msgpack.EncodedJSON{
				"device": dev.Key,
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "coil-1",
					"channel": ch,
				}},
			},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Device.NewWriter(tx).Create(ctx, &dev); err != nil {
				return err
			}
			return layer.Task.NewWriter(tx).Create(ctx, &tsk)
		})).To(Succeed())
		return tsk
	}
	createUser := func(ctx SpecContext) user.User {
		GinkgoHelper()
		return MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			layer.RBAC,
			layer.User,
			user.User{Name: uuid.New().String()},
		))
	}
	grant := func(
		ctx SpecContext,
		u user.User,
		actions []access.Action,
		objects ...resource.ID,
	) {
		GinkgoHelper()
		MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			layer.RBAC,
			layer.User,
			u.Key,
			actions,
			objects...,
		))
	}
	deleteOf := func(keys ...channel.Key) apiimpact.Change {
		return apiimpact.Change{
			Variant: apiimpact.DeleteChange{Records: channel.ResourceIDsFromKeys(keys)},
		}
	}
	retrieve := []access.Action{access.ActionRetrieve}
	exists := func(ctx SpecContext, k channel.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Channel.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Exists(ctx, db))
	}

	It("Should count a record that the subject cannot retrieve as hidden", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		createTask(ctx, ch)
		u := createUser(ctx)
		grant(ctx, u, retrieve, channel.ResourceID(ch))
		p := MustSucceed(
			plan(AuthedCtx(ctx, u), apiimpact.PlanRequest{Change: deleteOf(ch)}),
		)
		Expect(p.Entries).To(BeEmpty())
		Expect(p.Hidden).To(Equal(impact.Hidden{Restrict: 1}))
		Expect(p.Token).To(HaveLen(16))
	})

	It("Should list a record that the subject can retrieve", func(ctx SpecContext) {
		ch := createChannel(ctx)
		tsk := createTask(ctx, ch)
		u := createUser(ctx)
		grant(ctx, u, retrieve, channel.ResourceID(ch), task.ResourceID(tsk.Key))
		p := MustSucceed(
			plan(AuthedCtx(ctx, u), apiimpact.PlanRequest{Change: deleteOf(ch)}),
		)
		Expect(p.Hidden).To(BeZero())
		Expect(p.Entries).To(HaveLen(1))
		Expect(p.Entries[0].Record).To(Equal(task.ResourceID(tsk.Key)))
	})

	It("Should refuse to plan a channel that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		u := createUser(ctx)
		Expect(plan(AuthedCtx(ctx, u), apiimpact.PlanRequest{Change: deleteOf(ch)})).
			Error().To(MatchError(access.ErrDenied))
	})

	It("Should refuse to apply a channel that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		createTask(ctx, ch)
		u := createUser(ctx)
		for _, token := range []string{"zz", ""} {
			applied, err := apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
				Change: deleteOf(ch),
				Token:  token,
			})
			Expect(applied).To(BeZero())
			Expect(err).To(MatchError(access.ErrDenied))
		}
		Expect(exists(ctx, ch)).To(BeTrue())
	})

	It("Should name the record that the subject named in a retrieve denial", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		u := createUser(ctx)
		Expect(apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{Change: deleteOf(ch)})).
			Error().To(And(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring(
				"cannot retrieve %s",
				channel.ResourceID(ch),
			)),
		))
	})

	It(
		"Should name the channel that an apply names and the subject cannot retrieve",
		func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx)
			createTask(ctx, ch)
			u := createUser(ctx)
			for _, token := range []string{"zz", ""} {
				applied, err := apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
					Change: deleteOf(ch),
					Token:  token,
				})
				Expect(applied).To(BeZero())
				Expect(err).To(And(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						"%s cannot retrieve %s",
						u.ResourceID(),
						channel.ResourceID(ch),
					)),
				))
				_, planned := errors.AsType[impact.PlanChangedError](err)
				Expect(planned).To(BeFalse())
			}
			Expect(exists(ctx, ch)).To(BeTrue())
		},
	)

	It(
		"Should name the channel that a plan names and the subject cannot retrieve",
		func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx)
			u := createUser(ctx)
			Expect(
				plan(AuthedCtx(ctx, u), apiimpact.PlanRequest{Change: deleteOf(ch)}),
			).
				Error().
				To(And(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						"%s cannot retrieve %s",
						u.ResourceID(),
						channel.ResourceID(ch),
					)),
				))
		},
	)

	It("Should name the smallest pick key that is not a record", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		u := createUser(ctx)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			channel.ResourceID(ch),
		)
		unparsed := make(map[string]impact.Choice, 8)
		for c := 'a'; c <= 'h'; c++ {
			unparsed[string(c)] = impact.ChoiceDelete
		}
		refused := And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("a is not a record")),
		)
		for range 32 {
			Expect(plan(AuthedCtx(ctx, u), apiimpact.PlanRequest{
				Change: deleteOf(ch),
				Picks:  unparsed,
			})).Error().To(refused)
			Expect(apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
				Change: deleteOf(ch),
				Picks:  unparsed,
			})).Error().To(refused)
		}
		Expect(exists(ctx, ch)).To(BeTrue())
	})

	It("Should refuse a hidden delete with a denial that names no record", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		tsk := createTask(ctx, ch)
		u := createUser(ctx)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			channel.ResourceID(ch),
		)
		applied, err := apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
			Change: deleteOf(ch),
			Hidden: new(impact.ChoiceDelete),
		})
		Expect(applied).To(BeZero())
		Expect(err).To(And(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring("a record that it cannot see")),
		))
		Expect(err.Error()).ToNot(ContainSubstring(tsk.Key.String()))
		Expect(exists(ctx, ch)).To(BeTrue())
	})

	It("Should name a record that the subject can retrieve in a denial", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		tsk := createTask(ctx, ch)
		id := task.ResourceID(tsk.Key)
		u := createUser(ctx)
		// Every task and status is visible; the task delete cascades to its status.
		grant(
			ctx,
			u,
			retrieve,
			channel.ResourceID(ch),
			resource.ID{Type: resource.TypeTask},
			resource.ID{Type: resource.TypeStatus},
		)
		grant(ctx, u, []access.Action{access.ActionDelete}, channel.ResourceID(ch))
		applied, err := apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
			Change: deleteOf(ch),
			Picks:  map[string]impact.Choice{id.String(): impact.ChoiceDelete},
		})
		Expect(applied).To(BeZero())
		Expect(err).To(And(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring("cannot delete")),
		))
		Expect(err.Error()).ToNot(ContainSubstring("a record that it cannot see"))
		Expect(exists(ctx, ch)).To(BeTrue())
	})

	It("Should apply a current token with a pick that the plan does not reach", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		tsk := createTask(ctx, createChannel(ctx))
		u := createUser(ctx)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			channel.ResourceID(ch),
		)
		picks := map[string]impact.Choice{
			task.ResourceID(tsk.Key).String(): impact.ChoiceDelete,
		}
		picked := MustSucceed(plan(
			AuthedCtx(ctx, u),
			apiimpact.PlanRequest{Change: deleteOf(ch), Picks: picks},
		))
		unpicked := MustSucceed(
			plan(AuthedCtx(ctx, u), apiimpact.PlanRequest{Change: deleteOf(ch)}),
		)
		Expect(picked.Token).To(Equal(unpicked.Token))
		MustSucceed(apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
			Change: deleteOf(ch),
			Token:  picked.Token,
			Picks:  picks,
		}))
		Expect(exists(ctx, ch)).To(BeFalse())
		Expect(MustSucceed(layer.Task.NewRetrieve().
			Where(task.MatchKeys(tsk.Key)).
			Exists(ctx, db))).To(BeTrue())
	})

	It("Should send a changed plan through the error codec", func(ctx SpecContext) {
		ch := createChannel(ctx)
		changed := impact.PlanChangedError{Plan: impact.Plan{
			Records: []resource.ID{channel.ResourceID(ch)},
			Entries: []impact.Entry{{
				Record: task.ResourceID(uuid.New()),
				Effect: impact.EffectRestrict,
				Choices: []impact.Choice{
					impact.ChoiceCancel,
					impact.ChoiceDelete,
				},
				Default: new(impact.ChoiceCancel),
			}},
			Hidden: impact.Hidden{Restrict: 2},
			Token:  "0123456789abcdef0123456789abcdef",
		}}
		codec := errors.NewCodec(apiimpact.ErrorProvider())
		decoded := codec.Decode(codec.Encode(changed, false))
		Expect(decoded).To(MatchError("the records that the change touches changed"))
		got, ok := errors.AsType[impact.PlanChangedError](decoded)
		Expect(ok).To(BeTrue())
		Expect(got.Plan).To(Equal(changed.Plan))
	})
})
