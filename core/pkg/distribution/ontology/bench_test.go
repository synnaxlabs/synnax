// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology_test

import (
	"context"
	"fmt"
	"io"
	"iter"
	"slices"
	"strconv"
	"testing"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

type benchService struct {
	observe.Noop[iter.Seq[ontology.Change]]
}

var _ ontology.Service = (*benchService)(nil)

const benchType ontology.Type = "bench"

type BenchResource struct {
	Key string
}

func newBenchID(key string) ontology.ID {
	return ontology.ID{Key: key, Type: benchType}
}

var benchSchema = zyn.Object(map[string]zyn.Schema{
	"key": zyn.String(),
})

func (s *benchService) Type() ontology.Type { return benchType }

func (s *benchService) Schema() zyn.Schema { return benchSchema }

func (s *benchService) RetrieveResource(_ context.Context, key string, _ gorp.Tx) (ontology.Resource, error) {
	return core.NewResource(s.Schema(), newBenchID(key), key, BenchResource{Key: key}), nil
}

func (s *benchService) OpenNexter(context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	return slices.Values([]ontology.Resource{}), xio.NopCloser, nil
}

type benchEnv struct {
	ctx context.Context
	db  *gorp.DB
	otg *ontology.Ontology
	svc *benchService
}

func newBenchEnv(b *testing.B, enableSearch bool) *benchEnv {
	ctx := context.Background()
	db := gorp.Wrap(memkv.New())
	svc := &benchService{}
	var searchPtr *bool
	if enableSearch {
		searchPtr = config.True()
	} else {
		searchPtr = config.False()
	}
	otg, err := ontology.Open(ctx, ontology.Config{
		DB:           db,
		EnableSearch: searchPtr,
	})
	if err != nil {
		b.Fatalf("failed to open ontology: %v", err)
	}
	otg.RegisterService(svc)
	return &benchEnv{ctx: ctx, db: db, otg: otg, svc: svc}
}

func (e *benchEnv) close(b *testing.B) {
	if err := e.otg.Close(); err != nil {
		b.Errorf("failed to close ontology: %v", err)
	}
	if err := e.db.Close(); err != nil {
		b.Errorf("failed to close db: %v", err)
	}
}

func (e *benchEnv) populate(b *testing.B, count int) []ontology.ID {
	tx := e.db.OpenTx()
	defer func() { _ = tx.Close() }()
	w := e.otg.NewWriter(tx)
	ids := make([]ontology.ID, count)
	for i := range count {
		ids[i] = newBenchID(strconv.Itoa(i))
		if err := w.DefineResource(e.ctx, ids[i]); err != nil {
			b.Fatalf("failed to define resource: %v", err)
		}
	}
	if err := tx.Commit(e.ctx); err != nil {
		b.Fatalf("failed to commit: %v", err)
	}
	return ids
}

func (e *benchEnv) populateTree(b *testing.B, depth, width int) (root ontology.ID, leaves []ontology.ID) {
	tx := e.db.OpenTx()
	defer func() { _ = tx.Close() }()
	w := e.otg.NewWriter(tx)
	counter := 0
	var build func(d int, parent ontology.ID) []ontology.ID
	build = func(d int, parent ontology.ID) []ontology.ID {
		if d == 0 {
			return []ontology.ID{parent}
		}
		var result []ontology.ID
		for range width {
			child := newBenchID(strconv.Itoa(counter))
			counter++
			if err := w.DefineResource(e.ctx, child); err != nil {
				b.Fatalf("failed to define resource: %v", err)
			}
			if err := w.DefineRelationship(e.ctx, parent, ontology.ParentOf, child); err != nil {
				b.Fatalf("failed to define relationship: %v", err)
			}
			result = append(result, build(d-1, child)...)
		}
		return result
	}
	root = newBenchID("root")
	if err := w.DefineResource(e.ctx, root); err != nil {
		b.Fatalf("failed to define root: %v", err)
	}
	leaves = build(depth, root)
	if err := tx.Commit(e.ctx); err != nil {
		b.Fatalf("failed to commit: %v", err)
	}
	return root, leaves
}

func BenchmarkRetrieveByID(b *testing.B) {
	for _, count := range []int{100, 1000, 10000} {
		for _, batch := range []int{1, 10, 100} {
			b.Run(fmt.Sprintf("resources=%d/batch=%d", count, batch), func(b *testing.B) {
				env := newBenchEnv(b, false)
				defer env.close(b)
				ids := env.populate(b, count)
				queryIDs := ids[:batch]
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var res []ontology.Resource
					_ = env.otg.NewRetrieve().WhereIDs(queryIDs...).Entries(&res).Exec(env.ctx, nil)
				}
			})
		}
	}
}

func BenchmarkTraverseChildren(b *testing.B) {
	for _, depth := range []int{2, 4} {
		for _, width := range []int{5, 10} {
			b.Run(fmt.Sprintf("depth=%d/width=%d", depth, width), func(b *testing.B) {
				env := newBenchEnv(b, false)
				defer env.close(b)
				root, _ := env.populateTree(b, depth, width)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var res []ontology.Resource
					_ = env.otg.NewRetrieve().WhereIDs(root).TraverseTo(ontology.Children).Entries(&res).Exec(env.ctx, nil)
				}
			})
		}
	}
}

func BenchmarkTraverseParents(b *testing.B) {
	for _, depth := range []int{2, 5, 10} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			env := newBenchEnv(b, false)
			defer env.close(b)
			_, leaves := env.populateTree(b, depth, 1)
			leaf := leaves[0]
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var res []ontology.Resource
				_ = env.otg.NewRetrieve().WhereIDs(leaf).TraverseTo(ontology.Parents).Entries(&res).Exec(env.ctx, nil)
			}
		})
	}
}

func BenchmarkPagination(b *testing.B) {
	for _, total := range []int{1000, 10000} {
		for _, offset := range []int{0, total / 2} {
			b.Run(fmt.Sprintf("total=%d/offset=%d", total, offset), func(b *testing.B) {
				env := newBenchEnv(b, false)
				defer env.close(b)
				env.populate(b, total)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var res []ontology.Resource
					_ = env.otg.NewRetrieve().Offset(offset).Limit(50).Entries(&res).Exec(env.ctx, nil)
				}
			})
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	for _, count := range []int{1000, 10000} {
		b.Run(fmt.Sprintf("resources=%d", count), func(b *testing.B) {
			env := newBenchEnv(b, true)
			defer env.close(b)
			env.populate(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = env.otg.Search(env.ctx, search.Request{Term: "500"})
			}
		})
	}
}

func BenchmarkRetrieveByType(b *testing.B) {
	for _, count := range []int{1000, 10000} {
		b.Run(fmt.Sprintf("resources=%d", count), func(b *testing.B) {
			env := newBenchEnv(b, false)
			defer env.close(b)
			env.populate(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var res []ontology.Resource
				_ = env.otg.NewRetrieve().WhereTypes(benchType).Entries(&res).Exec(env.ctx, nil)
			}
		})
	}
}

func BenchmarkMultiHopTraversal(b *testing.B) {
	for _, depth := range []int{3, 5} {
		for _, hops := range []int{2, 3} {
			if hops > depth {
				continue
			}
			b.Run(fmt.Sprintf("depth=%d/hops=%d", depth, hops), func(b *testing.B) {
				env := newBenchEnv(b, false)
				defer env.close(b)
				root, _ := env.populateTree(b, depth, 3)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					q := env.otg.NewRetrieve().WhereIDs(root)
					for range hops {
						q = q.TraverseTo(ontology.Children)
					}
					var res []ontology.Resource
					_ = q.Entries(&res).Exec(env.ctx, nil)
				}
			})
		}
	}
}

func BenchmarkIntermediateTraversalOverhead(b *testing.B) {
	for _, width := range []int{10, 50} {
		b.Run(fmt.Sprintf("width=%d/depth=3", width), func(b *testing.B) {
			env := newBenchEnv(b, false)
			defer env.close(b)
			root, _ := env.populateTree(b, 3, width)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var res []ontology.Resource
				_ = env.otg.NewRetrieve().
					WhereIDs(root).
					TraverseTo(ontology.Children).
					TraverseTo(ontology.Children).
					TraverseTo(ontology.Children).
					Entries(&res).
					Exec(env.ctx, nil)
			}
		})
	}
}

func (e *benchEnv) populateParentsWithChildren(b *testing.B, numParents, childrenPerParent int) []ontology.ID {
	tx := e.db.OpenTx()
	defer func() { _ = tx.Close() }()
	w := e.otg.NewWriter(tx)
	parents := make([]ontology.ID, numParents)
	counter := 0
	for i := range numParents {
		parents[i] = newBenchID(fmt.Sprintf("parent-%d", i))
		if err := w.DefineResource(e.ctx, parents[i]); err != nil {
			b.Fatalf("failed to define parent: %v", err)
		}
		for range childrenPerParent {
			child := newBenchID(fmt.Sprintf("child-%d", counter))
			counter++
			if err := w.DefineResource(e.ctx, child); err != nil {
				b.Fatalf("failed to define child: %v", err)
			}
			if err := w.DefineRelationship(e.ctx, parents[i], ontology.ParentOf, child); err != nil {
				b.Fatalf("failed to define relationship: %v", err)
			}
		}
	}
	if err := tx.Commit(e.ctx); err != nil {
		b.Fatalf("failed to commit: %v", err)
	}
	return parents
}

func BenchmarkTraverseChildrenByType(b *testing.B) {
	for _, numParents := range []int{5, 20} {
		for _, childrenPerParent := range []int{10, 50} {
			b.Run(fmt.Sprintf("parents=%d/children=%d/nofilter", numParents, childrenPerParent), func(b *testing.B) {
				env := newBenchEnv(b, false)
				defer env.close(b)
				parents := env.populateParentsWithChildren(b, numParents, childrenPerParent)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var res []ontology.Resource
					_ = env.otg.NewRetrieve().
						WhereIDs(parents...).
						TraverseTo(ontology.Children).
						Entries(&res).
						Exec(env.ctx, nil)
				}
			})
			b.Run(fmt.Sprintf("parents=%d/children=%d/withfilter", numParents, childrenPerParent), func(b *testing.B) {
				env := newBenchEnv(b, false)
				defer env.close(b)
				parents := env.populateParentsWithChildren(b, numParents, childrenPerParent)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var res []ontology.Resource
					_ = env.otg.NewRetrieve().
						WhereIDs(parents...).
						TraverseTo(ontology.Children).
						WhereTypes(benchType).
						Entries(&res).
						Exec(env.ctx, nil)
				}
			})
		}
	}
}

var _ = lo.Must(1, nil)
