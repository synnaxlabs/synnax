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
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Tombstone break", Ordered, ContinueOnFailure, func() {
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
		proj = project.Project{Name: "Tombstone break"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	DescribeTable(
		"Should not count a tombstone as a record that keeps the channel",
		func(
			ctx SpecContext,
			held bool,
		) {
			past := channel.Channel{
				Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &past)
			})).To(Succeed())
			ch := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
				Rewires:  []channel.Rewire{{From: past.Key(), At: telem.Now()}},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
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
			if held {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
				})).To(Succeed())
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
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
						"channel": past.Key(),
					}},
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := layer.Device.NewWriter(tx).Create(ctx, &dev); err != nil {
					return err
				}
				return layer.Task.NewWriter(tx).Create(ctx, &tsk)
			})).To(Succeed())
			var p impact.Plan
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				p, err = layer.Impact.Apply(
					ctx,
					tx,
					impact.Change{Variant: impact.DeleteChannelsChange{
						Keys: []channel.Key{past.Key()},
					}},
					impact.ApplyOptions{Picks: map[string]impact.Choice{
						task.ResourceID(tsk.Key).String(): impact.ChoiceDelete,
					}},
				)
				return err
			})).To(Succeed())
			Expect(p.Kept).To(BeZero())
		},
		Entry(
			"a schematic held the deleted channel, so a tombstone holds its rewires",
			true,
		),
		Entry("nothing held the deleted channel, so no tombstone", false),
	)
})
