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
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// benchService opens an Arc service over a fresh node with 2n float channels,
// bench_in_i
// and bench_out_i for each i below n, and returns it with the keys of the channels.
func benchService(b *testing.B, n int) (*arc.Service, *gorp.DB, []channel.Key) {
	b.Helper()
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	node := testutil.OpenNode(ctx)
	statuses := MustSucceed(status.OpenService(ctx, status.ServiceConfig{DB: node.DB}))
	channels := MustSucceed(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.Storage.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statuses,
	}))
	roles := MustSucceed(role.OpenService(ctx, role.ServiceConfig{DB: node.DB}))
	racks := MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  node.DB,
		HostKey:             1,
		Status:              statuses,
		HealthCheckInterval: telem.Second,
		DriverRole: MustSucceed(
			racktestutil.CreateDriverRole(ctx, node.DB, roles),
		),
	}))
	table := MustSucceed(arc.OpenTable(ctx, arc.TableConfig{DB: node.DB}))
	arcTasks := MustSucceed(
		arctask.OpenService(ctx, arctask.ServiceConfig{DB: node.DB}),
	)
	imexSvc := imex.NewService()
	tasks := MustSucceed(task.OpenService(ctx, task.ServiceConfig{
		DB:           node.DB,
		Rack:         racks,
		Status:       statuses,
		ImEx:         imexSvc,
		Configs:      MustSucceed(taskconfig.NewRegistry(arcTasks.Stores()...)),
		ImExExcluded: []string{arctask.Type},
	}))
	svc := MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
		DB:               node.DB,
		Table:            table,
		Channel:          channels,
		Task:             tasks,
		MatchTasks:       arcTasks.MatchArcs,
		Status:           statuses,
		ImEx:             imexSvc,
		TaskSyncDebounce: telem.Second,
	}))
	b.Cleanup(func() {
		if err := (io.MultiCloser{
			node, statuses, channels, roles, racks, table, arcTasks, tasks, svc,
		}).Close(); err != nil {
			b.Error(err)
		}
	})
	return svc, node.DB, createBenchChannels(b, node.DB, channels, n)
}

// createBenchChannels creates bench_in_i and bench_out_i for each i below n and returns
// their keys in that order.
func createBenchChannels(
	b *testing.B,
	db *gorp.DB,
	channels *channel.Service,
	n int,
) []channel.Key {
	b.Helper()
	ctx := b.Context()
	keys := make([]channel.Key, 0, 2*n)
	if err := db.WithTx(ctx, func(tx gorp.Tx) error {
		for i := range n {
			for _, name := range []string{"bench_in_", "bench_out_"} {
				ch := channel.Channel{
					Name:     name + strconv.Itoa(i),
					DataType: telem.Float64T,
					Virtual:  true,
				}
				if err := channels.NewWriter(tx).Create(ctx, &ch); err != nil {
					return err
				}
				keys = append(keys, ch.Key())
			}
		}
		return nil
	}); err != nil {
		b.Fatal(err)
	}
	return keys
}

// benchBuild creates an automation from base and measures each build of it.
func benchBuild(b *testing.B, svc *arc.Service, db *gorp.DB, base arc.Base) {
	b.Helper()
	ctx := b.Context()
	a := newArc(arc.TypeAutomation, base)
	if err := db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.NewWriter(tx).Create(ctx, &a)
	}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := svc.Build(ctx, a.Base().Key); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBuild builds a text automation of n statements, each of which reads one
// channel and writes another, and a graph automation of n sources, each wired to a
// sink, on real Gorp.
func BenchmarkBuild(b *testing.B) {
	for _, n := range []int{10, 100} {
		b.Run(fmt.Sprintf("text=%d", n), func(b *testing.B) {
			svc, db, _ := benchService(b, n)
			var src strings.Builder
			for i := range n {
				fmt.Fprintf(&src, "bench_in_%d * 2.0 -> bench_out_%d\n", i, i)
			}
			benchBuild(b, svc, db, arc.Base{
				Name: "bench",
				Source: arc.Source{
					Variant: arc.TextSource{Text: text.Text{Raw: src.String()}},
				},
			})
		})
		b.Run(fmt.Sprintf("graph=%d", n), func(b *testing.B) {
			svc, db, keys := benchService(b, n)
			g := graph.Graph{Inputs: map[string]msgpack.EncodedJSON{}}
			for i := range n {
				source, sink := fmt.Sprintf("on_%d", i), fmt.Sprintf("write_%d", i)
				g.Nodes = append(
					g.Nodes,
					graph.Node{Key: source},
					graph.Node{Key: sink},
				)
				g.Inputs[source] = msgpack.EncodedJSON{
					"type":    "on",
					"channel": keys[2*i],
				}
				g.Inputs[sink] = msgpack.EncodedJSON{
					"type":    "write",
					"channel": keys[2*i+1],
				}
				g.Edges = append(g.Edges, graph.Edge{
					Key:    fmt.Sprintf("edge_%d", i),
					Source: ir.Handle{Node: source, Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: sink, Param: ir.DefaultInputParam},
				})
			}
			benchBuild(b, svc, db, arc.Base{
				Name:   "bench",
				Source: arc.Source{Variant: arc.GraphSource{Graph: g}},
			})
		})
	}
}
