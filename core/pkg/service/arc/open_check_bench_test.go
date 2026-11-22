// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"fmt"
	"strconv"
	"testing"
	"uuid"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/io"
	. "github.com/synnaxlabs/x/testutil"
)

// BenchmarkOpenTableFilled opens the Arc table over n stored text automations whose
// lists name the channels they read and write, as each Arc holds after a Core start.
func BenchmarkOpenTableFilled(b *testing.B) {
	for _, n := range []int{100, 1000} {
		b.Run(fmt.Sprintf("arcs=%d", n), func(b *testing.B) {
			db, first := storeFilledArcs(b, n)
			ctx := b.Context()
			rewrite := func(tx gorp.Tx) error {
				return gorp.WrapWriter[arc.Key, arc.Arc](tx).Set(ctx, first)
			}
			b.ReportAllocs()
			for b.Loop() {
				t := MustSucceed(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
				if err := db.WithTx(ctx, rewrite); err != nil {
					b.Fatal(err)
				}
				if err := t.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func storeFilledArcs(b *testing.B, n int) (*gorp.DB, arc.Arc) {
	b.Helper()
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	node := testutil.OpenNode(ctx)
	statuses := MustSucceed(status.OpenService(ctx, status.ServiceConfig{DB: node.DB}))
	channels := MustSucceed(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.Storage.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statuses,
	}))
	b.Cleanup(func() {
		if err := (io.MultiCloser{node, statuses, channels}).Close(); err != nil {
			b.Error(err)
		}
	})
	keys := createBenchChannels(b, node.DB, channels, 1)
	table := MustSucceed(arc.OpenTable(ctx, arc.TableConfig{DB: node.DB}))
	var root directory.Directory
	if err := gorp.NewRetrieve[directory.Key, directory.Directory]().
		Entry(&root).
		Exec(ctx, node.DB); err != nil {
		b.Fatal(err)
	}
	arcs := make([]arc.Arc, n)
	for i := range arcs {
		arcs[i] = newArc(arc.TypeAutomation, arc.Base{
			Key:       uuid.New(),
			Name:      "filled_" + strconv.Itoa(i),
			Directory: &root.Key,
			Source: arc.Source{Variant: arc.TextSource{
				Text: text.Text{Raw: "bench_in_0 * 2.0 -> bench_out_0"},
			}},
			Reads:  []channel.Key{keys[0]},
			Writes: []channel.Key{keys[1]},
		})
	}
	if err := node.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return gorp.WrapWriter[arc.Key, arc.Arc](tx).Set(ctx, arcs...)
	}); err != nil {
		b.Fatal(err)
	}
	if err := table.Close(); err != nil {
		b.Fatal(err)
	}
	return node.DB, arcs[0]
}
