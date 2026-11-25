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
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
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

// guessToken computes a plan token from what D-202 says it hashes: the change type,
// the sorted records, then each entry's record and effect, in entry order.
func guessToken(keys []channel.Key, entries []impact.Entry) string {
	h := sha256.New()
	h.Write([]byte(apiimpact.DeleteChangeType))
	sorted := slices.Clone(keys)
	slices.Sort(sorted)
	for _, k := range slices.Compact(sorted) {
		h.Write([]byte(channel.ResourceID(k).String()))
	}
	for _, e := range entries {
		h.Write([]byte{0})
		h.Write([]byte(e.Record.String()))
		h.Write([]byte{0})
		h.Write([]byte(e.Effect))
	}
	return hex.EncodeToString(h.Sum(nil)[:16])
}

var _ = Describe("Routes breaker", Ordered, ContinueOnFailure, func() {
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

	create := func(ctx SpecContext, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		ch.Name = UniqueChannelName()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	createIndex := func(ctx SpecContext) channel.Channel {
		GinkgoHelper()
		return create(ctx, channel.Channel{IsIndex: true, DataType: telem.TimestampT})
	}
	createData := func(ctx SpecContext, index channel.Channel) channel.Channel {
		GinkgoHelper()
		return create(ctx, channel.Channel{
			DataType:   telem.Float32T,
			LocalIndex: index.LocalKey,
		})
	}
	createVirtual := func(ctx SpecContext) channel.Key {
		GinkgoHelper()
		return create(
			ctx,
			channel.Channel{DataType: telem.Float32T, Virtual: true},
		).Key()
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
	// withDelete returns p as the plan of the delete of the channels with keys.
	withDelete := func(p impact.Plan, keys ...channel.Key) impact.Plan {
		p.Records = channel.ResourceIDsFromKeys(keys)
		return p
	}
	// deletedBy returns what p deletes.
	deletedBy := func(p impact.Plan) []resource.ID { return p.Records }
	retrieve := []access.Action{access.ActionRetrieve}
	retrieveDelete := []access.Action{access.ActionRetrieve, access.ActionDelete}

	It("Should not let a guessed hidden holder be confirmed from the token", func(
		ctx SpecContext,
	) {
		index := createIndex(ctx)
		hidden := createData(ctx, index)
		u := createUser(ctx)
		grant(ctx, u, retrieve, channel.ResourceID(index.Key()))
		keys := []channel.Key{index.Key()}
		p := MustSucceed(plan(
			AuthedCtx(ctx, u),
			apiimpact.PlanRequest{Change: deleteOf(keys...)},
		))
		Expect(p.Entries).To(BeEmpty())
		Expect(p.Hidden).To(Equal(impact.Hidden{Restrict: 1}))
		// The subject knows the change, the empty visible list, and one hidden entry.
		// Channel keys are a 12 bit node and a 20 bit counter, so it walks the node.
		var confirmed []resource.ID
		effects := []impact.Effect{
			impact.EffectRestrict,
			impact.EffectCascade,
			impact.EffectDetach,
		}
		node := uint32(index.Key()) &^ (1<<20 - 1)
		for local := uint32(1); local < 1<<20; local++ {
			guess := channel.ResourceID(channel.Key(node | local))
			for _, effect := range effects {
				entries := []impact.Entry{{Record: guess, Effect: effect}}
				if guessToken(keys, entries) == p.Token {
					confirmed = append(confirmed, guess)
				}
			}
		}
		Expect(confirmed).To(
			BeEmpty(),
			"the token confirms hidden channel %d",
			hidden.Key(),
		)
	})

	It(
		"Should answer a pick for a hidden record as it answers a pick for no record",
		func(
			ctx SpecContext,
		) {
			index := createIndex(ctx)
			hidden := createData(ctx, index)
			visible := createData(ctx, index)
			u := createUser(ctx)
			grant(
				ctx,
				u,
				retrieve,
				channel.ResourceID(index.Key()),
				channel.ResourceID(visible.Key()),
			)
			p := MustSucceed(plan(
				AuthedCtx(ctx, u),
				apiimpact.PlanRequest{Change: deleteOf(index.Key())},
			))
			Expect(p.Hidden).To(Equal(impact.Hidden{Restrict: 1}))
			Expect(p.Entries).To(HaveLen(1))
			probe := func(guess resource.ID) string {
				GinkgoHelper()
				_, err := apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
					Change: deleteOf(index.Key()),
					Token:  p.Token,
					Picks: map[string]impact.Choice{
						guess.String(): impact.ChoiceDelete,
						channel.ResourceID(visible.Key()).String(): "bogus",
					},
					Hidden: new(impact.ChoiceDelete),
				})
				Expect(err).To(MatchError(validate.ErrValidation))
				return strings.ReplaceAll(err.Error(), guess.String(), "<guess>")
			}
			right := channel.ResourceID(hidden.Key())
			wrong := channel.ResourceID(visible.Key() + 1000)
			Expect(probe(right)).To(Equal(probe(wrong)))
		},
	)

	It("Should keep hidden records out of a changed plan in the error codec", func(
		ctx SpecContext,
	) {
		ch := createVirtual(ctx)
		first := createTask(ctx, ch)
		u := createUser(ctx)
		grant(ctx, u, retrieveDelete, channel.ResourceID(ch))
		p := MustSucceed(plan(
			AuthedCtx(ctx, u),
			apiimpact.PlanRequest{Change: deleteOf(ch)},
		))
		second := createTask(ctx, ch)
		_, err := apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
			Change: deleteOf(ch),
			Token:  p.Token,
			Hidden: new(impact.ChoiceDelete),
		})
		changed, ok := errors.AsType[impact.PlanChangedError](err)
		Expect(ok).To(BeTrue())
		Expect(changed.Plan.Hidden).To(Equal(impact.Hidden{Restrict: 2}))
		codec := errors.NewCodec(apiimpact.ErrorProvider())
		payload := codec.Encode(err, false)
		Expect(payload.Data).ToNot(ContainSubstring(first.Key.String()))
		Expect(payload.Data).ToNot(ContainSubstring(second.Key.String()))
		decoded, ok := errors.AsType[impact.PlanChangedError](codec.Decode(payload))
		Expect(ok).To(BeTrue())
		Expect(decoded.Plan.Token).To(Equal(changed.Plan.Token))
		Expect(decoded.Plan.Hidden).To(Equal(impact.Hidden{Restrict: 2}))
	})

	It("Should round-trip a changed plan with a detach entry through the codec", func(
		ctx SpecContext,
	) {
		changed := impact.PlanChangedError{Plan: withDelete(impact.Plan{
			Entries: []impact.Entry{{
				Record: task.ResourceID(uuid.New()),
				Effect: impact.EffectDetach,
			}},
			Kept:  3,
			Token: "0123456789abcdef0123456789abcdef",
		}, createVirtual(ctx))}
		codec := errors.NewCodec(apiimpact.ErrorProvider())
		decoded, ok := errors.AsType[impact.PlanChangedError](
			codec.Decode(codec.Encode(changed, false)),
		)
		Expect(ok).To(BeTrue())
		Expect(decoded.Plan.Entries).To(HaveLen(1))
		Expect(decoded.Plan.Entries[0].Choices).To(BeEmpty())
		Expect(decoded.Plan.Entries[0].Default).To(BeNil())
		Expect(decoded.Plan.Kept).To(Equal(uint32(3)))
		Expect(deletedBy(decoded.Plan)).To(Equal(deletedBy(changed.Plan)))
	})

	It("Should refuse a pick from a subject that cannot see the picked record", func(
		ctx SpecContext,
	) {
		ch := createVirtual(ctx)
		tsk := createTask(ctx, ch)
		id := task.ResourceID(tsk.Key)
		u := createUser(ctx)
		grant(ctx, u, retrieveDelete, channel.ResourceID(ch))
		grant(ctx, u, retrieve, id)
		p := MustSucceed(plan(
			AuthedCtx(ctx, u),
			apiimpact.PlanRequest{Change: deleteOf(ch)},
		))
		Expect(p.Entries).To(HaveLen(1))
		// A second subject with the same plan never saw the task.
		other := createUser(ctx)
		grant(ctx, other, retrieveDelete, channel.ResourceID(ch))
		// The plan that the other subject sees differs, so the token refuses the apply
		// before the pick does, with a plan that does not name the task.
		_, err := apply(AuthedCtx(ctx, other), apiimpact.ApplyRequest{
			Change: deleteOf(ch),
			Token:  p.Token,
			Picks:  map[string]impact.Choice{id.String(): impact.ChoiceDelete},
		})
		changed, ok := errors.AsType[impact.PlanChangedError](err)
		Expect(ok).To(BeTrue(), "%v", err)
		Expect(changed.Plan.Entries).To(BeEmpty())
		Expect(changed.Plan.Hidden).To(Equal(impact.Hidden{Restrict: 1}))
		Expect(MustSucceed(layer.Task.NewRetrieve().
			Where(task.MatchKeys(tsk.Key)).
			Exists(ctx, db))).To(BeTrue())
	})
})
