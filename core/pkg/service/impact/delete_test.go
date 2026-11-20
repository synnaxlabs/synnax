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
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Delete", Ordered, func() {
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
		proj = project.Project{Name: "Impact"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	createChannel := func(ctx SpecContext, ch channel.Channel) channel.Channel {
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
	createSchematic := func(ctx SpecContext, ch channel.Key) schematic.Schematic {
		GinkgoHelper()
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = ch
		sch := schematic.Schematic{
			Name:    "pid",
			Project: &proj.Key,
			Nodes: []schematic.Node{{
				Key:    "value",
				Config: schematic.NodeConfig{Variant: value},
			}},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		return sch
	}
	deleteOf := func(keys ...channel.Key) impact.Change {
		return impact.Change{Variant: impact.DeleteChannelsChange{Keys: keys}}
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
	entry := func(id resource.ID, effect impact.Effect) impact.Entry {
		e := impact.Entry{ID: id.String(), Record: id, Effect: effect}
		if effect == impact.EffectRestrict {
			e.Choices = []impact.Choice{impact.ChoiceCancel, impact.ChoiceDelete}
			e.Default = new(impact.ChoiceCancel)
		}
		return e
	}

	It(
		"Should return one entry for each record that a delete changes and count the rest",
		func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, channel.Channel{})
			tsk := createTask(ctx, ch.Key())
			plot := createPlot(ctx, ch.Key())
			createSchematic(ctx, ch.Key())
			owner := channel.ResourceID(ch.Key())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Status.NewWriter(tx).Set(ctx, &status.Status[any]{
					Owner:   &owner,
					Name:    "reading",
					Variant: status.VariantSuccess,
					Message: "ok",
					Time:    telem.Now(),
				})
			})).To(Succeed())
			var st status.Status[any]
			Expect(layer.Status.NewRetrieve[any]().
				Where(status.MatchOwners[any](owner)).
				Entry(&st).
				Exec(ctx, db)).To(Succeed())
			p := MustSucceed(apply(ctx, impact.ApplyOptions{
				Picks: map[string]impact.Choice{
					task.ResourceID(tsk.Key).String(): impact.ChoiceDelete,
				},
			}, ch.Key()))
			Expect(p.Entries).To(ConsistOf(
				entry(task.ResourceID(tsk.Key), impact.EffectRestrict),
				entry(lineplot.ResourceID(plot.Key), impact.EffectDetach),
				entry(status.ResourceID(st.Key), impact.EffectCascade),
			))
			Expect(p.Kept).To(Equal(uint32(1)))
			Expect(p.Change).To(Equal(deleteOf(ch.Key())))
			Expect(exists(ctx, ch.Key())).To(BeFalse())
			Expect(layer.Task.NewRetrieve().
				Where(task.MatchKeys(tsk.Key)).
				Exists(ctx, db)).To(BeFalse())
			var stored lineplot.LinePlot
			Expect(layer.LinePlot.NewRetrieve().
				Where(lineplot.MatchKeys(plot.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Channels.Y1).To(BeEmpty())
		},
	)

	It(
		"Should refuse a delete with no picks and no token with today's restrict error",
		func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, channel.Channel{})
			tsk := createTask(ctx, ch.Key())
			Expect(apply(ctx, impact.ApplyOptions{}, ch.Key())).Error().
				To(MatchError(gorp.RestrictError{
					Target:    gorp.NewTarget("channel", ch.Key()),
					Holder:    gorp.NewTarget("modbus_read_config", tsk.Key),
					Reference: "channels.channel",
				}))
			Expect(exists(ctx, ch.Key())).To(BeTrue())
		},
	)

	It(
		"Should apply the defaults with no token when nothing restricts the delete",
		func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, channel.Channel{})
			plot := createPlot(ctx, ch.Key())
			sch := createSchematic(ctx, ch.Key())
			applied := MustSucceed(apply(ctx, impact.ApplyOptions{}, ch.Key()))
			Expect(applied).To(Equal(impact.Plan{Change: deleteOf(ch.Key())}))
			Expect(exists(ctx, ch.Key())).To(BeFalse())
			var stored lineplot.LinePlot
			Expect(layer.LinePlot.NewRetrieve().
				Where(lineplot.MatchKeys(plot.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Channels.Y1).To(BeEmpty())
			var kept schematic.Schematic
			Expect(layer.Schematic.NewRetrieve().
				Where(schematic.MatchKeys(sch.Key)).
				Entry(&kept).
				Exec(ctx, db)).To(Succeed())
			value, ok := kept.Nodes[0].Config.Variant.(schematic.ValueNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(value.Channel).To(Equal(ch.Key()))
		},
	)

	It(
		"Should delete a task picked for delete with the channel in one transaction",
		func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, channel.Channel{})
			tsk := createTask(ctx, ch.Key())
			id := task.ResourceID(tsk.Key).String()
			Expect(apply(ctx, impact.ApplyOptions{
				Picks: map[string]impact.Choice{id: impact.ChoiceDelete},
			}, ch.Key())).Error().ToNot(HaveOccurred())
			Expect(exists(ctx, ch.Key())).To(BeFalse())
			Expect(layer.Task.NewRetrieve().
				Where(task.MatchKeys(tsk.Key)).
				Exists(ctx, db)).To(BeFalse())
		},
	)

	It("Should delete a data channel picked for delete with its index", func(
		ctx SpecContext,
	) {
		index := createChannel(
			ctx,
			channel.Channel{IsIndex: true, DataType: telem.TimestampT},
		)
		data := createChannel(ctx, channel.Channel{
			DataType:   telem.Float32T,
			LocalIndex: index.LocalKey,
		})
		applied := MustSucceed(apply(ctx, impact.ApplyOptions{
			Picks: map[string]impact.Choice{
				channel.ResourceID(data.Key()).String(): impact.ChoiceDelete,
			},
		}, index.Key()))
		Expect(applied.Entries).To(Equal([]impact.Entry{
			entry(channel.ResourceID(data.Key()), impact.EffectRestrict),
		}))
		Expect(exists(ctx, index.Key())).To(BeFalse())
		Expect(exists(ctx, data.Key())).To(BeFalse())
	})

	It("Should refuse a pick for no entry", func(ctx SpecContext) {
		ch := createChannel(ctx, channel.Channel{})
		Expect(apply(ctx, impact.ApplyOptions{
			Picks: map[string]impact.Choice{"task:none": impact.ChoiceDelete},
		}, ch.Key())).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("no entry is task:none")),
		))
		Expect(exists(ctx, ch.Key())).To(BeTrue())
	})

	It("Should refuse a pick that its entry does not allow", func(ctx SpecContext) {
		ch := createChannel(ctx, channel.Channel{})
		plot := createPlot(ctx, ch.Key())
		id := lineplot.ResourceID(plot.Key).String()
		Expect(apply(ctx, impact.ApplyOptions{
			Picks: map[string]impact.Choice{id: impact.ChoiceDelete},
		}, ch.Key())).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("entry "+id+" cannot take delete")),
		))
		Expect(exists(ctx, ch.Key())).To(BeTrue())
	})
})
