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

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
)

// openBenchConfig opens a node and the services that the channel service needs, all
// closed when b ends, and returns the config of a channel service on the node.
func openBenchConfig(b *testing.B) (channel.ServiceConfig, *group.Service) {
	b.Helper()
	ctx := b.Context()
	node := testutil.OpenNode(ctx)
	b.Cleanup(func() {
		if err := node.Close(); err != nil {
			b.Error(err)
		}
	})
	groups, err := group.OpenService(ctx, group.ServiceConfig{DB: node.DB})
	if err != nil {
		b.Fatal(err)
	}
	statuses, err := status.OpenService(ctx, status.ServiceConfig{DB: node.DB})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := statuses.Close(); err != nil {
			b.Error(err)
		}
		if err := groups.Close(); err != nil {
			b.Error(err)
		}
	})
	return channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.Storage.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statuses,
	}, groups
}

// createFolderedChannels creates folders channel folders and n channels through cfg,
// half of the channels in those folders.
func createFolderedChannels(
	b *testing.B,
	cfg channel.ServiceConfig,
	groups *group.Service,
	n, folders int,
) {
	b.Helper()
	ctx := b.Context()
	svc, err := channel.OpenService(ctx, cfg)
	if err != nil {
		b.Fatal(err)
	}
	if err = cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		keys := make([]group.Key, folders)
		for i := range keys {
			g := group.Group{
				Name:  fmt.Sprintf("folder_%d", i),
				Scope: resource.TypeChannel,
			}
			if err := groups.NewWriter(tx).Create(ctx, &g); err != nil {
				return err
			}
			keys[i] = g.Key
		}
		chs := make([]channel.Channel, n)
		for i := range chs {
			chs[i] = channel.Channel{
				Name:     fmt.Sprintf("open_bench_%d", i),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			if i%2 == 0 {
				chs[i].Group = &keys[i%folders]
			}
		}
		return svc.NewWriter(tx).CreateMany(ctx, &chs)
	}); err != nil {
		b.Fatal(err)
	}
	if err = svc.Close(); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkOpenService opens the channel service over 100,000 stored channels, half
// of them in 100 folders, and waits for its indexes to fill, as a Core start does.
func BenchmarkOpenService(b *testing.B) {
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	cfg, groups := openBenchConfig(b)
	createFolderedChannels(b, cfg, groups, 100_000, 100)
	b.ReportAllocs()
	for b.Loop() {
		opened, err := channel.OpenService(ctx, cfg)
		if err != nil {
			b.Fatal(err)
		}
		// A query on the group reference waits for the indexes to fill.
		if _, err = opened.NewRetrieve().
			Where(channel.MatchGrouped(false)).
			Exists(ctx, cfg.DB); err != nil {
			b.Fatal(err)
		}
		if err = opened.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
