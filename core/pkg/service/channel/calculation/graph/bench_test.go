// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	graph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/telem"
)

// benchGraphEnv holds a Graph hydrated from a fixed number of calculated channels.
type benchGraphEnv struct {
	graph   *graph.Graph
	closer  io.MultiCloser
	indexes []channel.Key
}

// newBenchGraphEnv creates count calculated channels off one stored channel, then opens
// a Graph over them. The returned indexes are the auto-created index of each.
func newBenchGraphEnv(b *testing.B, count int) *benchGraphEnv {
	gomega.RegisterTestingT(b)
	node := mock.OpenNode(b.Context())
	otg, err := ontology.Open(b.Context(), ontology.Config{DB: node.DB})
	if err != nil {
		b.Fatalf("failed to open ontology: %v", err)
	}
	searchIdx, err := search.OpenIndex()
	if err != nil {
		b.Fatalf("failed to open search index: %v", err)
	}
	groupSvc, err := group.OpenService(b.Context(), group.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Search:   searchIdx,
	})
	if err != nil {
		b.Fatalf("failed to open group service: %v", err)
	}
	labelSvc, err := label.OpenService(b.Context(), label.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	})
	if err != nil {
		b.Fatalf("failed to open label service: %v", err)
	}
	statusSvc, err := status.OpenService(b.Context(), status.ServiceConfig{
		DB:       node.DB,
		Group:    groupSvc,
		Ontology: otg,
		Label:    labelSvc,
		Search:   searchIdx,
	})
	if err != nil {
		b.Fatalf("failed to open status service: %v", err)
	}
	chSvc, err := channel.OpenService(b.Context(), channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       searchIdx,
		Status:       statusSvc,
	})
	if err != nil {
		b.Fatalf("failed to open channel service: %v", err)
	}
	w := chSvc.NewWriter(nil)
	idx := channel.Channel{
		Name:     "bench_graph_time",
		DataType: telem.TimestampT,
		IsIndex:  true,
	}
	if err := w.Create(b.Context(), &idx); err != nil {
		b.Fatalf("failed to create index channel: %v", err)
	}
	base := channel.Channel{
		Name:       "bench_graph_base",
		DataType:   telem.Float32T,
		LocalIndex: idx.LocalKey,
	}
	if err := w.Create(b.Context(), &base); err != nil {
		b.Fatalf("failed to create base channel: %v", err)
	}
	calcs := make([]channel.Channel, count)
	for i := range calcs {
		calcs[i] = channel.Channel{
			Name:       fmt.Sprintf("bench_graph_calc_%d", i),
			DataType:   telem.Float32T,
			Expression: "return bench_graph_base * 2",
		}
	}
	if err := w.CreateMany(b.Context(), &calcs); err != nil {
		b.Fatalf("failed to create calculated channels: %v", err)
	}
	g, err := graph.Open(b.Context(), graph.Config{
		DB:      node.DB,
		Channel: chSvc,
		Status:  statusSvc,
	})
	if err != nil {
		b.Fatalf("failed to open graph: %v", err)
	}
	indexes := make([]channel.Key, len(calcs))
	for i, c := range calcs {
		indexes[i] = c.Index()
	}
	return &benchGraphEnv{
		graph:   g,
		indexes: indexes,
		closer: io.MultiCloser{
			g, node, otg, searchIdx, groupSvc, chSvc, statusSvc, labelSvc,
		},
	}
}

func (e *benchGraphEnv) close(b *testing.B) {
	if err := e.closer.Close(); err != nil {
		b.Errorf("failed to close env: %v", err)
	}
}

// BenchmarkOwnerOfIndex measures resolving a free index to the calculated channel that
// writes it, which the iterator does once per requested free index per Open. The count
// axis is cluster-wide calculated channels, since OwnerOfIndex walks every node.
func BenchmarkOwnerOfIndex(b *testing.B) {
	for _, count := range []int{10, 100, 1_000, 10_000} {
		env := newBenchGraphEnv(b, count)
		b.Run(fmt.Sprintf("calculated=%d/hit", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				key := env.indexes[i%len(env.indexes)]
				if _, ok := env.graph.OwnerOfIndex(key); !ok {
					b.Fatalf("failed to resolve index %v", key)
				}
			}
		})
		b.Run(fmt.Sprintf("calculated=%d/miss", count), func(b *testing.B) {
			absent := channel.NewKey(0, 999_999)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, ok := env.graph.OwnerOfIndex(absent); ok {
					b.Fatal("resolved an index that does not exist")
				}
			}
		})
		env.close(b)
	}
}
