// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/telem"
)

// benchEnv holds a channel service whose table has a fixed number of calculated
// channels.
type benchEnv struct {
	// svc is the channel service under test.
	svc *channel.Service
	// closer tears down the service and everything it was opened over.
	closer io.MultiCloser
	// indexes are the auto-created index keys of the calculated channels.
	indexes []channel.Key
}

// newBenchEnv creates count calculated channels off one stored channel.
func newBenchEnv(b *testing.B, count int) *benchEnv {
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
	svc, err := channel.OpenService(b.Context(), channel.ServiceConfig{
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
	w := svc.NewWriter(nil)
	idx := channel.Channel{
		Name:     "bench_time",
		DataType: telem.TimestampT,
		IsIndex:  true,
	}
	if err := w.Create(b.Context(), &idx); err != nil {
		b.Fatalf("failed to create index channel: %v", err)
	}
	base := channel.Channel{
		Name:       "bench_base",
		DataType:   telem.Float32T,
		LocalIndex: idx.LocalKey,
	}
	if err := w.Create(b.Context(), &base); err != nil {
		b.Fatalf("failed to create base channel: %v", err)
	}
	calcs := make([]channel.Channel, count)
	for i := range calcs {
		calcs[i] = channel.Channel{
			Name:       fmt.Sprintf("bench_calc_%d", i),
			DataType:   telem.Float32T,
			Expression: "return bench_base * 2",
		}
	}
	if err := w.CreateMany(b.Context(), &calcs); err != nil {
		b.Fatalf("failed to create calculated channels: %v", err)
	}
	indexes := make([]channel.Key, len(calcs))
	for i, c := range calcs {
		indexes[i] = c.Index()
	}
	return &benchEnv{
		svc:     svc,
		indexes: indexes,
		closer: io.MultiCloser{
			node, otg, searchIdx, groupSvc, svc, statusSvc, labelSvc,
		},
	}
}

func (e *benchEnv) close(b *testing.B) {
	if err := e.closer.Close(); err != nil {
		b.Errorf("failed to close env: %v", err)
	}
}

// owners resolves the calculated channels indexed by key.
func (e *benchEnv) owners(b *testing.B, key channel.Key) []channel.Channel {
	var owners []channel.Channel
	if err := e.svc.NewRetrieve().
		Where(channel.MatchCalculated()).
		Where(channel.MatchIndexes(key)).
		Entries(&owners).
		Exec(b.Context(), nil); err != nil {
		b.Fatalf("failed to resolve owner: %v", err)
	}
	return owners
}

// BenchmarkMatchIndexes measures resolving a free index to its calculated owner as the
// table grows. The scan case retrieves every calculated channel, the cost a full-table
// fallback pays.
func BenchmarkMatchIndexes(b *testing.B) {
	for _, count := range []int{10, 100, 1_000, 10_000} {
		env := newBenchEnv(b, count)
		b.Run(fmt.Sprintf("calculated=%d/hit", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := range b.N {
				if n := len(env.owners(b, env.indexes[i%len(env.indexes)])); n != 1 {
					b.Fatalf("expected one owner, got %d", n)
				}
			}
		})
		b.Run(fmt.Sprintf("calculated=%d/miss", count), func(b *testing.B) {
			absent := channel.NewKey(0, 999_999)
			b.ReportAllocs()
			for range b.N {
				if n := len(env.owners(b, absent)); n != 0 {
					b.Fatalf("resolved an index that does not exist: %d", n)
				}
			}
		})
		b.Run(fmt.Sprintf("calculated=%d/scan", count), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				var all []channel.Channel
				if err := env.svc.NewRetrieve().
					Where(channel.MatchCalculated()).
					Entries(&all).
					Exec(b.Context(), nil); err != nil {
					b.Fatalf("failed to scan: %v", err)
				}
				if len(all) != count {
					b.Fatalf("expected %d calculated channels, got %d", count, len(all))
				}
			}
		})
		env.close(b)
	}
}
