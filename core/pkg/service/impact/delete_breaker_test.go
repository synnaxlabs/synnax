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
	"fmt"
	"math/rand/v2"
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
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete breaker", Ordered, ContinueOnFailure, func() {
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
		proj = project.Project{Name: "Impact breaker"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	createChannel := func(ctx SpecContext, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		if ch.Name == "" {
			ch.Name = UniqueChannelName()
		}
		if ch.DataType == "" {
			ch.DataType, ch.Virtual = telem.Float32T, true
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	createCalculated := func(ctx SpecContext) channel.Channel {
		GinkgoHelper()
		return createChannel(ctx, channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Expression: "return 1 + 1",
			Virtual:    true,
		})
	}
	exists := func(ctx SpecContext, keys ...channel.Key) bool {
		GinkgoHelper()
		n := MustSucceed(layer.Channel.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			Count(ctx, db))
		return n == len(keys)
	}
	taskExists := func(ctx SpecContext, k task.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Task.NewRetrieve().
			Where(task.MatchKeys(k)).
			Exists(ctx, db))
	}
	keyWithPrefix := func(prefix string) task.Key {
		return uuid.MustParse(fmt.Sprintf("%s-%012x", prefix, rand.Uint64()>>16))
	}
	// createModbusTask creates a modbus task of typ ("modbus_read" or "modbus_write")
	// with key that holds each of chs. A nil key takes a random one.
	createModbusTask := func(
		ctx SpecContext,
		typ string,
		key task.Key,
		chs ...channel.Key,
	) task.Task {
		GinkgoHelper()
		dev := device.Device{
			Key:      UniqueChannelName(),
			Rack:     layer.Rack.EmbeddedKey,
			Name:     "plc",
			Make:     "modbus",
			Model:    "modbus",
			Location: "127.0.0.1:502",
		}
		entries := make([]any, len(chs))
		for i, ch := range chs {
			entries[i] = map[string]any{
				"type":    "coil",
				"key":     fmt.Sprintf("coil-%d", i),
				"address": i,
				"channel": ch,
			}
		}
		tsk := task.Task{
			Key:  key,
			Name: typ,
			Type: typ,
			Config: msgpack.EncodedJSON{
				"device":   dev.Key,
				"channels": entries,
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
	createTask := func(ctx SpecContext, chs ...channel.Key) task.Task {
		GinkgoHelper()
		return createModbusTask(ctx, "modbus_read", uuid.Nil(), chs...)
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
	retrievePlot := func(ctx SpecContext, k lineplot.Key) lineplot.LinePlot {
		GinkgoHelper()
		var stored lineplot.LinePlot
		Expect(layer.LinePlot.NewRetrieve().
			Where(lineplot.MatchKeys(k)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		return stored
	}
	deleteOf := func(keys ...channel.Key) impact.Change {
		return impact.Change{Variant: impact.DeleteChannelsChange{Keys: keys}}
	}
	apply := func(
		ctx SpecContext,
		picks map[string]impact.Choice,
		keys ...channel.Key,
	) (applied impact.Plan, err error) {
		err = db.WithTx(ctx, func(tx gorp.Tx) error {
			applied, err = layer.Impact.Apply(
				ctx,
				tx,
				deleteOf(keys...),
				impact.ApplyOptions{Picks: picks},
			)
			return err
		})
		return applied, err
	}
	// deleteManyErr returns the error of the delete that the channel route ran before
	// it went through Apply, in a transaction that it then aborts.
	deleteManyErr := func(ctx SpecContext, keys ...channel.Key) error {
		GinkgoHelper()
		tx := db.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		return layer.Channel.NewWriter(tx).DeleteMany(ctx, keys, false)
	}
	entry := func(id resource.ID, effect impact.Effect) impact.Entry {
		e := impact.Entry{ID: id.String(), Record: id, Effect: effect}
		if effect == impact.EffectRestrict {
			e.Choices = []impact.Choice{impact.ChoiceCancel, impact.ChoiceDelete}
			e.Default = new(impact.ChoiceCancel)
		}
		return e
	}

	Describe("Calculated channel index", func() {
		It("Should list a line plot that loses the index of a deleted calculated "+
			"channel", func(ctx SpecContext) {
			calc := createCalculated(ctx)
			Expect(calc.LocalIndex).ToNot(BeZero())
			plot := createPlot(ctx, calc.Index())
			applied := MustSucceed(apply(ctx, nil, calc.Key()))
			Expect(exists(ctx, calc.Index())).To(BeFalse())
			Expect(retrievePlot(ctx, plot.Key).Channels.Y1).To(BeEmpty())
			Expect(applied.Entries).To(ContainElement(
				entry(lineplot.ResourceID(plot.Key), impact.EffectDetach),
			))
		})

		It("Should let a caller pick to delete a task that holds the index of a "+
			"deleted calculated channel", func(ctx SpecContext) {
			calc := createCalculated(ctx)
			tsk := createTask(ctx, calc.Index())
			id := task.ResourceID(tsk.Key).String()
			Expect(apply(
				ctx,
				map[string]impact.Choice{id: impact.ChoiceDelete},
				calc.Key(),
			)).Error().ToNot(HaveOccurred())
			Expect(exists(ctx, calc.Key())).To(BeFalse())
			Expect(taskExists(ctx, tsk.Key)).To(BeFalse())
		})
	})

	Describe("Built-in channel", func() {
		It("Should refuse a built-in channel that a task holds as DeleteMany does",
			func(ctx SpecContext) {
				builtIns := layer.Channel.BuiltIns()
				Expect(builtIns).ToNot(BeEmpty())
				builtIn := builtIns[0]
				tsk := createTask(ctx, builtIn)
				DeferCleanup(func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return layer.Task.NewWriter(tx).Delete(ctx, tsk.Key, false)
					})).To(Succeed())
				})
				want := deleteManyErr(ctx, builtIn)
				Expect(want).To(MatchError(ContainSubstring("can't delete built-in")))
				Expect(apply(ctx, nil, builtIn)).Error().To(MatchError(want.Error()))
			})

		It("Should leave a task picked for delete in place when Apply through the DB "+
			"fails on a built-in channel", func(ctx SpecContext) {
			builtIn := layer.Channel.BuiltIns()[0]
			tsk := createTask(ctx, builtIn)
			DeferCleanup(func(ctx SpecContext) {
				if taskExists(ctx, tsk.Key) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return layer.Task.NewWriter(tx).Delete(ctx, tsk.Key, false)
					})).To(Succeed())
				}
			})
			id := task.ResourceID(tsk.Key).String()
			_, err := layer.Impact.Apply(
				ctx,
				db,
				deleteOf(builtIn),
				impact.ApplyOptions{
					Picks: map[string]impact.Choice{id: impact.ChoiceDelete},
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(exists(ctx, builtIn)).To(BeTrue())
			Expect(taskExists(ctx, tsk.Key)).To(BeTrue())
		})
	})

	Describe("Restrict error", func() {
		It("Should name the target that DeleteMany names for one task that holds "+
			"two deleted channels", func(ctx SpecContext) {
			a := createChannel(ctx, channel.Channel{})
			b := createChannel(ctx, channel.Channel{})
			createTask(ctx, a.Key(), b.Key())
			want := deleteManyErr(ctx, b.Key(), a.Key())
			Expect(want).To(HaveOccurred())
			Expect(apply(ctx, nil, b.Key(), a.Key())).Error().
				To(MatchError(want.Error()))
		})

		It("Should name the holder that DeleteMany names for two tasks of two types",
			func(ctx SpecContext) {
				ch := createChannel(ctx, channel.Channel{})
				createModbusTask(
					ctx,
					"modbus_read",
					keyWithPrefix("ffffffff-ffff-4fff-bfff"),
					ch.Key(),
				)
				createModbusTask(
					ctx,
					"modbus_write",
					keyWithPrefix("00000000-0000-4000-8000"),
					ch.Key(),
				)
				want := deleteManyErr(ctx, ch.Key())
				Expect(want).To(HaveOccurred())
				Expect(apply(ctx, nil, ch.Key())).Error().To(MatchError(want.Error()))
			})
	})
})
