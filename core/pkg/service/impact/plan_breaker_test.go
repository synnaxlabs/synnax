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
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Plan breaker", Ordered, ContinueOnFailure, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
		proj  project.Project
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
		proj = project.Project{Name: "Impact plan breaker"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	create := func(ctx SpecContext, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		ch.Name = UniqueChannelName()
		if ch.DataType == "" {
			ch.DataType, ch.Virtual = telem.Float32T, true
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	exists := func(ctx SpecContext, k channel.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Channel.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Exists(ctx, db))
	}
	taskExists := func(ctx SpecContext, k task.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Task.NewRetrieve().
			Where(task.MatchKeys(k)).
			Exists(ctx, db))
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
	createPlot := func(ctx SpecContext, chs ...channel.Key) lineplot.LinePlot {
		GinkgoHelper()
		plot := defaults.Fill(lineplot.LinePlot{Name: "plot", Project: &proj.Key})
		plot.Channels.Y1 = chs
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.LinePlot.NewWriter(tx).Create(ctx, &plot)
		})).To(Succeed())
		return plot
	}
	setStatus := func(ctx SpecContext, key string, owner channel.Key) {
		GinkgoHelper()
		id := channel.ResourceID(owner)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Status.NewWriter(tx).Set(ctx, &status.Status[any]{
				Key:     key,
				Owner:   &id,
				Name:    "reading",
				Variant: status.VariantSuccess,
				Message: "ok",
				Time:    telem.Now(),
			})
		})).To(Succeed())
	}
	deleteOf := func(keys ...channel.Key) []resource.ID {
		return channel.ResourceIDsFromKeys(keys)
	}
	plan := func(
		ctx SpecContext,
		visible func(resource.ID) bool,
		keys ...channel.Key,
	) impact.Plan {
		GinkgoHelper()
		return MustSucceed(layer.Impact.Plan(
			ctx,
			db,
			deleteOf(keys...),
			impact.PlanOptions{Visible: visible},
		))
	}
	// withPicks returns the apply options that make picks.
	withPicks := func(picks map[string]impact.Choice) impact.ApplyOptions {
		return impact.ApplyOptions{Picks: picksOf(picks)}
	}
	apply := func(
		ctx SpecContext,
		opts impact.ApplyOptions,
		keys ...channel.Key,
	) (applied impact.Plan, err error) {
		err = db.WithTx(ctx, func(tx gorp.Tx) error {
			applied, err = layer.Impact.Apply(ctx, tx, deleteOf(keys...), opts)
			return err
		})
		return applied, err
	}
	hide := func(hidden ...resource.ID) func(resource.ID) bool {
		return func(r resource.ID) bool { return !slices.Contains(hidden, r) }
	}

	It("Should refuse a hidden pick that is not a choice", func(ctx SpecContext) {
		ch := create(ctx, channel.Channel{})
		plot := createPlot(ctx, ch.Key())
		visible := hide(lineplot.ResourceID(plot.Key))
		p := plan(ctx, visible, ch.Key())
		Expect(p.Hidden).To(Equal(impact.Hidden{Detach: 1}))
		_, err := apply(ctx, impact.ApplyOptions{
			Token:   p.Token,
			Hidden:  new(impact.Choice("bogus")),
			Visible: visible,
		}, ch.Key())
		Expect(err).To(
			MatchError(validate.ErrValidation),
			"the channel exists after the apply: %v",
			exists(ctx, ch.Key()),
		)
	})

	It(
		"Should give a new token when a NUL in a status key splits into two holders",
		func(
			ctx SpecContext,
		) {
			a := create(ctx, channel.Channel{})
			b := create(ctx, channel.Channel{})
			first, second := "a-"+uuid.New().String(), "b-"+uuid.New().String()
			forged := first + "\x00cascade\x00status:" + second
			setStatus(ctx, forged, a.Key())
			before := plan(ctx, nil, a.Key(), b.Key())
			Expect(before.Entries).To(HaveLen(1))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Status.NewWriter(tx).Delete(ctx, forged)
			})).To(Succeed())
			setStatus(ctx, first, a.Key())
			setStatus(ctx, second, b.Key())
			after := plan(ctx, nil, a.Key(), b.Key())
			Expect(after.Entries).To(HaveLen(2))
			_, err := apply(
				ctx,
				impact.ApplyOptions{Token: before.Token},
				a.Key(),
				b.Key(),
			)
			_, changed := errors.AsType[impact.PlanChangedError](err)
			Expect(changed).To(
				BeTrue(),
				"token %s before, %s after; apply returned %v; channels exist: %v",
				before.Token,
				after.Token,
				err,
				exists(ctx, a.Key()) && exists(ctx, b.Key()),
			)
		},
	)

	// D-225 length-prefixes each field. With no lengths, two holders whose keys shift a
	// field boundary hash the same text with the same entry count.
	It("Should give a new token when two status keys shift a field boundary", func(
		ctx SpecContext,
	) {
		a := create(ctx, channel.Channel{})
		b := create(ctx, channel.Channel{})
		seam := string(impact.EffectCascade) + string(status.ResourceID("").Type)
		id := uuid.New().String()
		k1, k2 := "a"+id, "b"+id+seam+"c"+id
		k1Shifted, k2Shifted := "a"+id+seam+"b"+id, "c"+id
		setStatus(ctx, k1, a.Key())
		setStatus(ctx, k2, b.Key())
		before := plan(ctx, nil, a.Key(), b.Key())
		Expect(before.Entries).To(HaveLen(2))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Status.NewWriter(tx).Delete(ctx, k1, k2)
		})).To(Succeed())
		setStatus(ctx, k1Shifted, a.Key())
		setStatus(ctx, k2Shifted, b.Key())
		after := plan(ctx, nil, a.Key(), b.Key())
		Expect(after.Entries).To(HaveLen(2))
		Expect(after.Token).ToNot(Equal(before.Token))
	})
	It(
		"Should refuse a stale token when a holder appears on a hidden channel it deletes",
		func(
			ctx SpecContext,
		) {
			index := create(
				ctx,
				channel.Channel{IsIndex: true, DataType: telem.TimestampT},
			)
			data := create(ctx, channel.Channel{
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			})
			dataID := channel.ResourceID(data.Key())
			p := plan(ctx, hide(dataID), index.Key())
			Expect(p.Entries).To(BeEmpty())
			Expect(p.Hidden).To(Equal(impact.Hidden{Restrict: 1}))
			late := createTask(ctx, data.Key())
			_, err := apply(ctx, impact.ApplyOptions{
				Token:   p.Token,
				Hidden:  new(impact.ChoiceDelete),
				Visible: hide(dataID, task.ResourceID(late.Key)),
			}, index.Key())
			_, changed := errors.AsType[impact.PlanChangedError](err)
			Expect(changed).To(
				BeTrue(),
				"apply returned %v; the late task exists: %v; the index exists: %v",
				err,
				taskExists(ctx, late.Key),
				exists(ctx, index.Key()),
			)
		},
	)

	It("Should give a new token when only the visibility of an entry changes", func(
		ctx SpecContext,
	) {
		ch := create(ctx, channel.Channel{})
		tsk := createTask(ctx, ch.Key())
		id := task.ResourceID(tsk.Key)
		seen := plan(ctx, nil, ch.Key())
		hidden := plan(ctx, hide(id), ch.Key())
		Expect(hidden.Token).ToNot(Equal(seen.Token))
		Expect(hidden.Entries).To(BeEmpty())
		Expect(hidden.Hidden).To(Equal(impact.Hidden{Restrict: 1}))
		_, err := apply(ctx, impact.ApplyOptions{
			Token:   seen.Token,
			Hidden:  new(impact.ChoiceDelete),
			Visible: hide(id),
		}, ch.Key())
		changed, ok := errors.AsType[impact.PlanChangedError](err)
		Expect(ok).To(BeTrue())
		Expect(changed.Plan).To(Equal(hidden))
		Expect(taskExists(ctx, tsk.Key)).To(BeTrue())
	})

	It("Should apply a hidden pick with no hidden entries as a no-op", func(
		ctx SpecContext,
	) {
		ch := create(ctx, channel.Channel{})
		p := plan(ctx, hide(), ch.Key())
		Expect(p.Hidden).To(BeZero())
		MustSucceed(apply(ctx, impact.ApplyOptions{
			Token:   p.Token,
			Hidden:  new(impact.ChoiceCancel),
			Visible: hide(),
		}, ch.Key()))
		Expect(exists(ctx, ch.Key())).To(BeFalse())
	})

	It("Should check picks with no token and not check the plan", func(
		ctx SpecContext,
	) {
		ch := create(ctx, channel.Channel{})
		tsk := createTask(ctx, ch.Key())
		id := task.ResourceID(tsk.Key)
		Expect(apply(
			ctx,
			withPicks(map[string]impact.Choice{id.String(): "bogus"}),
			ch.Key(),
		)).Error().To(MatchError(validate.ErrValidation))
		Expect(exists(ctx, ch.Key())).To(BeTrue())
	})

	It("Should refuse a garbage token with the plan as it is now", func(
		ctx SpecContext,
	) {
		ch := create(ctx, channel.Channel{})
		createTask(ctx, ch.Key())
		_, err := apply(ctx, impact.ApplyOptions{
			Token:   "zz",
			Visible: hide(),
		}, ch.Key())
		changed, ok := errors.AsType[impact.PlanChangedError](err)
		Expect(ok).To(BeTrue())
		Expect(changed.Plan.Token).To(Equal(plan(ctx, nil, ch.Key()).Token))
		Expect(exists(ctx, ch.Key())).To(BeTrue())
	})
})
