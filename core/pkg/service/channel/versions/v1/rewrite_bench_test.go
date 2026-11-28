// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// BenchmarkRewrite rewrites a calculation of lines statements, reads of which read a
// renamed channel, each on its own line.
func BenchmarkRewrite(b *testing.B) {
	ctx := context.Background()
	for _, lines := range []int{1_000, 10_000} {
		for _, reads := range []int{1, 100} {
			b.Run(fmt.Sprintf("lines=%d/reads=%d", lines, reads), func(b *testing.B) {
				w, src := benchRewrite(ctx, b, lines, reads)
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if _, err := w.Rewrite(ctx, src); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// benchRewrite returns a rewriter over the rename of Metrics to Metrics_2, and a
// calculation of lines statements, the first reads of which read Metrics.
func benchRewrite(
	ctx context.Context,
	b *testing.B,
	lines, reads int,
) (v1.Rewriter, v1.Source) {
	b.Helper()
	db := gorp.Wrap(memkv.New())
	b.Cleanup(func() {
		if err := db.Close(); err != nil {
			b.Error(err)
		}
	})
	if err := resource.StageRename(
		ctx,
		db,
		resource.ID{Type: resource.TypeChannel, Key: "1"},
		"Metrics",
		resource.ReasonClash,
	); err != nil {
		b.Fatal(err)
	}
	renames, err := v1.TextRename(db, map[v1.Key]v1.TextChannel{
		1: {Name: "Metrics_2", DataType: telem.Float64T},
	})
	if err != nil {
		b.Fatal(err)
	}
	var text strings.Builder
	for i := range lines {
		if i < reads {
			fmt.Fprintf(&text, "x%d := Metrics * %d.0\n", i, i)
		} else {
			fmt.Fprintf(&text, "x%d := %d.0\n", i, i)
		}
	}
	text.WriteString("return 1.0")
	w := v1.Rewriter{Renames: renames, Ambient: stl.NewSymbols()}
	return w, v1.Source{Text: text.String(), Form: v1.FormCalculation}
}
