// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/errors"
)

// BenchmarkDeleteFolders measures a folder delete as the route ran it before impact
// ("today"), and through Apply as the route runs it now ("impact"). Each run rolls
// back, so the folder stays.
func BenchmarkDeleteFolders(b *testing.B) {
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	node := testutil.OpenNode(ctx)
	layer, err := svctestutil.OpenLayer(ctx, node)
	if err != nil {
		b.Fatalf("failed to open the layer: %v", err)
	}
	defer func() {
		if err := errors.Join(layer.Close(), node.Close()); err != nil {
			b.Errorf("failed to close: %v", err)
		}
	}()
	db := node.DB
	folder := group.Group{Name: "bench", Scope: resource.TypeChannel}
	if err := db.WithTx(ctx, func(tx gorp.Tx) error {
		return layer.Group.NewWriter(tx).Create(ctx, &folder)
	}); err != nil {
		b.Fatalf("failed to create the folder: %v", err)
	}
	keys := []group.Key{folder.Key}
	run := func(b *testing.B, op func(tx gorp.Tx) error) {
		b.Helper()
		b.ReportAllocs()
		for range b.N {
			tx := db.OpenTx()
			if err := errors.Join(op(tx), tx.Close()); err != nil {
				b.Fatalf("failed to delete: %v", err)
			}
		}
	}
	b.Run("today", func(b *testing.B) {
		run(b, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Delete(ctx, keys...)
		})
	})
	b.Run("impact", func(b *testing.B) {
		run(b, func(tx gorp.Tx) error {
			_, err := layer.Impact.Apply(
				ctx,
				tx,
				impact.Change{Variant: impact.DeleteFoldersChange{Keys: keys}},
				impact.ApplyOptions{},
			)
			return err
		})
	})
}
