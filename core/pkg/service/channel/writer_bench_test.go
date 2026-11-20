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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
)

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
