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
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/metrics"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Folder breaker 2", func() {
	var (
		cfg      metrics.ServiceConfig
		groups   *group.Service
		channels *channel.Service
		db       *gorp.DB
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		groups = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: node.DB}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: node.DB}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: node.DB}))
		channels = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		graph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
			DB:      node.DB,
			Channel: channels,
			Status:  statuses,
		}))
		framers := MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
			DB:           node.DB,
			Framer:       node.Framer,
			Channel:      channels,
			ChannelGraph: graph,
		}))
		cfg = metrics.ServiceConfig{
			Channel:            channels,
			Group:              groups,
			Framer:             framers,
			DB:                 node.DB,
			HostKey:            node.Identity.HostKey,
			Storage:            node.Storage,
			CollectionInterval: 5 * time.Second,
		}
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
	})
	folder := func(ctx context.Context) group.Group {
		var members []channel.Channel
		Expect(channels.NewRetrieve().
			Where(channel.MatchNames("sy_node_1_metrics_time")).
			Entries(&members).
			Exec(ctx, db)).To(Succeed())
		Expect(members).To(HaveLen(1))
		Expect(members[0].Group).ToNot(BeNil())
		var g group.Group
		Expect(groups.NewRetrieve().
			Where(group.MatchKeys(*members[0].Group)).
			Entry(&g).
			Exec(ctx, db)).To(Succeed())
		return g
	}
	rootChannel := func(ctx context.Context, tx gorp.Tx, name string) error {
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		return channels.NewWriter(tx).Create(ctx, &ch)
	}

	It("Should open after a user deletes the folder and names a root channel Metrics",
		func(ctx SpecContext) {
			f := folder(ctx)
			var members []channel.Channel
			Expect(channels.NewRetrieve().
				Where(channel.MatchGroups(f.Key)).
				Entries(&members).
				Exec(ctx, db)).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				keys := channel.KeysFromChannels(members)
				w := channels.NewWriter(tx)
				if err := w.SetGroup(ctx, nil, keys...); err != nil {
					return err
				}
				if err := groups.NewWriter(tx).Delete(ctx, f.Key); err != nil {
					return err
				}
				return rootChannel(ctx, tx, "Metrics")
			})).To(Succeed())
			Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
		},
	)

	It("Should open a v0.62 store whose Metrics folder was renamed and a root "+
		"channel holds the name Metrics", func(ctx SpecContext) {
		f := folder(ctx)
		// A v0.62 Core never wrote the folder key.
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := tx.Delete(ctx, []byte("sy_metrics_folder")); err != nil {
				return err
			}
			gw := groups.NewWriter(tx)
			if err := gw.Rename(ctx, f.Key, "Host metrics"); err != nil {
				return err
			}
			return rootChannel(ctx, tx, "Metrics")
		})).To(Succeed())
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
	})

	It("Should keep the stored folder after a user moves it under another folder",
		func(ctx SpecContext) {
			f := folder(ctx)
			parent := MustSucceed(
				groups.CreateOrRetrieve(ctx, "Plant", resource.TypeChannel),
			)
			id := parent.ResourceID()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groups.NewWriter(tx).SetParent(ctx, &id, f.Key)
			})).To(Succeed())
			Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
			Expect(folder(ctx).Key).To(Equal(f.Key))
			var roots []group.Group
			Expect(groups.NewRetrieve().
				Where(group.MatchNames("Metrics")).
				Entries(&roots).
				Exec(ctx, db)).To(Succeed())
			Expect(roots).To(HaveLen(1))
		},
	)

	It("Should keep the stored folder after a user renames it", func(ctx SpecContext) {
		f := folder(ctx)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groups.NewWriter(tx).Rename(ctx, f.Key, "Host metrics")
		})).To(Succeed())
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
		var named []group.Group
		Expect(groups.NewRetrieve().
			Where(group.MatchNames("Metrics")).
			Entries(&named).
			Exec(ctx, db)).To(Succeed())
		Expect(named).To(BeEmpty())
	})

	It("Should take a user's root folder named Metrics after the stored one is gone",
		func(ctx SpecContext) {
			f := folder(ctx)
			var members []channel.Channel
			Expect(channels.NewRetrieve().
				Where(channel.MatchGroups(f.Key)).
				Entries(&members).
				Exec(ctx, db)).To(Succeed())
			var mine group.Group
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				keys := channel.KeysFromChannels(members)
				w := channels.NewWriter(tx)
				if err := w.SetGroup(ctx, nil, keys...); err != nil {
					return err
				}
				if err := groups.NewWriter(tx).Delete(ctx, f.Key); err != nil {
					return err
				}
				mine = group.Group{Name: "Metrics", Scope: resource.TypeChannel}
				return groups.NewWriter(tx).Create(ctx, &mine)
			})).To(Succeed())
			Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
			Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
		},
	)
})
