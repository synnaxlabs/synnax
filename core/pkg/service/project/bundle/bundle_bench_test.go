// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bundle_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/defaults"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// benchWorkspace is a project whose schematics, line plots, logs, and tables link 5,000
// channels in folders three deep.
type benchWorkspace struct {
	ctx     context.Context
	db      *gorp.DB
	layer   *service.Layer
	project project.Key
}

// openBenchWorkspace opens a Core and creates the workspace in it, closed when b ends.
func openBenchWorkspace(b *testing.B) benchWorkspace {
	b.Helper()
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	node := disttestutil.OpenNode(ctx)
	layer := MustSucceed(svctestutil.OpenLayer(ctx, node))
	b.Cleanup(func() {
		gomega.Expect(layer.Close()).To(gomega.Succeed())
		gomega.Expect(node.Close()).To(gomega.Succeed())
	})
	w := benchWorkspace{ctx: ctx, db: node.DB, layer: layer}
	keys := w.createChannels(b)
	proj := project.Project{Name: "bench"}
	gomega.Expect(w.db.WithTx(ctx, func(tx gorp.Tx) error {
		return layer.Project.NewWriter(tx).Create(ctx, &proj)
	})).To(gomega.Succeed())
	w.project = proj.Key
	w.createVisualizations(b, keys)
	return w
}

// createChannels creates 10 channels in each of 500 folders, 10 x 10 x 5 deep.
func (w benchWorkspace) createChannels(b *testing.B) []channel.Key {
	b.Helper()
	keys := make([]channel.Key, 0, 5000)
	gomega.Expect(w.db.WithTx(w.ctx, func(tx gorp.Tx) error {
		groups := w.layer.Group.NewWriter(tx)
		folder := func(name string, parent *group.Group) (group.Group, error) {
			g := group.Group{Name: name, Scope: resource.TypeChannel}
			if parent != nil {
				g.Parent = new(parent.ResourceID())
			}
			return g, groups.Create(w.ctx, &g)
		}
		for a := range 10 {
			top, err := folder(fmt.Sprintf("area_%d", a), nil)
			if err != nil {
				return err
			}
			for s := range 10 {
				mid, err := folder(fmt.Sprintf("system_%d", s), &top)
				if err != nil {
					return err
				}
				for l := range 5 {
					leaf, err := folder(fmt.Sprintf("line_%d", l), &mid)
					if err != nil {
						return err
					}
					prefix := fmt.Sprintf("ch_%d_%d_%d", a, s, l)
					created, err := w.createLeafChannels(tx, leaf, prefix)
					if err != nil {
						return err
					}
					keys = append(keys, created...)
				}
			}
		}
		return nil
	})).To(gomega.Succeed())
	return keys
}

// createLeafChannels creates 10 channels named prefix_<i> in the folder leaf.
func (w benchWorkspace) createLeafChannels(
	tx gorp.Tx,
	leaf group.Group,
	prefix string,
) ([]channel.Key, error) {
	chs := make([]channel.Channel, 10)
	for i := range chs {
		chs[i] = channel.Channel{
			Name:     fmt.Sprintf("%s_%d", prefix, i),
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &leaf.Key,
		}
	}
	if err := w.layer.Channel.NewWriter(tx).CreateMany(w.ctx, &chs); err != nil {
		return nil, err
	}
	return channel.KeysFromChannels(chs), nil
}

// createVisualizations creates 50 schematics with 100 links each, 20 line plots with 20
// channels each, 10 logs with 20 channels each, and 10 tables with 50 cells each.
func (w benchWorkspace) createVisualizations(b *testing.B, keys []channel.Key) {
	b.Helper()
	next := 0
	take := func() channel.Key {
		k := keys[next%len(keys)]
		next++
		return k
	}
	gomega.Expect(w.db.WithTx(w.ctx, func(tx gorp.Tx) error {
		for s := range 50 {
			if err := w.layer.Schematic.NewWriter(tx).Create(
				w.ctx,
				new(benchSchematic(w.project, s, take)),
			); err != nil {
				return err
			}
		}
		for p := range 20 {
			lp := defaults.Fill(lineplot.LinePlot{
				Name:    fmt.Sprintf("plot_%d", p),
				Project: &w.project,
			})
			for range 20 {
				lp.Channels.Y1 = append(lp.Channels.Y1, take())
			}
			if err := w.layer.LinePlot.NewWriter(tx).Create(w.ctx, &lp); err != nil {
				return err
			}
		}
		for l := range 10 {
			if err := w.layer.Log.NewWriter(tx).
				Create(w.ctx, new(benchLog(w.project, l, take))); err != nil {
				return err
			}
		}
		for t := range 10 {
			if err := w.layer.Table.NewWriter(tx).
				Create(w.ctx, new(benchTable(w.project, t, take))); err != nil {
				return err
			}
		}
		return nil
	})).To(gomega.Succeed())
}

func benchSchematic(
	proj project.Key,
	n int,
	take func() channel.Key,
) schematic.Schematic {
	sch := schematic.Schematic{Name: fmt.Sprintf("schematic_%d", n), Project: &proj}
	for i := range 100 {
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = take()
		sch.Nodes = append(sch.Nodes, schematic.Node{
			Key:    fmt.Sprintf("value_%d", i),
			Config: schematic.NodeConfig{Variant: value},
		})
	}
	return sch
}

func benchLog(proj project.Key, n int, take func() channel.Key) log.Log {
	l := defaults.Fill(log.Log{Name: fmt.Sprintf("log_%d", n), Project: &proj})
	for range 20 {
		l.Channels = append(l.Channels, defaults.Fill(log.ChannelEntry{
			Channel: take(),
			Color: &color.Ref{
				Variant: color.FixedRef{Value: MustSucceed(xcolor.Parse("#ff0000"))},
			},
			Notation: notation.NotationStandard,
		}))
	}
	return l
}

func benchTable(proj project.Key, n int, take func() channel.Key) table.Table {
	t := table.Table{Name: fmt.Sprintf("table_%d", n), Project: &proj}
	for c := range 5 {
		t.Columns = append(
			t.Columns,
			table.Column{Key: fmt.Sprintf("column_%d", c), Size: 72},
		)
	}
	for r := range 10 {
		row := table.Row{Key: fmt.Sprintf("row_%d", r), Size: 36}
		for c := range 5 {
			var cell table.ValueCellConfig
			cell.SetDefaults()
			cell.Channel = take()
			row.Cells = append(row.Cells, table.Cell{
				Key:    fmt.Sprintf("cell_%d_%d", r, c),
				Config: table.CellConfig{Variant: cell},
			})
		}
		t.Rows = append(t.Rows, row)
	}
	return t
}

// BenchmarkExport measures the export of a project bundle that links 5,000 channels.
func BenchmarkExport(b *testing.B) {
	w := openBenchWorkspace(b)
	b.ReportAllocs()
	for b.Loop() {
		MustSucceed2(
			w.layer.Bundle.Export(w.ctx, w.db, w.project, xjson.Codec, readAll),
		)
	}
}

// BenchmarkImport measures the decode and import of that bundle. Each import rolls
// back, so the Core keeps one copy.
func BenchmarkImport(b *testing.B) {
	w := openBenchWorkspace(b)
	files, _ := MustSucceed2(
		w.layer.Bundle.Export(w.ctx, w.db, w.project, xjson.Codec, readAll),
	)
	b.ReportAllocs()
	for b.Loop() {
		bundle := MustSucceed(w.layer.Bundle.Decode(w.ctx, files, "bench"))
		tx := w.db.OpenTx()
		MustSucceed(w.layer.Bundle.Import(w.ctx, tx, bundle))
		gomega.Expect(tx.Close()).To(gomega.Succeed())
	}
}
