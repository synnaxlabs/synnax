// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/metrics"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// A store from before the sibling rule can hold a name in several folders. A start
// never fails on a name.
var _ = Describe("Ambiguity breaker", func() {
	var (
		cfg      metrics.ServiceConfig
		groups   *group.Service
		channels *channel.Service
		db       *gorp.DB
	)
	BeforeEach(func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		db = n.DB
		groups = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: n.DB}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: n.DB}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: n.DB}))
		channels = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: n.Channel,
			DB:      n.DB,
			HostKey: n.Identity.HostKey,
			Status:  statuses,
		}))
		graph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
			DB:      n.DB,
			Channel: channels,
			Status:  statuses,
		}))
		framers := MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
			DB:           n.DB,
			Framer:       n.Framer,
			Channel:      channels,
			ChannelGraph: graph,
		}))
		cfg = metrics.ServiceConfig{
			Channel:            channels,
			Group:              groups,
			Framer:             framers,
			DB:                 n.DB,
			HostKey:            n.Identity.HostKey,
			Storage:            n.Storage,
			CollectionInterval: 5 * time.Second,
		}
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
	})
	userFolder := func(ctx context.Context, name string) group.Group {
		return MustSucceed(groups.CreateOrRetrieve(ctx, name, resource.TypeChannel))
	}
	// storeDuplicate writes a channel named name in g straight to the table, as a store
	// from before the sibling rule holds it.
	storeDuplicate := func(ctx context.Context, name string, g *group.Key) {
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(
			ctx,
			channel.Channel{
				Name:        name,
				Leaseholder: node.KeyFree,
				LocalKey:    1_000_001,
				DataType:    telem.Float32T,
				Virtual:     true,
				Group:       g,
			},
		)).To(Succeed())
	}
	named := func(ctx context.Context, name string) channel.Channel {
		var ch channel.Channel
		Expect(channels.NewRetrieve().
			Where(channel.MatchNames(name)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch
	}

	It("Should open when another folder holds the name of an input of a calculated "+
		"metric", func(ctx SpecContext) {
		plant := userFolder(ctx, "Plant")
		storeDuplicate(ctx, "sy_node_1_metrics_ts_size_gb", &plant.Key)
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
	})

	It("Should open when a user moved the metrics index out of its folder and "+
		"another folder holds its name", func(ctx SpecContext) {
		idx := named(ctx, "sy_node_1_metrics_time")
		plant := userFolder(ctx, "Plant")
		lab := userFolder(ctx, "Lab")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).SetGroup(ctx, &plant.Key, idx.Key())
		})).To(Succeed())
		storeDuplicate(ctx, "sy_node_1_metrics_time", &lab.Key)
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
	})
})
