// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/log/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

func channelsOf(l log.Log) []channel.Key {
	keys := make([]channel.Key, len(l.Channels))
	for i, e := range l.Channels {
		keys[i] = e.Channel
	}
	return keys
}

var _ = Describe("Channel references", func() {
	retrieve := func(ctx SpecContext, db *gorp.DB, key log.Key) log.Log {
		GinkgoHelper()
		var res log.Log
		Expect(gorp.NewRetrieve[log.Key, log.Log]().
			Where(gorp.MatchKeys[log.Key, log.Log](key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	It("Should refuse a channel the Core does not hold", func(ctx SpecContext) {
		l := log.Log{
			Project:  &proj.Key,
			Name:     "missing",
			Channels: []log.ChannelEntry{{Channel: ch5}, {Channel: 404}},
		}
		Expect(svc.NewWriter(nil).Create(ctx, &l)).To(MatchError(
			ContainSubstring("channels.channel: channel 404 does not exist"),
		))
	})

	It(
		"Should remove the entry of a deleted channel and keep the others",
		func(ctx SpecContext) {
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, channelSvc, 3))
			first, gone, third := keys[0], keys[1], keys[2]
			l := log.Log{
				Project: &proj.Key,
				Name:    "detach",
				Channels: []log.ChannelEntry{
					{Channel: first, Alias: "first"},
					{Channel: gone},
					{Channel: third, Alias: "third"},
				},
			}
			Expect(svc.NewWriter(nil).Create(ctx, &l)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, l.Key)).To(Succeed())
			})
			Expect(channelSvc.NewWriter(nil).Delete(ctx, gone, false)).To(Succeed())
			res := retrieve(ctx, db, l.Key)
			Expect(channelsOf(res)).To(Equal([]channel.Key{first, third}))
			Expect(res.Channels[0].Alias).To(Equal("first"))
			Expect(res.Channels[1].Alias).To(Equal("third"))
		},
	)

	It(
		"Should drop channels the Core does not hold from an imported log",
		func(ctx SpecContext) {
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, channelSvc, 2))
			kept, gone := keys[0], keys[1]
			l := log.Log{
				Project:  &proj.Key,
				Name:     "export",
				Channels: []log.ChannelEntry{{Channel: kept}, {Channel: gone}},
			}
			Expect(svc.NewWriter(nil).Create(ctx, &l)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, l.ResourceID()))
			Expect(channelSvc.NewWriter(nil).Delete(ctx, gone, false)).To(Succeed())
			id := MustSucceed(imexSvc.Import(
				ctx,
				db,
				WireRoundTrip(env),
				imex.ImportOptions{Parent: proj.ResourceID()},
			))
			res := retrieve(ctx, db, MustSucceed(uuid.Parse(id.Key)))
			Expect(channelsOf(res)).To(Equal([]channel.Key{kept}))
		},
	)

	It("Should drop channels the Core does not hold on open", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
		}))
		channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel:      node.Channel,
			DB:           node.DB,
			HostProvider: node.Cluster,
			Ontology:     otg,
			Status:       statusSvc,
		}))
		kept := MustSucceed(channeltestutil.CreateVirtual(ctx, channelSvc, 1))[0]
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         node.DB,
			Namespace:  "Log",
			Migrations: versions.Migrations,
		})).To(Succeed())
		stored := log.Log{
			Key:      uuid.New(),
			Name:     "log",
			Channels: []log.ChannelEntry{{Channel: 6}, {Channel: kept}},
		}
		Expect(gorp.NewCreate[log.Key, log.Log]().
			Entry(&stored).
			Exec(ctx, node.DB)).To(Succeed())
		MustOpen(log.OpenService(ctx, log.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
			ImEx:     imex.NewService(),
		}))
		Expect(channelsOf(retrieve(ctx, node.DB, stored.Key))).
			To(Equal([]channel.Key{kept}))
	})
})
