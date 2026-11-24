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
	"context"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/graph"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
)

type benchEnv struct {
	db       *gorp.DB
	statuses *status.Service
	channels *channel.Service
	creator  calctestutil.Creator
}

func openBenchEnv(b *testing.B) benchEnv {
	b.Helper()
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	node := testutil.OpenNode(ctx)
	labels := lo.Must(label.OpenService(ctx, label.ServiceConfig{DB: node.DB}))
	statuses := lo.Must(status.OpenService(ctx, status.ServiceConfig{DB: node.DB}))
	channels := lo.Must(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.Storage.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statuses,
	}))
	calcs := lo.Must(calculation.OpenService(ctx, calculation.ServiceConfig{
		DB:      node.DB,
		Channel: channels,
		Framer:  node.Framer,
	}))
	b.Cleanup(func() {
		lo.Must0(calcs.Close())
		lo.Must0(channels.Close())
		lo.Must0(statuses.Close())
		lo.Must0(labels.Close())
		lo.Must0(node.Close())
	})
	return benchEnv{
		db:       node.DB,
		statuses: statuses,
		channels: channels,
		creator: calctestutil.Creator{
			DB:          node.DB,
			Channel:     channels,
			Calculation: calcs,
		},
	}
}

func (e benchEnv) openGraph(b *testing.B) *graph.Graph {
	b.Helper()
	return lo.Must(graph.Open(b.Context(), graph.Config{
		DB:          e.db,
		Channel:     e.channels,
		Calculation: e.creator.Calculation,
		Status:      e.statuses,
	}))
}

// createRaw creates n virtual channels, named prefix and a number, in one transaction.
func (e benchEnv) createRaw(ctx context.Context, prefix string, n int) {
	chs := make([]channel.Channel, n)
	for i := range chs {
		chs[i] = channel.Channel{
			Name:     prefix + strconv.Itoa(i),
			DataType: telem.Float32T,
			Virtual:  true,
		}
	}
	lo.Must0(e.db.WithTx(ctx, func(tx gorp.Tx) error {
		return e.channels.NewWriter(tx).CreateMany(ctx, &chs)
	}))
}

// createCalculations creates a virtual channel and n calculations that read it.
func (e benchEnv) createCalculations(ctx context.Context, prefix string, n int) {
	e.createRaw(ctx, prefix+"_base_", 1)
	calcs := make([]calctestutil.Calculation, n)
	for i := range calcs {
		calcs[i] = calctestutil.Calculation{
			Name:       prefix + "_calc_" + strconv.Itoa(i),
			DataType:   telem.Float32T,
			Expression: "return " + prefix + "_base_0 * 2",
		}
	}
	lo.Must(e.creator.CreateMany(ctx, calcs))
}

// BenchmarkGraphChannelChange creates n virtual channels in one transaction per
// operation while the graph holds 10 calculations, and waits until the graph has
// handled each one.
func BenchmarkGraphChannelChange(b *testing.B) {
	for _, n := range []int{1, 100} {
		b.Run("channels="+strconv.Itoa(n), func(b *testing.B) {
			ctx := b.Context()
			e := openBenchEnv(b)
			e.createCalculations(ctx, "change", 10)
			g := e.openGraph(b)
			defer func() { lo.Must0(g.Close()) }()
			var handled atomic.Int64
			disconnect := g.Observe().
				OnChange(func(_ context.Context, r graph.Changes) {
					for c := range r {
						if strings.HasPrefix(c.Value.Name, "raw") {
							handled.Add(1)
						}
					}
				})
			defer disconnect()
			b.ReportAllocs()
			op := 0
			for b.Loop() {
				e.createRaw(ctx, "raw"+strconv.Itoa(op)+"_", n)
				op++
				for handled.Load() < int64(op*n) {
					time.Sleep(20 * time.Microsecond)
				}
			}
		})
	}
}

// BenchmarkGraphOpen opens and closes the graph over n calculations and 1,000 virtual
// channels. Each open runs the start pass over every calculation.
func BenchmarkGraphOpen(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		b.Run("calculations="+strconv.Itoa(n), func(b *testing.B) {
			ctx := b.Context()
			e := openBenchEnv(b)
			e.createCalculations(ctx, "open", n)
			e.createRaw(ctx, "open_raw_", 1000)
			b.ReportAllocs()
			for b.Loop() {
				lo.Must0(e.openGraph(b).Close())
			}
		})
	}
}
