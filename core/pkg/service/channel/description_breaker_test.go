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
	"math/rand/v2"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Description breaker", func() {
	uniqueWord := func() string { return fmt.Sprintf("kw%016x", rand.Uint64()) }
	described := func(name, description string) channel.Channel {
		return channel.Channel{
			Name:        name,
			DataType:    telem.Float32T,
			Virtual:     true,
			Description: description,
		}
	}
	create := func(ctx SpecContext, ch *channel.Channel, opts ...channel.CreateOption) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, ch, opts...)
		})
	}
	apply := func(ctx SpecContext, k channel.Key, acts ...channel.Action) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.NewWriter(tx).Apply(ctx, k, acts...)
			return err
		})
	}
	setDescription := func(v any) channel.Action {
		return channel.NewSetAction(channel.SetPayload{
			Path:  []string{"description"},
			Value: v,
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
	search := func(ctx SpecContext, term string) channel.Keys {
		GinkgoHelper()
		var found []channel.Channel
		Expect(svc.NewRetrieve().Search(term).Entries(&found).Exec(ctx, db)).
			To(Succeed())
		return channel.KeysFromChannels(found)
	}

	It("Should rank a channel named by the word above one that only describes it", func(
		ctx SpecContext,
	) {
		word := uniqueWord()
		named := described(word, "Pressure at the inlet of the main tank")
		Expect(create(ctx, &named)).To(Succeed())
		mentions := described(UniqueChannelName(), word)
		Expect(create(ctx, &mentions)).To(Succeed())
		found := search(ctx, word)
		Expect(found).To(ContainElements(named.Key(), mentions.Key()))
		Expect(found[0]).To(Equal(named.Key()))
	})

	It("Should rank an exact name above a longer name, whatever its description", func(
		ctx SpecContext,
	) {
		word := uniqueWord()
		exact := described(word, "Pressure at the inlet of the main tank")
		Expect(create(ctx, &exact)).To(Succeed())
		longer := described(word+"_2", "")
		Expect(create(ctx, &longer)).To(Succeed())
		found := search(ctx, word)
		Expect(found).To(ContainElements(exact.Key(), longer.Key()))
		Expect(found[0]).To(Equal(exact.Key()))
	})

	It("Should drop the old words of a description from search after a set", func(
		ctx SpecContext,
	) {
		before, after := uniqueWord(), uniqueWord()
		ch := described(UniqueChannelName(), "flow "+before)
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(search(ctx, before)).To(ContainElement(ch.Key()))
		Expect(apply(ctx, ch.Key(), setDescription("flow "+after))).To(Succeed())
		Expect(search(ctx, before)).ToNot(ContainElement(ch.Key()))
		Expect(search(ctx, after)).To(ContainElement(ch.Key()))
	})

	It("Should drop a deleted channel from description search", func(ctx SpecContext) {
		word := uniqueWord()
		ch := described(UniqueChannelName(), word)
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		Expect(search(ctx, word)).ToNot(ContainElement(ch.Key()))
	})

	It("Should refuse a set of the description to a number", func(ctx SpecContext) {
		ch := described(UniqueChannelName(), "kept")
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(apply(ctx, ch.Key(), setDescription(5))).
			To(MatchError(validate.ErrValidation))
		Expect(stored(ctx, ch.Key()).Description).To(Equal("kept"))
	})

	It("Should refuse a batch that renames and sets a description over the cap", func(
		ctx SpecContext,
	) {
		ch := described(UniqueChannelName(), "kept")
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(apply(
			ctx,
			ch.Key(),
			channel.NewRenameAction(channel.RenamePayload{Name: UniqueChannelName()}),
			setDescription(strings.Repeat("d", 1025)),
		)).To(MatchError(validate.ErrValidation))
		got := stored(ctx, ch.Key())
		Expect(got.Name).To(Equal(ch.Name))
		Expect(got.Description).To(Equal("kept"))
	})

	It("Should count the cap in code points, taking 1,024 astral characters", func(
		ctx SpecContext,
	) {
		ch := described(UniqueChannelName(), strings.Repeat("\U0001F600", 1024))
		Expect(create(ctx, &ch)).To(Succeed())
		over := described(UniqueChannelName(), strings.Repeat("\U0001F600", 1025))
		Expect(create(ctx, &over)).To(MatchError(validate.ErrValidation))
	})

	It("Should clear or refuse a null set of the description, never half apply it", func(
		ctx SpecContext,
	) {
		ch := described(UniqueChannelName(), "gone")
		Expect(create(ctx, &ch)).To(Succeed())
		if err := apply(ctx, ch.Key(), setDescription(nil)); err != nil {
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(stored(ctx, ch.Key()).Description).To(Equal("gone"))
			return
		}
		Expect(stored(ctx, ch.Key()).Description).To(BeEmpty())
	})

	It("Should keep the stored description on a retrieve-if-name-exists create", func(
		ctx SpecContext,
	) {
		ch := described(UniqueChannelName(), "first")
		Expect(create(ctx, &ch)).To(Succeed())
		again := described(ch.Name, "second")
		Expect(create(ctx, &again, channel.RetrieveIfNameExists())).To(Succeed())
		Expect(again.Key()).To(Equal(ch.Key()))
		Expect(again.Description).To(Equal("first"))
		Expect(stored(ctx, ch.Key()).Description).To(Equal("first"))
	})

	It("Should keep path actions working on a channel whose description it took", func(
		ctx SpecContext,
	) {
		ch := described(UniqueChannelName(), "bad \xff byte")
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(apply(ctx, ch.Key(), setDescription("fixed"))).To(Succeed())
		Expect(stored(ctx, ch.Key()).Description).To(Equal("fixed"))
	})

	It("Should find a channel with no description by its name", func(ctx SpecContext) {
		word := uniqueWord()
		ch := described(word, "")
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(search(ctx, word)).To(HaveExactElements(ch.Key()))
	})
})
