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
	"context"
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// BenchmarkHeal measures a create that takes the path of a tombstone whose dead key n
// displays hold, which retargets each of them. With n of 0, a holder that the heal
// keeps holds the dead key, so the create still heals. tombstone=none is a create with
// no tombstone at its path.
func BenchmarkHeal(b *testing.B) {
	// none is a create at a path with no tombstone, the cost a heal adds to.
	const none = -1
	for _, n := range []int{none, 0, 1, 10, 100} {
		label := fmt.Sprintf("holders=%d", n)
		if n == none {
			label = "tombstone=none"
		}
		b.Run(label, func(b *testing.B) {
			gomega.RegisterTestingT(b)
			ctx := b.Context()
			node := testutil.OpenNode(ctx)
			db := gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec))
			run, stop := context.WithCancel(context.Background())
			ran := make(chan error, 1)
			go func() { ran <- db.Run(run) }()
			b.Cleanup(func() {
				stop()
				<-ran
				_ = db.Close()
				_ = node.Close()
			})
			statusSvc, err := status.OpenService(ctx, status.ServiceConfig{DB: db})
			if err != nil {
				b.Fatal(err)
			}
			svc, err := channel.OpenService(ctx, channel.ServiceConfig{
				Channel: node.Channel,
				DB:      db,
				HostKey: node.Identity.HostKey,
				Status:  statusSvc,
				Retargets: func(d gorp.Dependent) bool {
					return d.Holder.Table == "display"
				},
			})
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() {
				_ = svc.Close()
				_ = statusSvc.Close()
			})
			displays, err := gorp.OpenTable(ctx, gorp.TableConfig[string, healHolder]{
				DB:   db,
				Name: "display",
				References: weakChannelReference(func(h *healHolder) *channel.Key {
					return &h.Channel
				}),
			})
			if err != nil {
				b.Fatal(err)
			}
			keepers, err := gorp.OpenTable(ctx, gorp.TableConfig[string, healKeeper]{
				DB:   db,
				Name: "keeper",
				References: weakChannelReference(func(h *healKeeper) *channel.Key {
					return &h.Channel
				}),
			})
			if err != nil {
				b.Fatal(err)
			}
			write := func(f func(tx gorp.Tx) error) {
				if err := db.WithTx(ctx, f); err != nil {
					b.Fatal(err)
				}
			}
			held := make([]healHolder, max(n, 0))
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				b.StopTimer()
				name := fmt.Sprintf("heal_%d", i)
				ch := channel.Channel{
					Name:     name,
					DataType: telem.Float32T,
					Virtual:  true,
				}
				if n == none {
					b.StartTimer()
					write(func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &ch)
					})
					continue
				}
				dead := channel.Channel{
					Name:     name,
					DataType: telem.Float32T,
					Virtual:  true,
				}
				write(func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &dead)
				})
				write(func(tx gorp.Tx) error {
					if n == 0 {
						return keepers.NewCreate().
							Entry(&healKeeper{Key: name, Channel: dead.Key()}).
							Exec(ctx, tx)
					}
					for j := range held {
						held[j] = healHolder{
							Key:     fmt.Sprintf("%s_%d", name, j),
							Channel: dead.Key(),
						}
					}
					return displays.NewCreate().Entries(&held).Exec(ctx, tx)
				})
				write(func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, dead.Key(), false)
				})
				b.StartTimer()
				write(func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})
			}
		})
	}
}
