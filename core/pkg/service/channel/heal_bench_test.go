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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// healBench is a healing channel service with a table of displays that a heal retargets
// and a table of keepers that it does not.
type healBench struct {
	b        *testing.B
	db       *gorp.DB
	svc      *channel.Service
	displays *gorp.Table[string, healHolder]
	keepers  *gorp.Table[string, healKeeper]
}

func openHealBench(b *testing.B) healBench {
	b.Helper()
	ctx := b.Context()
	node := testutil.OpenNode(ctx)
	db := gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec))
	run, stop := context.WithCancel(context.Background())
	ran := make(chan error, 1)
	go func() { ran <- db.Run(run) }()
	b.Cleanup(func() {
		stop()
		if err := errors.Join(<-ran, db.Close(), node.Close()); err != nil {
			b.Error(err)
		}
	})
	statusSvc, err := status.OpenService(ctx, status.ServiceConfig{DB: db})
	if err != nil {
		b.Fatal(err)
	}
	h := healBench{b: b, db: db}
	h.svc, err = channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      db,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
		TS:      node.Storage.TS,
		Retargets: func(d gorp.Dependent) (bool, error) {
			return d.Holder.Table == "display", nil
		},
	})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := errors.Join(h.svc.Close(), statusSvc.Close()); err != nil {
			b.Error(err)
		}
	})
	if h.displays, err = gorp.OpenTable(ctx, gorp.TableConfig[string, healHolder]{
		DB:   db,
		Name: "display",
		References: weakChannelReference(func(h *healHolder) *channel.Key {
			return &h.Channel
		}),
	}); err != nil {
		b.Fatal(err)
	}
	if h.keepers, err = gorp.OpenTable(ctx, gorp.TableConfig[string, healKeeper]{
		DB:   db,
		Name: "keeper",
		References: weakChannelReference(func(h *healKeeper) *channel.Key {
			return &h.Channel
		}),
	}); err != nil {
		b.Fatal(err)
	}
	if err = h.svc.Heal(ctx); err != nil {
		b.Fatal(err)
	}
	return h
}

func (h healBench) write(f func(tx gorp.Tx) error) {
	if err := h.db.WithTx(h.b.Context(), f); err != nil {
		h.b.Fatal(err)
	}
}

// tombstone deletes a new channel at the path of name that holders, the displays that
// it fills, hold, or that one keeper holds when holders is empty.
func (h healBench) tombstone(name string, holders []healHolder) {
	ctx := h.b.Context()
	dead := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
	h.write(func(tx gorp.Tx) error {
		return h.svc.NewWriter(tx).Create(ctx, &dead)
	})
	h.write(func(tx gorp.Tx) error {
		if len(holders) == 0 {
			return h.keepers.NewCreate().
				Entry(&healKeeper{Key: name, Channel: dead.Key()}).
				Exec(ctx, tx)
		}
		for j := range holders {
			holders[j] = healHolder{
				Key:     fmt.Sprintf("%s_%d", name, j),
				Channel: dead.Key(),
			}
		}
		return h.displays.NewCreate().Entries(&holders).Exec(ctx, tx)
	})
	h.write(func(tx gorp.Tx) error {
		return h.svc.NewWriter(tx).Delete(ctx, dead.Key(), false)
	})
}

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
			h := openHealBench(b)
			holders := make([]healHolder, max(n, 0))
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				b.StopTimer()
				name := fmt.Sprintf("heal_%d", i)
				if n != none {
					h.tombstone(name, holders)
				}
				ch := channel.Channel{
					Name:     name,
					DataType: telem.Float32T,
					Virtual:  true,
				}
				b.StartTimer()
				h.write(func(tx gorp.Tx) error {
					return h.svc.NewWriter(tx).Create(b.Context(), &ch)
				})
			}
		})
	}
}
