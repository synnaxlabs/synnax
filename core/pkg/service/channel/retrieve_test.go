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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", Ordered, func() {
	var (
		db  *gorp.DB
		svc *channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		svc = openService(ctx, node)
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a set of channels", func(ctx SpecContext) {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).CreateMany(ctx, &created)
			})).To(Succeed())

			var resChannels []channel.Channel

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchLeaseholders(1)).
				Entries(&resChannels).
				Exec(ctx, db)).To(Succeed())
			Expect(resChannels).To(HaveLen(len(created)))
		})
		It("Should correctly retrieve a channel by its key", func(ctx SpecContext) {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).CreateMany(ctx, &created)
			})).To(Succeed())
			var resChannels []channel.Channel

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchKeys(created[0].Key())).
				Entries(&resChannels).
				Exec(ctx, db)).To(Succeed())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Key()).To(Equal(created[0].Key()))
		})
		It("Should correctly retrieve a channel by its name", func(ctx SpecContext) {
			n := UniqueChannelName()
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     n,
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).CreateMany(ctx, &created)
			})).To(Succeed())
			var resChannels []channel.Channel

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchNames(n)).
				Entries(&resChannels).
				Exec(ctx, db)).To(Succeed())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal(n))
		})
		It(
			"Should return a well formatted error if a channel cannot be found by its key",
			func(ctx SpecContext) {
				var resChannels []channel.Channel
				Expect(svc.
					NewRetrieve().
					Where(channel.MatchKeys(435)).
					Entries(&resChannels).
					Exec(ctx, db),
				).To(MatchError(ContainSubstring("Channels with keys [435] not found")))
			},
		)
		It("Should correctly filter channels by search term", func(ctx SpecContext) {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "a_completely_different_name",
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "catalina",
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).CreateMany(ctx, &created)
			})).To(Succeed())
			var resChannels []channel.Channel
			Expect(svc.
				NewRetrieve().
				Search("catalina").
				Entries(&resChannels).
				Exec(ctx, db)).To(Succeed())
			Expect(resChannels).To(HaveExactElements(HaveField("Name", "catalina")))
		})

		It(
			"Should return an error when retrieving a channel with a key of 0",
			func(ctx SpecContext) {
				var resChannels []channel.Channel
				Expect(svc.
					NewRetrieve().
					Where(channel.MatchKeys(0)).
					Entries(&resChannels).
					Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			},
		)
	})
	Describe("MatchVirtual", func() {
		It("Should match every virtual channel, free ones too", func(ctx SpecContext) {
			plain := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			free := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.Float32T,
				Virtual:     true,
				Leaseholder: node.KeyFree,
			}
			stored := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := svc.NewWriter(tx)
				if err := w.Create(ctx, &plain); err != nil {
					return err
				}
				if err := w.Create(ctx, &stored); err != nil {
					return err
				}
				return w.Create(ctx, &free, channel.FreeIndex())
			})).To(Succeed())
			match := func(virtual bool) channel.Keys {
				GinkgoHelper()
				var res []channel.Channel
				Expect(svc.NewRetrieve().
					Where(channel.MatchVirtual(virtual)).
					Entries(&res).
					Exec(ctx, db)).To(Succeed())
				return channel.KeysFromChannels(res)
			}
			Expect(match(true)).To(ContainElements(plain.Key(), free.Key()))
			Expect(match(true)).ToNot(ContainElement(stored.Key()))
			Expect(match(false)).To(ContainElement(stored.Key()))
			Expect(match(false)).ToNot(ContainElements(plain.Key(), free.Key()))
		})
	})

	Describe("MatchIndexes", func() {
		createIndexed := func(ctx SpecContext, prefix string) (
			channel.Channel, []channel.Channel,
		) {
			GinkgoHelper()
			idx := channel.Channel{
				Name: prefix + "_time", DataType: telem.TimestampT, IsIndex: true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &idx)
			})).To(Succeed())
			data := []channel.Channel{
				{
					Name:       prefix + "_a",
					DataType:   telem.Float32T,
					LocalIndex: idx.LocalKey,
				},
				{
					Name:       prefix + "_b",
					DataType:   telem.Float32T,
					LocalIndex: idx.LocalKey,
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).CreateMany(ctx, &data)
			})).To(Succeed())
			return idx, data
		}
		retrieveKeys := func(ctx SpecContext, keys ...channel.Key) channel.Keys {
			GinkgoHelper()
			var results []channel.Channel
			Expect(svc.NewRetrieve().
				Where(svc.MatchIndexes(keys...)).
				Entries(&results).
				Exec(ctx, db)).To(Succeed())
			return channel.KeysFromChannels(results)
		}

		It(
			"Should match every data channel indexed by the key",
			func(ctx SpecContext) {
				idx, data := createIndexed(ctx, "mi_stored")
				Expect(retrieveKeys(ctx, idx.Key())).To(ConsistOf(
					data[0].Key(), data[1].Key(),
				))
			},
		)

		It("Should match a free channel by its free index", func(ctx SpecContext) {
			free := channel.Channel{
				Name:        "mi_free",
				DataType:    telem.Float32T,
				Virtual:     true,
				Leaseholder: node.KeyFree,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &free, channel.FreeIndex())
			})).To(Succeed())
			Expect(retrieveKeys(ctx, free.Index())).To(ConsistOf(free.Key()))
		})

		It("Should match across several index keys", func(ctx SpecContext) {
			first, firstData := createIndexed(ctx, "mi_first")
			second, secondData := createIndexed(ctx, "mi_second")
			Expect(retrieveKeys(ctx, first.Key(), second.Key())).To(ConsistOf(
				firstData[0].Key(), firstData[1].Key(),
				secondData[0].Key(), secondData[1].Key(),
			))
		})

		It("Should reject a same local index on another leaseholder", func(
			ctx SpecContext,
		) {
			idx, _ := createIndexed(ctx, "mi_twin")
			Expect(retrieveKeys(ctx, distchannel.NewKey(2, idx.LocalKey))).To(BeEmpty())
		})

		It("Should match nothing when given no keys", func(ctx SpecContext) {
			createIndexed(ctx, "mi_none")
			Expect(retrieveKeys(ctx)).To(BeEmpty())
		})

		It("Should stop matching a channel once it is deleted", func(ctx SpecContext) {
			idx, data := createIndexed(ctx, "mi_deleted")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, data[0].Key(), false)
			})).To(Succeed())
			Expect(retrieveKeys(ctx, idx.Key())).To(ConsistOf(data[1].Key()))
		})
	})

	Describe("Exists", func() {
		It("Should return true if a channel exists", func(ctx SpecContext) {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).CreateMany(ctx, &created)
			})).To(Succeed())

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchKeys(created[0].Key())).
				Exists(ctx, db),
			).To(BeTrue())
		})
	})
})
