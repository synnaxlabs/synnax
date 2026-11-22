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

	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkResolve measures one lookup of the Arc symbol resolver beside a table of
// 10,001 channels: by key, by name, by path at depths 1 and 3, and of a path that no
// channel holds.
func BenchmarkResolve(b *testing.B) {
	env := newBenchEnv(b, 5_000)
	b.Cleanup(func() { env.close(b) })
	key := createResolveFolders(b, env)
	r := env.svc.NewArcSymbolResolver(env.db)
	for _, c := range []struct {
		name    string
		resolve func(context.Context) (*symbol.Symbol, error)
	}{
		{"key", func(ctx context.Context) (*symbol.Symbol, error) {
			return r.ResolveKey(ctx, uint32(key))
		}},
		{"name", func(ctx context.Context) (*symbol.Symbol, error) {
			return r.Resolve(ctx, "at_3")
		}},
		{"path/depth=1", func(ctx context.Context) (*symbol.Symbol, error) {
			return r.Resolve(ctx, "folder_0.at_1")
		}},
		{"path/depth=3", func(ctx context.Context) (*symbol.Symbol, error) {
			return r.Resolve(ctx, "folder_0.folder_1.folder_2.at_3")
		}},
	} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := c.resolve(b.Context()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
	b.Run("missing", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := r.Resolve(b.Context(), "folder_0.nope"); err == nil {
				b.Fatal("resolved a path that no channel holds")
			}
		}
	})
}

// createResolveFolders makes folders folder_0 to folder_2, each inside the one before,
// and a channel at_<depth> in each. It returns the key of the deepest channel.
func createResolveFolders(b *testing.B, env *benchEnv) channel.Key {
	b.Helper()
	var key channel.Key
	if err := env.db.WithTx(b.Context(), func(tx gorp.Tx) error {
		var parent *group.Key
		for i := range 3 {
			g := group.Group{
				Name:  fmt.Sprintf("folder_%d", i),
				Scope: resource.TypeChannel,
			}
			if parent != nil {
				g.Parent = new(group.ResourceID(*parent))
			}
			if err := env.groups.NewWriter(tx).Create(b.Context(), &g); err != nil {
				return err
			}
			parent = &g.Key
			ch := channel.Channel{
				Name:     fmt.Sprintf("at_%d", i+1),
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    parent,
			}
			if err := env.svc.NewWriter(tx).Create(b.Context(), &ch); err != nil {
				return err
			}
			key = ch.Key()
		}
		return nil
	}); err != nil {
		b.Fatal(err)
	}
	return key
}
