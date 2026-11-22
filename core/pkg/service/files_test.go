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
	"encoding/json/v2"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// numbers returns every number in the decoded JSON value v.
func numbers(v any) []float64 {
	switch v := v.(type) {
	case float64:
		return []float64{v}
	case []any:
		var out []float64
		for _, e := range v {
			out = append(out, numbers(e)...)
		}
		return out
	case map[string]any:
		var out []float64
		for _, e := range v {
			out = append(out, numbers(e)...)
		}
		return out
	}
	return nil
}

// inProject returns the one entry of entries whose project is proj.
func inProject[E any](entries []E, project func(E) *project.Key, proj project.Key) E {
	GinkgoHelper()
	var found []E
	for _, e := range entries {
		if p := project(e); p != nil && *p == proj {
			found = append(found, e)
		}
	}
	Expect(found).To(HaveLen(1))
	return found[0]
}

var _ = Describe("Channel paths in files", Ordered, func() {
	var (
		l    *service.Layer
		db   *gorp.DB
		proj project.Project
		key  channel.Key
		path string
		tsk  task.Task
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		l = MustOpen(svctestutil.OpenLayer(ctx, node))
		folder, name := UniqueChannelName(), UniqueChannelName()
		path = folder + "." + name
		proj = project.Project{Name: "Paths"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := l.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
				return err
			}
			g := group.Group{Name: folder, Scope: resource.TypeChannel}
			if err := l.Group.NewWriter(tx).Create(ctx, &g); err != nil {
				return err
			}
			ch := channel.Channel{
				Name:     name,
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    new(g.Key),
			}
			if err := l.Channel.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			key = ch.Key()
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = key
			sch := schematic.Schematic{
				Name:    "Schematic",
				Project: &proj.Key,
				Nodes: []schematic.Node{
					{Key: "value", Config: schematic.NodeConfig{Variant: value}},
				},
			}
			if err := l.Schematic.NewWriter(tx).Create(ctx, &sch); err != nil {
				return err
			}
			tbl := table.Table{
				Name:    "Table",
				Project: &proj.Key,
				Columns: []table.Column{{Key: "c0", Size: 80}},
				Rows: []table.Row{{Key: "r0", Size: 30, Cells: []table.Cell{{
					Key: "cell",
					Config: defaults.Fill(table.CellConfig{
						Variant: table.ValueCellConfig{Channel: key},
					}),
				}}}},
			}
			if err := l.Table.NewWriter(tx).Create(ctx, &tbl); err != nil {
				return err
			}
			lp := defaults.Fill(lineplot.LinePlot{Name: "Plot", Project: &proj.Key})
			lp.Channels.Y1 = []channel.Key{key}
			if err := l.LinePlot.NewWriter(tx).Create(ctx, &lp); err != nil {
				return err
			}
			lg := defaults.Fill(log.Log{Name: "Log", Project: &proj.Key})
			lg.Channels = []log.ChannelEntry{defaults.Fill(log.ChannelEntry{
				Channel:  key,
				Notation: notation.NotationStandard,
			})}
			if err := l.Log.NewWriter(tx).Create(ctx, &lg); err != nil {
				return err
			}
			dev := device.Device{
				Key:      UniqueChannelName(),
				Rack:     l.Rack.EmbeddedKey,
				Name:     "plc",
				Make:     "modbus",
				Model:    "modbus",
				Location: "127.0.0.1:502",
			}
			if err := l.Device.NewWriter(tx).Create(ctx, &dev); err != nil {
				return err
			}
			tsk = task.Task{
				Name: "Read",
				Type: "modbus_read",
				Config: msgpack.EncodedJSON{
					"device": dev.Key,
					"channels": []any{map[string]any{
						"type":    "coil",
						"key":     "coil-1",
						"channel": key,
					}},
				},
			}
			if err := l.Task.NewWriter(tx).Create(ctx, &tsk); err != nil {
				return err
			}
			pn := panel.Panel{
				Name:    "Panel",
				Project: &proj.Key,
				Root: panel.Node{Variant: panel.LeafNode{Tabs: []panel.Tab{{
					Variant: panel.ResourceTab{
						Key:      uuid.New(),
						Resource: tsk.ResourceID(),
					},
				}}}},
			}
			return l.Panel.NewWriter(tx).Create(ctx, &pn)
		})).To(Succeed())
	})

	It("Should name the channel by its path in every member", func(ctx SpecContext) {
		files, _ := MustSucceed2(l.Bundle.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(7))
		for p, data := range files {
			var body map[string]any
			Expect(json.Unmarshal(data, &body)).To(Succeed())
			if body["type"] == "project" || body["type"] == "panel" {
				continue
			}
			By(p)
			Expect(string(data)).To(ContainSubstring(`"` + path + `"`))
			Expect(numbers(body)).ToNot(ContainElement(BeNumerically("==", key)))
		}
	})

	It("Should bind the channel of every member again on import", func(
		ctx SpecContext,
	) {
		files, _ := MustSucceed2(l.Bundle.Export(ctx, db, proj.Key, xjson.Codec))
		b := MustSucceed(l.Bundle.Decode(ctx, files, "Paths.zip"))
		tx := DeferClose(db.OpenTx())
		imported := MustSucceed(l.Bundle.Import(ctx, tx, b)).Key
		Expect(imported).ToNot(Equal(proj.Key))

		var schs []schematic.Schematic
		Expect(l.Schematic.NewRetrieve().Entries(&schs).Exec(ctx, tx)).To(Succeed())
		sch := inProject(schs, func(s schematic.Schematic) *project.Key {
			return s.Project
		}, imported)
		value, ok := sch.Nodes[0].Config.Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(value.Channel).To(Equal(key))

		var tbls []table.Table
		Expect(l.Table.NewRetrieve().Entries(&tbls).Exec(ctx, tx)).To(Succeed())
		tbl := inProject(tbls, func(t table.Table) *project.Key {
			return t.Project
		}, imported)
		cell, ok := tbl.Rows[0].Cells[0].Config.Variant.(table.ValueCellConfig)
		Expect(ok).To(BeTrue())
		Expect(cell.Channel).To(Equal(key))

		var lps []lineplot.LinePlot
		Expect(l.LinePlot.NewRetrieve().Entries(&lps).Exec(ctx, tx)).To(Succeed())
		Expect(inProject(lps, func(p lineplot.LinePlot) *project.Key {
			return p.Project
		}, imported).Channels.Y1).To(Equal([]channel.Key{key}))

		var lgs []log.Log
		Expect(l.Log.NewRetrieve().Entries(&lgs).Exec(ctx, tx)).To(Succeed())
		Expect(inProject(lgs, func(lg log.Log) *project.Key {
			return lg.Project
		}, imported).Channels[0].Channel).To(Equal(key))

		var tasks []task.Task
		Expect(l.Task.NewRetrieve().
			Where(task.MatchTypes("modbus_read")).
			Entries(&tasks).
			Exec(ctx, tx)).To(Succeed())
		tasks = slices.DeleteFunc(tasks, func(t task.Task) bool {
			return t.Key == tsk.Key
		})
		Expect(tasks).To(HaveLen(1))
		Expect(tasks[0].Config["channels"]).To(ConsistOf(
			HaveKeyWithValue("channel", BeNumerically("==", key)),
		))
	})
})
