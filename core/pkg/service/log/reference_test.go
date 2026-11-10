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
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/log/versions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/defaults"
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

	It(
		"Should attach the log as written to the delete that detaches a channel",
		func(ctx SpecContext) {
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 2))
			gone, kept := keys[0], keys[1]
			l := defaults.Fill(log.Log{
				Project:  &proj.Key,
				Name:     "edited",
				Channels: []log.ChannelEntry{{Channel: gone}, {Channel: kept}},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &l)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, l.Key)
				})).To(Succeed())
			})
			rec := &Recorder[log.Key, log.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, gone, false)
			})).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(l.Key))
			Expect(seen[0].Actions).To(HaveLen(1))
			Expect(seen[0].Actions[0].Create).ToNot(BeNil())
			Expect(channelsOf(seen[0].Actions[0].Create.Log)).
				To(Equal([]channel.Key{kept}))
		},
	)

	It("Should refuse a channel the Core does not hold", func(ctx SpecContext) {
		l := defaults.Fill(log.Log{
			Project:  &proj.Key,
			Name:     "missing",
			Channels: []log.ChannelEntry{{Channel: ch5}, {Channel: 404}},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &l)
		})).To(MatchError(
			ContainSubstring("channels.channel: channel 404 does not exist"),
		))
	})

	It(
		"Should remove the entry of a deleted channel and keep the others",
		func(ctx SpecContext) {
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 3))
			first, gone, third := keys[0], keys[1], keys[2]
			l := defaults.Fill(log.Log{
				Project: &proj.Key,
				Name:    "detach",
				Channels: []log.ChannelEntry{
					{Channel: first, Alias: "first"},
					{Channel: gone},
					{Channel: third, Alias: "third"},
				},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &l)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, l.Key)
				})).To(Succeed())
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, gone, false)
			})).To(Succeed())
			res := retrieve(ctx, db, l.Key)
			Expect(channelsOf(res)).To(Equal([]channel.Key{first, third}))
			Expect(res.Channels[0].Alias).To(Equal("first"))
			Expect(res.Channels[1].Alias).To(Equal("third"))
		},
	)

	It(
		"Should drop channels the Core does not hold from an imported log",
		func(ctx SpecContext) {
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 2))
			kept, gone := keys[0], keys[1]
			l := defaults.Fill(log.Log{
				Project:  &proj.Key,
				Name:     "export",
				Channels: []log.ChannelEntry{{Channel: kept}, {Channel: gone}},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &l)
			})).To(Succeed())
			env := MustSucceed(svc.Export(ctx, db, l.ResourceID()))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, gone, false)
			})).To(Succeed())
			var id resource.ID
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				id, err = imexSvc.Import(ctx, tx, WireRoundTrip(env),
					imex.ImportOptions{Parent: proj.ResourceID()})
				return err
			})).To(Succeed())
			res := retrieve(ctx, db, MustSucceed(uuid.Parse(id.Key)))
			Expect(channelsOf(res)).To(Equal([]channel.Key{kept}))
		},
	)

	It("Should drop channels the Core does not hold on open", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: node.DB,
		}))
		channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statusSvc,
		}))
		kept := MustSucceed(
			channeltestutil.CreateVirtual(ctx, node.DB, channelSvc, 1),
		)[0]
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         node.DB,
			Namespace:  "Log",
			Migrations: versions.Migrations,
		})).To(Succeed())
		stored := defaults.Fill(log.Log{
			Key:      uuid.New(),
			Name:     "log",
			Channels: []log.ChannelEntry{{Channel: 6}, {Channel: kept}},
		})
		Expect(gorp.NewCreate[log.Key, log.Log]().
			Entry(&stored).
			Exec(ctx, node.DB)).To(Succeed())
		MustOpen(log.OpenService(ctx, log.ServiceConfig{
			DB:   node.DB,
			ImEx: imex.NewService(),
		}))
		Expect(channelsOf(retrieve(ctx, node.DB, stored.Key))).
			To(Equal([]channel.Key{kept}))
	})
})
