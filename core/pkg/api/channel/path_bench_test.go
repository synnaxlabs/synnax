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
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkPathFill measures a retrieve of a page of 100 channels in 10 folders two
// levels down, with and without include_path.
func BenchmarkPathFill(b *testing.B) {
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	node := disttestutil.OpenNode(ctx)
	layer, err := svctestutil.OpenLayer(ctx, node)
	if err != nil {
		b.Fatalf("failed to open the layer: %v", err)
	}
	defer func() {
		if err := layer.Close(); err != nil {
			b.Errorf("failed to close the layer: %v", err)
		}
		if err := node.Close(); err != nil {
			b.Errorf("failed to close the node: %v", err)
		}
	}()
	retrieve := benchRetrieve(b, node.DB, layer)
	u, err := rbactestutil.CreateUser(
		ctx, node.DB, layer.RBAC, layer.User, user.User{Name: "bench_paths"},
	)
	if err != nil {
		b.Fatalf("failed to create the user: %v", err)
	}
	if _, err = rbactestutil.Grant(
		ctx,
		node.DB,
		layer.RBAC,
		layer.User,
		u.Key,
		[]access.Action{access.ActionRetrieve},
		resource.ID{Type: resource.TypeChannel},
		resource.ID{Type: resource.TypeGroup},
	); err != nil {
		b.Fatalf("failed to grant retrieve: %v", err)
	}
	subject := AuthedCtx(ctx, u)
	channels := make([]channel.Channel, 0, 100)
	if err = node.DB.WithTx(ctx, func(tx gorp.Tx) error {
		top := group.Group{Name: "bench_paths", Scope: resource.TypeChannel}
		if err := layer.Group.NewWriter(tx).Create(ctx, &top); err != nil {
			return err
		}
		for f := range 10 {
			folder := group.Group{
				Name:   fmt.Sprintf("folder_%d", f),
				Scope:  resource.TypeChannel,
				Parent: new(top.ResourceID()),
			}
			if err := layer.Group.NewWriter(tx).Create(ctx, &folder); err != nil {
				return err
			}
			for c := range 10 {
				channels = append(channels, channel.Channel{
					Name:     fmt.Sprintf("bench_path_%d_%d", f, c),
					DataType: telem.Float32T,
					Virtual:  true,
					Group:    &folder.Key,
				})
			}
		}
		return layer.Channel.NewWriter(tx).CreateMany(ctx, &channels)
	}); err != nil {
		b.Fatalf("failed to create the channels: %v", err)
	}
	keys := channel.KeysFromChannels(channels)
	run := func(b *testing.B, path bool) {
		b.Helper()
		b.ReportAllocs()
		for range b.N {
			res, err := retrieve(subject, apichannel.RetrieveRequest{
				Where:       &channel.Where{Key: &filter.Set[channel.Key]{In: keys}},
				IncludePath: path,
			})
			if err != nil {
				b.Fatalf("failed to retrieve: %v", err)
			}
			if len(res.Channels) != len(keys) || path && res.Channels[0].Path == nil {
				b.Fatalf("expected %d channels with paths", len(keys))
			}
		}
	}
	b.Run("page", func(b *testing.B) { run(b, false) })
	b.Run("page_path", func(b *testing.B) { run(b, true) })
}
