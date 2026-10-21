// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/telem"
)

func must(b *testing.B, err error) {
	b.Helper()
	if err != nil {
		b.Fatal(err)
	}
}

func closeLater(b *testing.B, c io.Closer, err error) {
	b.Helper()
	must(b, err)
	b.Cleanup(func() { must(b, c.Close()) })
}

type benchStack struct {
	svc *arc.Service
	otg *ontology.Ontology
}

// openBenchStack opens an Arc service and what it needs on an in-memory node. The task
// sync debounce is long enough that no resync runs during a benchmark. The mock node
// asserts with Gomega, which reports to b.
func openBenchStack(b *testing.B) benchStack {
	b.Helper()
	gomega.RegisterTestingT(b)
	ctx := context.Background()
	node := mock.OpenNode(ctx)
	b.Cleanup(func() { must(b, node.Close()) })
	otg, err := ontology.Open(ctx, ontology.Config{DB: node.DB})
	closeLater(b, otg, err)
	idx, err := search.OpenIndex()
	closeLater(b, idx, err)
	groupSvc, err := group.OpenService(ctx, group.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Search:   idx,
	})
	closeLater(b, groupSvc, err)
	labelSvc, err := label.OpenService(ctx, label.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		Search:   idx,
	})
	closeLater(b, labelSvc, err)
	statusSvc, err := status.OpenService(ctx, status.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		Label:    labelSvc,
		Search:   idx,
	})
	closeLater(b, statusSvc, err)
	channelSvc, err := channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       idx,
		Status:       statusSvc,
	})
	closeLater(b, channelSvc, err)
	rackSvc, err := rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  node.DB,
		Ontology:            otg,
		Group:               groupSvc,
		HostProvider:        mock.NewStaticHostProvider(1),
		Status:              statusSvc,
		HealthCheckInterval: telem.Hour,
		Search:              idx,
	})
	closeLater(b, rackSvc, err)
	imexSvc := imex.NewService()
	arcTaskSvc, err := arctask.OpenService(ctx, arctask.ServiceConfig{DB: node.DB})
	closeLater(b, arcTaskSvc, err)
	configs, err := taskconfig.NewRegistry(arcTaskSvc.Stores()...)
	must(b, err)
	taskSvc, err := task.OpenService(ctx, task.ServiceConfig{
		DB:           node.DB,
		Ontology:     otg,
		Group:        groupSvc,
		Rack:         rackSvc,
		Status:       statusSvc,
		Search:       idx,
		ImEx:         imexSvc,
		Configs:      configs,
		ImExExcluded: []string{arctask.Type},
	})
	closeLater(b, taskSvc, err)
	svc, err := arc.OpenService(ctx, arc.ServiceConfig{
		DB:                  node.DB,
		Ontology:            otg,
		Group:               groupSvc,
		Channel:             channelSvc,
		Task:                taskSvc,
		Status:              statusSvc,
		Search:              idx,
		ImEx:                imexSvc,
		TextSweepQuiescence: telem.Hour,
		TextSweepThreshold:  1 << 20,
		TaskSyncDebounce:    telem.Hour,
	})
	closeLater(b, svc, err)
	return benchStack{svc: svc, otg: otg}
}

// createArcs creates a chain of depth modules, each importing the one before it, and n
// automations. At every module, per of them import it. It returns the first module,
// the one every other module and importer depends on.
func (s benchStack) createArcs(b *testing.B, n, per, depth int) arc.Arc {
	b.Helper()
	ctx := context.Background()
	w := s.svc.NewWriter(nil)
	var root arc.Arc
	for k := range depth {
		module := arc.Arc{
			Name: fmt.Sprintf("m%d", k),
			Type: arc.TypeModule,
			Mode: arc.ModeText,
			Text: newText("export func f() i64 { return 1 }\n"),
		}
		if k > 0 {
			module.Text = newText(fmt.Sprintf(
				"import m%d\n\nexport func f() i64 { return m%d.f() }\n", k-1, k-1,
			))
		}
		must(b, w.Create(ctx, &module))
		if k == 0 {
			root = module
		}
	}
	for i := range n {
		a := arc.Arc{
			Name: fmt.Sprintf("arc_%d", i),
			Type: arc.TypeAutomation,
			Mode: arc.ModeText,
			Text: newText("valve -> chamber\n"),
		}
		if i < per*depth {
			m := i / per
			a.Text = newText(fmt.Sprintf("import m%d\n\nvalve -> m%d.f{}\n", m, m))
		}
		must(b, w.Create(ctx, &a))
	}
	return root
}

// resyncWork is what one resync pays per Arc: a parse to find its imports and the
// content hash.
func resyncWork(b *testing.B, a arc.Arc) {
	b.Helper()
	t, diag := arctext.Parse(a.Text.Materialize(), parser.Config{})
	if diag != nil {
		b.Fatal(diag)
	}
	_ = parser.Imports(t.AST)
	if _, err := arc.Hash(a, nil); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkImporters compares two ways to find the Arcs to resync after an edit to the
// first module: resyncing every Arc, and walking the module's incoming imports edges
// transitively. Both then pay the per-Arc resync work for each Arc they found. n is
// the automation count, depth the module chain length, per the importers per module,
// so I = (depth - 1) + per * depth Arcs depend on the first module. Each cell opens its
// own stack and closes it before the next.
func BenchmarkImporters(b *testing.B) {
	ctx := context.Background()
	for _, n := range []int{10, 100, 1000, 10000} {
		for _, depth := range []int{1, 4, 8} {
			for _, per := range []int{1, 5, 25, 100} {
				if per*depth > n {
					continue
				}
				name := fmt.Sprintf("n=%d/depth=%d/per=%d", n, depth, per)
				b.Run(name, func(b *testing.B) {
					stack := openBenchStack(b)
					root := stack.createArcs(b, n, per, depth)
					b.Run("resync-all", func(b *testing.B) {
						b.ReportAllocs()
						for b.Loop() {
							var arcs []arc.Arc
							must(b, stack.svc.NewRetrieve().
								Entries(&arcs).
								Exec(ctx, nil))
							for _, a := range arcs {
								resyncWork(b, a)
							}
						}
					})
					b.Run("imports-edge", func(b *testing.B) {
						b.ReportAllocs()
						for b.Loop() {
							keys, err := stack.svc.Importers(ctx, nil, root.Key)
							must(b, err)
							var arcs []arc.Arc
							must(b, stack.svc.NewRetrieve().
								Where(arc.MatchKeys(keys...)).
								Entries(&arcs).
								Exec(ctx, nil))
							for _, a := range arcs {
								resyncWork(b, a)
							}
						}
					})
				})
			}
		}
	}
}
