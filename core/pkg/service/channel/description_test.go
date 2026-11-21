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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Description", func() {
	described := func(description string) channel.Channel {
		return channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.Float32T,
			Virtual:     true,
			Description: description,
		}
	}
	create := func(ctx SpecContext, ch *channel.Channel) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, ch)
		})
	}
	stored := func(ctx SpecContext, k channel.Key) channel.Channel {
		GinkgoHelper()
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch
	}
	tooLong := SatisfyAll(
		MatchError(validate.ErrValidation),
		MatchError(ContainSubstring("description")),
	)

	It("Should store the description of a created channel", func(ctx SpecContext) {
		ch := described("Pressure at the inlet of the main tank")
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(stored(ctx, ch.Key()).Description).
			To(Equal("Pressure at the inlet of the main tank"))
	})

	It("Should take a description of 1,024 bytes and refuse one of 1,025", func(
		ctx SpecContext,
	) {
		longest := described(strings.Repeat("d", 1024))
		Expect(create(ctx, &longest)).To(Succeed())
		over := described(strings.Repeat("d", 1025))
		Expect(create(ctx, &over)).To(tooLong)
	})

	It("Should set a description through Apply, within the limit", func(
		ctx SpecContext,
	) {
		ch := described("")
		Expect(create(ctx, &ch)).To(Succeed())
		set := func(description string) error {
			return db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := svc.NewWriter(tx).Apply(
					ctx,
					ch.Key(),
					channel.NewSetAction(channel.SetPayload{
						Path:  []string{"description"},
						Value: description,
					}),
				)
				return err
			})
		}
		Expect(set("Valve command from the sequence")).To(Succeed())
		Expect(
			stored(ctx, ch.Key()).Description,
		).To(Equal("Valve command from the sequence"))
		Expect(set(strings.Repeat("d", 1025))).To(tooLong)
		Expect(
			stored(ctx, ch.Key()).Description,
		).To(Equal("Valve command from the sequence"))
	})

	It("Should find a channel by a word of its description", func(ctx SpecContext) {
		word := "zephyrium" + strings.ReplaceAll(UniqueChannelName(), "_", "")
		ch := described("Flow through the " + word + " line")
		Expect(create(ctx, &ch)).To(Succeed())
		var found []channel.Channel
		Expect(svc.NewRetrieve().Search(word).Entries(&found).Exec(ctx, db)).
			To(Succeed())
		Expect(channel.KeysFromChannels(found)).To(ContainElement(ch.Key()))
	})
})
