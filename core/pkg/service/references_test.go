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
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	schematicversions "github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("References", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
	})

	createTask := func(ctx SpecContext, typ string, cfg msgpack.EncodedJSON) task.Task {
		GinkgoHelper()
		t := task.Task{Name: typ, Type: typ, Config: cfg}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Task.NewWriter(tx).Create(ctx, &t)
		})).To(Succeed())
		return t
	}

	Describe("Task configs", func() {
		var (
			ch  channel.Channel
			dev device.Device
			tsk task.Task
		)
		BeforeEach(func(ctx SpecContext) {
			ch = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			dev = device.Device{
				Key:      UniqueChannelName(),
				Rack:     layer.Rack.EmbeddedKey,
				Name:     "plc",
				Make:     "modbus",
				Model:    "modbus",
				Location: "127.0.0.1:502",
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Device.NewWriter(tx).Create(ctx, &dev)
			})).To(Succeed())
			tsk = createTask(ctx, "modbus_read", msgpack.EncodedJSON{
				"device": dev.Key,
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "coil-1",
					"channel": ch.Key(),
				}},
			})
		})

		It("Should refuse deleting a channel that a task config names", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(
				MatchError(gorp.RestrictError{
					Target:    gorp.NewTarget("channel", ch.Key()),
					Holder:    gorp.NewTarget("modbus_read_config", tsk.Key),
					Reference: "channels.channel",
				}),
			)
		})

		It("Should refuse deleting a device that a task config names", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Device.NewWriter(tx).Delete(ctx, dev.Key)
			})).To(
				MatchError(gorp.RestrictError{
					Target:    gorp.NewTarget("device", dev.Key),
					Holder:    gorp.NewTarget("modbus_read_config", tsk.Key),
					Reference: "device",
				}),
			)
		})

		It("Should delete the channel and device once the task is gone", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Task.NewWriter(tx).Delete(ctx, tsk.Key, false)
			})).
				To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).
				To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Device.NewWriter(tx).Delete(ctx, dev.Key)
			})).To(Succeed())
		})
	})

	Describe("Uses", func() {
		It("Should list each record that reads or writes a channel", func(
			ctx SpecContext,
		) {
			proj := project.Project{Name: "Uses"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Project.NewWriter(tx).Create(ctx, &proj)
			})).To(Succeed())
			ch := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			dev := device.Device{
				Key:      UniqueChannelName(),
				Rack:     layer.Rack.EmbeddedKey,
				Name:     "plc",
				Make:     "modbus",
				Model:    "modbus",
				Location: "127.0.0.1:502",
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Device.NewWriter(tx).Create(ctx, &dev)
			})).To(Succeed())
			tsk := createTask(ctx, "modbus_read", msgpack.EncodedJSON{
				"device": dev.Key,
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "coil-1",
					"channel": ch.Key(),
				}},
			})
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
			lg := defaults.Fill(log.Log{Name: "log", Project: &proj.Key})
			lg.Channels = []log.ChannelEntry{
				defaults.Fill(log.ChannelEntry{Channel: ch.Key()}),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := layer.Schematic.NewWriter(tx).Create(ctx, &sch); err != nil {
					return err
				}
				if err := layer.LinePlot.NewWriter(tx).Create(ctx, &plot); err != nil {
					return err
				}
				return layer.Log.NewWriter(tx).Create(ctx, &lg)
			})).To(Succeed())
			Expect(layer.Channel.Uses(ctx, db, ch.Key())).To(ConsistOf(
				channel.Use{
					Holder:    gorp.NewTarget("modbus_read_config", tsk.Key),
					Direction: channel.DirectionWrite,
				},
				channel.Use{
					Holder:    gorp.NewTarget("schematic", sch.Key),
					Direction: channel.DirectionRead,
				},
				channel.Use{
					Holder:    gorp.NewTarget("lineplot", plot.Key),
					Direction: channel.DirectionRead,
				},
				channel.Use{
					Holder:    gorp.NewTarget("log", lg.Key),
					Direction: channel.DirectionRead,
				},
			))
		})
	})

	Describe("Schematic and table channels", func() {
		var (
			ch   channel.Channel
			proj project.Project
		)
		BeforeEach(func(ctx SpecContext) {
			proj = project.Project{Name: "Visuals"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Project.NewWriter(tx).Create(ctx, &proj)
			})).To(Succeed())
			ch = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
		})

		deleteChannel := func(ctx SpecContext) {
			GinkgoHelper()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
		}

		valueSchematic := func(key channel.Key) schematic.Schematic {
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = key
			return schematic.Schematic{
				Name:    "pid",
				Project: &proj.Key,
				Nodes: []schematic.Node{{
					Key:    "value",
					Config: schematic.NodeConfig{Variant: value},
				}},
			}
		}

		It("Should keep a schematic and its key when its channel is deleted", func(
			ctx SpecContext,
		) {
			s := valueSchematic(ch.Key())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Schematic.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			deleteChannel(ctx)
			var res schematic.Schematic
			Expect(layer.Schematic.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			value, ok := res.Nodes[0].Config.Variant.(schematic.ValueNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(value.Channel).To(Equal(ch.Key()))
		})

		It("Should refuse a schematic that names a channel key never issued", func(
			ctx SpecContext,
		) {
			key := distchannel.NewKey(ch.Leaseholder, ch.LocalKey+1000)
			s := valueSchematic(key)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Schematic.NewWriter(tx).Create(ctx, &s)
			})).To(MatchError(ContainSubstring(
				"channel " + key.String() + " was never issued",
			)))
		})

		It("Should clear a never-issued key on import and keep a deleted one", func(
			ctx SpecContext,
		) {
			dead := valueSchematic(ch.Key())
			deleteChannel(ctx)
			never := valueSchematic(
				distchannel.NewKey(ch.Leaseholder, ch.LocalKey+1000),
			)
			dead.Nodes = append(dead.Nodes, never.Nodes[0])
			dead.Nodes[1].Key = "never"
			env := imex.Envelope{
				Version: schematicversions.Latest,
				Type:    string(resource.TypeSchematic),
				Name:    dead.Name,
			}
			Expect(env.Encode(dead)).To(Succeed())
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
			var res schematic.Schematic
			Expect(layer.Schematic.NewRetrieve().
				Where(schematic.MatchKeys(MustSucceed(uuid.Parse(id.Key)))).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			channelOf := func(n schematic.Node) channel.Key {
				GinkgoHelper()
				value, ok := n.Config.Variant.(schematic.ValueNodeConfig)
				Expect(ok).To(BeTrue())
				return value.Channel
			}
			Expect(channelOf(res.Nodes[0])).To(Equal(ch.Key()))
			Expect(channelOf(res.Nodes[1])).To(BeZero())
		})

		It("Should keep a table cell and its key when its channel is deleted", func(
			ctx SpecContext,
		) {
			var cell table.ValueCellConfig
			cell.SetDefaults()
			cell.Channel = ch.Key()
			t := table.Table{
				Name:    "readings",
				Project: &proj.Key,
				Columns: []table.Column{{Key: "column", Size: 72}},
				Rows: []table.Row{{
					Key:  "row",
					Size: 36,
					Cells: []table.Cell{{
						Key:    "cell",
						Config: table.CellConfig{Variant: cell},
					}},
				}},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Table.NewWriter(tx).Create(ctx, &t)
			})).To(Succeed())
			deleteChannel(ctx)
			var res table.Table
			Expect(layer.Table.NewRetrieve().
				Where(table.MatchKeys(t.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			stored, ok := res.Rows[0].Cells[0].Config.Variant.(table.ValueCellConfig)
			Expect(ok).To(BeTrue())
			Expect(stored.Channel).To(Equal(ch.Key()))
		})
	})

	It(
		"Should clear the status of a PagerDuty alert and keep the alert",
		func(ctx SpecContext) {
			st := status.Status[any]{
				Key:     UniqueChannelName(),
				Name:    "pressure",
				Variant: status.VariantWarning,
				Time:    telem.Now(),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Status.NewWriter(tx).Set(ctx, &st)
			})).To(Succeed())
			tsk := createTask(ctx, pagerduty.AlertTaskType, msgpack.EncodedJSON{
				"alerts": []any{map[string]any{
					"key":    "alert-1",
					"status": st.Key,
				}},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Status.NewWriter(tx).Delete(ctx, st.Key)
			})).To(Succeed())
			var res task.Task
			Expect(layer.Task.NewRetrieve().
				Where(task.MatchKeys(tsk.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.Config["alerts"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("key", "alert-1"),
				HaveKeyWithValue("status", ""),
			)))
		},
	)
})
