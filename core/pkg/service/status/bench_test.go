// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"io"
	"testing"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// closeOnCleanup fails b if err is not nil and closes v when b finishes.
func closeOnCleanup(b *testing.B, v io.Closer, err error) {
	b.Helper()
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := v.Close(); err != nil {
			b.Error(err)
		}
	})
}

func BenchmarkWriter(b *testing.B) {
	ctx := b.Context()
	db := gorp.Wrap(memkv.New())
	b.Cleanup(func() {
		if err := db.Close(); err != nil {
			b.Error(err)
		}
	})
	otg, err := ontology.Open(ctx, ontology.Config{DB: db})
	closeOnCleanup(b, otg, err)
	g, err := group.OpenService(ctx, group.ServiceConfig{DB: db, Ontology: otg})
	closeOnCleanup(b, g, err)
	lbl, err := label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	})
	closeOnCleanup(b, lbl, err)
	svc, err := status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
		Label:    lbl,
	})
	closeOnCleanup(b, svc, err)
	b.Run("Set", func(b *testing.B) {
		variants := [2]status.Variant{status.VariantError, status.VariantSuccess}
		i := 0
		for b.Loop() {
			s := status.Status[any]{
				Key:     "bench",
				Variant: variants[i%2],
				Message: "Message",
				Time:    telem.Now(),
			}
			if err := svc.NewWriter(nil).Set(ctx, &s); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}
