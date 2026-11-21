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
	"runtime"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/telem"
)

// openCreateBench opens a channel service over a fresh node, and returns the database
// it writes to.
func openCreateBench(b *testing.B) (*channel.Service, *gorp.DB) {
	b.Helper()
	gomega.RegisterTestingT(b)
	node := testutil.OpenNode(b.Context())
	statuses, err := status.OpenService(b.Context(), status.ServiceConfig{DB: node.DB})
	if err != nil {
		b.Fatalf("failed to open status service: %v", err)
	}
	svc, err := channel.OpenService(b.Context(), channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.Storage.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statuses,
	})
	if err != nil {
		b.Fatalf("failed to open channel service: %v", err)
	}
	b.Cleanup(func() {
		if err := (io.MultiCloser{node, svc, statuses}).Close(); err != nil {
			b.Errorf("failed to close: %v", err)
		}
	})
	return svc, node.DB
}

// BenchmarkCreate measures a create of plain channels, batch at a time in one
// transaction.
func BenchmarkCreate(b *testing.B) {
	for _, batch := range []int{1, 100} {
		b.Run(fmt.Sprintf("batch=%d", batch), func(b *testing.B) {
			svc, db := openCreateBench(b)
			batches := make([][]channel.Channel, b.N)
			for i := range batches {
				batches[i] = make([]channel.Channel, batch)
				for j := range batches[i] {
					batches[i][j] = channel.Channel{
						Name:     fmt.Sprintf("bench_%d_%d", i, j),
						DataType: telem.Float32T,
						Virtual:  true,
					}
				}
			}
			runtime.GC()
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
					return svc.NewWriter(tx).CreateMany(b.Context(), &batches[i])
				}); err != nil {
					b.Fatalf("failed to create channels: %v", err)
				}
			}
		})
	}
}

// BenchmarkWriterCreate measures a create of a batch of virtual channels in a table
// that holds 1,000 channels.
func BenchmarkWriterCreate(b *testing.B) {
	for _, batch := range []int{1, 100} {
		b.Run(fmt.Sprintf("batch=%d", batch), func(b *testing.B) {
			env := newBenchEnv(b, 1_000)
			b.Cleanup(func() { env.close(b) })
			db := env.db
			chs := make([]channel.Channel, batch)
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				for j := range chs {
					chs[j] = channel.Channel{
						Name:     fmt.Sprintf("create_%d_%d", i, j),
						DataType: telem.Float32T,
						Virtual:  true,
					}
				}
				if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
					return env.svc.NewWriter(tx).CreateMany(b.Context(), &chs)
				}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkWriterRename measures a rename of one channel in a table that holds 1,000
// channels.
func BenchmarkWriterRename(b *testing.B) {
	env := newBenchEnv(b, 1_000)
	b.Cleanup(func() { env.close(b) })
	db := env.db
	ch := channel.Channel{Name: "rename_0", DataType: telem.Float32T, Virtual: true}
	if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
		return env.svc.NewWriter(tx).Create(b.Context(), &ch)
	}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		name := fmt.Sprintf("rename_%d", i+1)
		if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
			return env.svc.NewWriter(tx).Rename(b.Context(), ch.Key(), name, false)
		}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWriterMove measures a move of one channel between a group and the top, in a
// table that holds 1,000 channels.
func BenchmarkWriterMove(b *testing.B) {
	env := newBenchEnv(b, 1_000)
	b.Cleanup(func() { env.close(b) })
	g, err := env.groups.CreateOrRetrieve(b.Context(), "Moves", resource.TypeChannel)
	if err != nil {
		b.Fatal(err)
	}
	ch := channel.Channel{Name: "move_0", DataType: telem.Float32T, Virtual: true}
	if err = env.db.WithTx(b.Context(), func(tx gorp.Tx) error {
		return env.svc.NewWriter(tx).Create(b.Context(), &ch)
	}); err != nil {
		b.Fatal(err)
	}
	targets := []*group.Key{&g.Key, nil}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		if err = env.db.WithTx(b.Context(), func(tx gorp.Tx) error {
			return env.svc.NewWriter(tx).SetGroup(b.Context(), targets[i%2], ch.Key())
		}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOpenService measures an open of the channel service over a table of 2,001
// channels, through the populate of its indexes, and its close.
func BenchmarkOpenService(b *testing.B) {
	ctx := b.Context()
	node := testutil.OpenNode(ctx)
	groupSvc, err := group.OpenService(ctx, group.ServiceConfig{DB: node.DB})
	if err != nil {
		b.Fatal(err)
	}
	statusSvc, err := status.OpenService(ctx, status.ServiceConfig{DB: node.DB})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := (io.MultiCloser{node, groupSvc, statusSvc}).Close(); err != nil {
			b.Error(err)
		}
	})
	cfg := channel.ServiceConfig{
		Channel: node.Channel,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}
	open := func() *channel.Service {
		svc, err := channel.OpenService(ctx, cfg)
		if err != nil {
			b.Fatal(err)
		}
		return svc
	}
	svc := open()
	createBenchChannels(b, node.DB, svc, 1_000)
	if err = svc.Close(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		svc = open()
		// A create waits for the populate of the indexes that it checks.
		ch := channel.Channel{
			Name:     fmt.Sprintf("open_%d", i),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		if err = node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		}); err != nil {
			b.Fatal(err)
		}
		if err = svc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
