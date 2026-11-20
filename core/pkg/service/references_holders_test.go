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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	tableversions "github.com/synnaxlabs/synnax/pkg/service/table/versions"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("References across holders", Ordered, func() {
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
		proj = project.Project{Name: "Holders"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	createChannel := func(ctx SpecContext) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}

	deleteChannel := func(ctx SpecContext, k channel.Key) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Delete(ctx, k, false)
		})
	}

	valueTable := func(keys ...channel.Key) table.Table {
		cells := make([]table.Cell, len(keys))
		for i, k := range keys {
			var cell table.ValueCellConfig
			cell.SetDefaults()
			cell.Channel = k
			cells[i] = table.Cell{
				Key:    "cell" + string(rune('a'+i)),
				Config: table.CellConfig{Variant: cell},
			}
		}
		cols := make([]table.Column, len(keys))
		for i := range keys {
			cols[i] = table.Column{Key: "col" + string(rune('a'+i)), Size: 72}
		}
		return table.Table{
			Name:    "readings",
			Project: &proj.Key,
			Columns: cols,
			Rows:    []table.Row{{Key: "row", Size: 36, Cells: cells}},
		}
	}

	It("Should clear a never-issued key on a table import and keep a deleted one", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		Expect(deleteChannel(ctx, ch.Key())).To(Succeed())
		never := distchannel.NewKey(ch.Leaseholder, ch.LocalKey+1000)
		t := valueTable(ch.Key(), never)
		env := imex.Envelope{
			Version: tableversions.Latest,
			Type:    string(resource.TypeTable),
			Name:    t.Name,
		}
		Expect(env.Encode(t)).To(Succeed())
		var id resource.ID
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			id, err = layer.ImEx.Import(
				ctx,
				tx,
				WireRoundTrip(env),
				imex.ImportOptions{Parent: proj.ResourceID()},
			)
			return err
		})).To(Succeed())
		var res table.Table
		Expect(layer.Table.NewRetrieve().
			Where(table.MatchKeys(MustSucceed(uuid.Parse(id.Key)))).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		channelOf := func(c table.Cell) channel.Key {
			GinkgoHelper()
			v, ok := c.Config.Variant.(table.ValueCellConfig)
			Expect(ok).To(BeTrue())
			return v.Channel
		}
		Expect(channelOf(res.Rows[0].Cells[0])).To(Equal(ch.Key()))
		Expect(channelOf(res.Rows[0].Cells[1])).To(BeZero())
	})

	It(
		"Should name only the task when a task, a schematic, and a plot use a channel",
		func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx)
			dev := device.Device{
				Key:      UniqueChannelName(),
				Rack:     layer.Rack.EmbeddedKey,
				Name:     "plc",
				Make:     "modbus",
				Model:    "modbus",
				Location: "127.0.0.1:502",
			}
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = ch.Key()
			sch := schematic.Schematic{
				Name:    "pid",
				Project: &proj.Key,
				Nodes: []schematic.Node{{
					Key:    "value",
					Config: schematic.NodeConfig{Variant: value},
				}},
			}
			plot := defaults.Fill(lineplot.LinePlot{Name: "plot", Project: &proj.Key})
			plot.Channels.Y1 = []channel.Key{ch.Key()}
			tsk := task.Task{
				Name: "modbus_read",
				Type: "modbus_read",
				Config: msgpack.EncodedJSON{
					"device": dev.Key,
					"channels": []any{map[string]any{
						"type":    "coil",
						"key":     "coil-1",
						"channel": ch.Key(),
					}},
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := layer.Device.NewWriter(tx).Create(ctx, &dev); err != nil {
					return err
				}
				if err := layer.Schematic.NewWriter(tx).Create(ctx, &sch); err != nil {
					return err
				}
				if err := layer.LinePlot.NewWriter(tx).Create(ctx, &plot); err != nil {
					return err
				}
				return layer.Task.NewWriter(tx).Create(ctx, &tsk)
			})).To(Succeed())
			Expect(deleteChannel(ctx, ch.Key())).To(MatchError(gorp.RestrictError{
				Target:    gorp.NewTarget("channel", ch.Key()),
				Holder:    gorp.NewTarget("modbus_read_config", tsk.Key),
				Reference: "channels.channel",
			}))
			var stored lineplot.LinePlot
			Expect(layer.LinePlot.NewRetrieve().
				Where(lineplot.MatchKeys(plot.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Channels.Y1).To(Equal([]channel.Key{ch.Key()}))
		},
	)

	It("Should list a valve as a reader of its state and a writer of its command", func(
		ctx SpecContext,
	) {
		state, command := createChannel(ctx), createChannel(ctx)
		var valve schematic.ValveNodeConfig
		valve.SetDefaults()
		valve.StateChannel, valve.CommandChannel = state.Key(), command.Key()
		sch := schematic.Schematic{
			Name:    "pid",
			Project: &proj.Key,
			Nodes: []schematic.Node{{
				Key:    "valve",
				Config: schematic.NodeConfig{Variant: valve},
			}},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		uses := MustSucceed(layer.Channel.Uses(
			ctx,
			db,
			[]channel.Key{state.Key(), command.Key()},
		))
		holder := gorp.NewTarget("schematic", sch.Key)
		Expect(uses).To(Equal([][]channel.Use{
			{{Holder: holder, Direction: channel.DirectionRead}},
			{{Holder: holder, Direction: channel.DirectionWrite}},
		}))
	})

	It("Should match the channels that a record reads or writes as used", func(
		ctx SpecContext,
	) {
		read, state, command, unused := createChannel(ctx),
			createChannel(ctx),
			createChannel(ctx),
			createChannel(ctx)
		var valve schematic.ValveNodeConfig
		valve.SetDefaults()
		valve.StateChannel, valve.CommandChannel = state.Key(), command.Key()
		sch := schematic.Schematic{
			Name:    "pid",
			Project: &proj.Key,
			Nodes: []schematic.Node{{
				Key:    "valve",
				Config: schematic.NodeConfig{Variant: valve},
			}},
		}
		t := valueTable(read.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Schematic.NewWriter(tx).Create(ctx, &sch); err != nil {
				return err
			}
			return layer.Table.NewWriter(tx).Create(ctx, &t)
		})).To(Succeed())
		keys := []channel.Key{read.Key(), state.Key(), command.Key(), unused.Key()}
		retrieve := func(used bool) []channel.Key {
			var found []channel.Channel
			Expect(layer.Channel.NewRetrieve().
				Where(channel.MatchKeys(keys...)).
				Where(channel.MatchUsed(used)).
				Where(channel.MatchInternal(false)).
				Entries(&found).
				Exec(ctx, db)).To(Succeed())
			return channel.KeysFromChannels(found)
		}
		Expect(retrieve(true)).To(ConsistOf(read.Key(), state.Key(), command.Key()))
		Expect(retrieve(false)).To(ConsistOf(unused.Key()))
	})

	It("Should leave a built-in channel out of an Arc symbol search", func(
		ctx SpecContext,
	) {
		prefix := "search_" + UniqueChannelName()
		builtIn := channel.Channel{
			Name:     prefix + "_builtin",
			DataType: telem.JSONT,
			Virtual:  true,
		}
		user := channel.Channel{
			Name:     prefix + "_user",
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := layer.Channel.NewWriter(tx)
			if err := w.Declare(ctx, &builtIn); err != nil {
				return err
			}
			return w.Create(ctx, &user)
		})).To(Succeed())
		r := layer.Channel.NewArcSymbolResolver(db)
		Eventually(func(g Gomega) {
			g.Expect(r.Search(ctx, prefix)).
				To(ContainElement(HaveField("ID", int(user.Key()))))
		}).Should(Succeed())
		Expect(r.Search(ctx, prefix)).
			ToNot(ContainElement(HaveField("ID", int(builtIn.Key()))))
	})

	It(
		"Should keep the weak uses of a deleted channel and drop the detached ones",
		func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx)
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = ch.Key()
			sch := schematic.Schematic{
				Name:    "pid",
				Project: &proj.Key,
				Nodes: []schematic.Node{{
					Key:    "value",
					Config: schematic.NodeConfig{Variant: value},
				}},
			}
			plot := defaults.Fill(lineplot.LinePlot{Name: "plot", Project: &proj.Key})
			plot.Channels.Y1 = []channel.Key{ch.Key()}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := layer.Schematic.NewWriter(tx).Create(ctx, &sch); err != nil {
					return err
				}
				return layer.LinePlot.NewWriter(tx).Create(ctx, &plot)
			})).To(Succeed())
			Expect(deleteChannel(ctx, ch.Key())).To(Succeed())
			uses := MustSucceed(layer.Channel.Uses(ctx, db, []channel.Key{ch.Key()}))
			Expect(uses[0]).To(ConsistOf(channel.Use{
				Holder:    gorp.NewTarget("schematic", sch.Key),
				Direction: channel.DirectionRead,
			}))
		},
	)
})
