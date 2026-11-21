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
