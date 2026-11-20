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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Used breaker", func() {
	retrieveUsed := func(ctx SpecContext, used bool, keys ...channel.Key) []channel.Key {
		GinkgoHelper()
		var found []channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			Where(channel.MatchUsed(used)).
			Entries(&found).
			Exec(ctx, db)).To(Succeed())
		return channel.KeysFromChannels(found)
	}

	It("Should not list as unused an index channel whose delete is refused", func(
		ctx SpecContext,
	) {
		idx := channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Leaseholder: 1,
		}
		data := channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.Float64T,
			Leaseholder: 1,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			if err := w.Create(ctx, &idx); err != nil {
				return err
			}
			data.LocalIndex = idx.LocalKey
			return w.Create(ctx, &data)
		})).To(Succeed())
		unused := retrieveUsed(ctx, false, idx.Key(), data.Key())
		Expect(unused).ToNot(BeEmpty())
		for _, k := range unused {
			tx := db.OpenTx()
			err := svc.NewWriter(tx).Delete(ctx, k, false)
			Expect(tx.Close()).To(Succeed())
			Expect(err).ToNot(HaveOccurred(), "channel %d is listed as unused", k)
		}
	})

	It("Should list each data channel as a reader of its index", func(
		ctx SpecContext,
	) {
		idx := channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Leaseholder: 1,
		}
		data := channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.Float64T,
			Leaseholder: 1,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			if err := w.Create(ctx, &idx); err != nil {
				return err
			}
			data.LocalIndex = idx.LocalKey
			return w.Create(ctx, &data)
		})).To(Succeed())
		Expect(svc.Uses(ctx, db, []channel.Key{idx.Key()})).To(Equal([][]channel.Use{{{
			Holder:    gorp.NewTarget("channel", data.Key()),
			Direction: channel.DirectionRead,
		}}}))
	})

	// Calculations name the channels they read in text and store no link to them yet,
	// so `used` leaves them out, as its doc says.
	It("Should count only stored links, so a calculation's input is unused", func(
		ctx SpecContext,
	) {
		data := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		calc := channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float32T,
			Virtual:    true,
			Expression: "return " + data.Name + " * 2",
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			if err := w.Create(ctx, &data); err != nil {
				return err
			}
			return w.Create(ctx, &calc)
		})).To(Succeed())
		Expect(retrieveUsed(ctx, true, data.Key())).To(BeEmpty())
	})
})
