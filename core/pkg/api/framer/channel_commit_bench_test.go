// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkChannelCommit measures a channel commit through the service layer, with the
// signals that it feeds: a create of 1 or 100 channels, and a rename.
func BenchmarkChannelCommit(b *testing.B) {
	for _, n := range []int{1, 100} {
		b.Run(fmt.Sprintf("create=%d", n), func(b *testing.B) {
			layer, db := openServiceLayer(b)
			batches := make([][]channel.Channel, b.N)
			for i := range batches {
				batches[i] = make([]channel.Channel, n)
				for j := range batches[i] {
					batches[i][j] = channel.Channel{
						Name:     fmt.Sprintf("commit_%d_%d", i, j),
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
					return layer.Channel.NewWriter(tx).
						CreateMany(b.Context(), &batches[i])
				}); err != nil {
					b.Fatalf("failed to create channels: %v", err)
				}
			}
		})
	}
	b.Run("rename", func(b *testing.B) {
		layer, db := openServiceLayer(b)
		ch := channel.Channel{Name: "commit", DataType: telem.Float32T, Virtual: true}
		if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(b.Context(), &ch)
		}); err != nil {
			b.Fatalf("failed to create the channel: %v", err)
		}
		names := make([]string, b.N)
		for i := range names {
			names[i] = fmt.Sprintf("commit_%d", i)
		}
		runtime.GC()
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).
					Rename(b.Context(), ch.Key(), names[i], false)
			}); err != nil {
				b.Fatalf("failed to rename the channel: %v", err)
			}
		}
	})
}
