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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Name breaker", Ordered, ContinueOnFailure, func() {
	var (
		db   *gorp.DB
		bsvc *channel.Service
		rec  *Recorder[channel.Key, channel.Action]
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, _ := serviceConfig(ctx, node, node.DB)
		// A Core started with the old flag, as the Names suite models it.
		cfg.ValidateNames = new(false)
		bsvc = MustOpen(channel.OpenService(ctx, cfg))
	})
	BeforeEach(func() {
		rec = &Recorder[channel.Key, channel.Action]{}
		DeferCleanup(bsvc.Actions().OnChange(rec.Record))
	})
	virtual := func(name string) channel.Channel {
		return channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
	}
	create := func(
		ctx context.Context,
		ch *channel.Channel,
		opts ...channel.CreateOption,
	) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return bsvc.NewWriter(tx).Create(ctx, ch, opts...)
		})
	}
	stored := func(ctx context.Context, k channel.Key) channel.Channel {
		GinkgoHelper()
		var ch channel.Channel
		Expect(bsvc.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch
	}
	// storeName writes name past the writer, as a Core that ran with the old flag
	// stored it.
	storeName := func(ctx context.Context, k channel.Key, name string) {
		GinkgoHelper()
		ch := stored(ctx, k)
		ch.Name = name
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, ch)).
			To(Succeed())
	}
	quoted := func(s string) string { return fmt.Sprintf("%+q", s) }
	// lastRename returns the name of the last rename action the writer attached,
	// quoted so that two normal forms print apart.
	lastRename := func() string {
		GinkgoHelper()
		frames := rec.Snapshot()
		Expect(frames).ToNot(BeEmpty())
		acts := frames[len(frames)-1].Actions
		Expect(acts).To(HaveLen(1))
		Expect(acts[0].Rename).ToNot(BeNil())
		return quoted(acts[0].Rename.Name)
	}

	It("Should broadcast a renamed name in the form that the Core stores",
		func(ctx SpecContext) {
			ch := virtual(UniqueChannelName())
			Expect(create(ctx, &ch)).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return bsvc.NewWriter(tx).
					Rename(ctx, ch.Key(), "n\u0303_brk_rename", false)
			})).To(Succeed())
			Expect(quoted(stored(ctx, ch.Key()).Name)).
				To(Equal(quoted("\u00F1_brk_rename")))
			Expect(lastRename()).To(Equal(quoted("\u00F1_brk_rename")))
		})

	It("Should broadcast a name renamed through Apply in the form that the Core stores",
		func(ctx SpecContext) {
			ch := virtual(UniqueChannelName())
			Expect(create(ctx, &ch)).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := bsvc.NewWriter(tx).Apply(
					ctx,
					ch.Key(),
					channel.NewRenameAction(
						channel.RenamePayload{Name: "n\u0303_brk_apply"},
					),
				)
				return err
			})).To(Succeed())
			Expect(quoted(stored(ctx, ch.Key()).Name)).
				To(Equal(quoted("\u00F1_brk_apply")))
			Expect(lastRename()).To(Equal(quoted("\u00F1_brk_apply")))
		})

	It("Should retrieve a stored channel by the exact name that it holds",
		func(ctx SpecContext) {
			ch := virtual(UniqueChannelName())
			Expect(create(ctx, &ch)).To(Succeed())
			storeName(ctx, ch.Key(), "o\u0308_brk_retrieve")
			again := virtual("o\u0308_brk_retrieve")
			Expect(create(ctx, &again, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(again.Key()).To(Equal(ch.Key()))
		})

	It("Should refuse a name that a stored channel holds in another normal form",
		func(ctx SpecContext) {
			ch := virtual(UniqueChannelName())
			Expect(create(ctx, &ch)).To(Succeed())
			storeName(ctx, ch.Key(), "o\u0308_brk_clash")
			other := virtual("\u00F6_brk_clash")
			Expect(create(ctx, &other)).
				To(MatchError(ContainSubstring("already exists")))
		})

	It("Should find a created channel by the name the create sent",
		func(ctx SpecContext) {
			ch := virtual("n\u0303_brk_lookup")
			Expect(create(ctx, &ch)).To(Succeed())
			// A lookup enters through a Where, which the API puts in NFC.
			name := "n\u0303_brk_lookup"
			lookup := MustSucceed(channel.MatchWhere(channel.NormalizeWhere(
				channel.Where{Name: &filter.String{Eq: &name}},
			)))
			Expect(bsvc.NewRetrieve().Where(lookup).Exists(ctx, db)).To(BeTrue())
		})

	It("Should delete a channel by the name the create sent", func(ctx SpecContext) {
		ch := virtual("n\u0303_brk_delete")
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return bsvc.NewWriter(tx).
				DeleteManyByNames(ctx, []string{"n\u0303_brk_delete"}, false)
		})).To(Succeed())
		Expect(bsvc.NewRetrieve().
			Where(channel.MatchKeys(ch.Key())).
			Exists(ctx, db)).To(BeFalse())
	})

	It("Should name the position of a clashing channel in a create of many",
		func(ctx SpecContext) {
			taken := virtual(UniqueChannelName())
			Expect(create(ctx, &taken)).To(Succeed())
			chs := []channel.Channel{virtual(UniqueChannelName()), virtual(taken.Name)}
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				return bsvc.NewWriter(tx).CreateMany(ctx, &chs)
			})
			Expect(err).To(MatchError(validate.ErrValidation))
			pathErr, ok := errors.AsType[validate.PathError](err)
			Expect(ok).To(BeTrue())
			Expect(pathErr.Path).To(Equal([]string{"1", "name"}))
		})

	It("Should not use up channel keys on a create that the name index refuses",
		func(ctx SpecContext) {
			index := func(name string) channel.Channel {
				return channel.Channel{
					Name:     name,
					DataType: telem.TimestampT,
					IsIndex:  true,
				}
			}
			first := index(UniqueChannelName())
			Expect(create(ctx, &first)).To(Succeed())
			for range 3 {
				dup := index(first.Name)
				Expect(create(ctx, &dup)).
					To(MatchError(ContainSubstring("already exists")))
			}
			next := index(UniqueChannelName())
			Expect(create(ctx, &next)).To(Succeed())
			Expect(next.LocalKey).To(Equal(first.LocalKey + 1))
		})

	It("Should refuse a name that shows as nothing", func(ctx SpecContext) {
		ch := virtual("\u2800")
		Expect(create(ctx, &ch)).
			To(MatchError(ContainSubstring("must hold a visible character")))
	})

	// The cases below pass. They pin what the breaker checked.

	It("Should refuse two names in one request that are equal after NFC",
		func(ctx SpecContext) {
			chs := []channel.Channel{
				virtual("\u00E9_brk_staged"),
				virtual("e\u0301_brk_staged"),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return bsvc.NewWriter(tx).CreateMany(ctx, &chs)
			})).To(MatchError(ContainSubstring("duplicate channel name")))
		})

	It("Should refuse two creates of one name in one transaction and free it after",
		func(ctx SpecContext) {
			a, b := virtual("brk_staged_tx"), virtual("brk_staged_tx")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := bsvc.NewWriter(tx)
				if err := w.Create(ctx, &a); err != nil {
					return err
				}
				return w.Create(ctx, &b)
			})).To(MatchError(ContainSubstring(
				"channel with name 'brk_staged_tx' already exists",
			)))
			c := virtual("brk_staged_tx")
			Expect(create(ctx, &c)).To(Succeed())
		})

	It("Should rename a channel to the NFC form of its own stored name",
		func(ctx SpecContext) {
			ch := virtual(UniqueChannelName())
			Expect(create(ctx, &ch)).To(Succeed())
			storeName(ctx, ch.Key(), "o\u0308_brk_self")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return bsvc.NewWriter(tx).
					Rename(ctx, ch.Key(), "\u00F6_brk_self", false)
			})).To(Succeed())
			Expect(quoted(stored(ctx, ch.Key()).Name)).
				To(Equal(quoted("\u00F6_brk_self")))
		})

	It("Should swap two names through a free name in one transaction",
		func(ctx SpecContext) {
			a, b := virtual("brk_swap_a"), virtual("brk_swap_b")
			Expect(create(ctx, &a)).To(Succeed())
			Expect(create(ctx, &b)).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := bsvc.NewWriter(tx)
				if err := w.Rename(ctx, a.Key(), "brk_swap_tmp", false); err != nil {
					return err
				}
				if err := w.Rename(ctx, b.Key(), "brk_swap_a", false); err != nil {
					return err
				}
				return w.Rename(ctx, a.Key(), "brk_swap_b", false)
			})).To(Succeed())
			Expect(stored(ctx, a.Key()).Name).To(Equal("brk_swap_b"))
			Expect(stored(ctx, b.Key()).Name).To(Equal("brk_swap_a"))
		})

	It("Should refuse a swap in one RenameMany, as the name query did",
		func(ctx SpecContext) {
			a, b := virtual("brk_many_a"), virtual("brk_many_b")
			Expect(create(ctx, &a)).To(Succeed())
			Expect(create(ctx, &b)).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return bsvc.NewWriter(tx).RenameMany(
					ctx,
					channel.Keys{a.Key(), b.Key()},
					[]string{"brk_many_b", "brk_many_a"},
					false,
				)
			})).To(MatchError(ContainSubstring("already exists")))
		})

	It("Should rename onto the name of a channel deleted in the same transaction",
		func(ctx SpecContext) {
			a, b := virtual("brk_deleted"), virtual(UniqueChannelName())
			Expect(create(ctx, &a)).To(Succeed())
			Expect(create(ctx, &b)).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := bsvc.NewWriter(tx)
				if err := w.Delete(ctx, a.Key(), false); err != nil {
					return err
				}
				return w.Rename(ctx, b.Key(), "brk_deleted", false)
			})).To(Succeed())
			Expect(stored(ctx, b.Key()).Name).To(Equal("brk_deleted"))
		})

	It("Should refuse a calculated channel whose index name a channel holds",
		func(ctx SpecContext) {
			taken := virtual("brk_calc_time")
			Expect(create(ctx, &taken)).To(Succeed())
			calc := channel.Channel{
				Name:       "brk_calc",
				DataType:   telem.Float64T,
				Expression: "return 1.0",
			}
			Expect(create(ctx, &calc)).To(MatchError(ContainSubstring(
				"cannot serve as the index for a calculated channel",
			)))
			Expect(bsvc.NewRetrieve().
				Where(channel.MatchNames("brk_calc")).
				Exists(ctx, db)).To(BeFalse())
		})

	It("Should declare a built-in over two stored user channels of its name",
		func(ctx SpecContext) {
			a, b := virtual(UniqueChannelName()), virtual(UniqueChannelName())
			Expect(create(ctx, &a)).To(Succeed())
			Expect(create(ctx, &b)).To(Succeed())
			storeName(ctx, a.Key(), "sy_brk_builtin")
			storeName(ctx, b.Key(), "sy_brk_builtin")
			builtIn := channel.Channel{
				Name:     "sy_brk_builtin",
				DataType: telem.Uint8T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return bsvc.NewWriter(tx).Declare(ctx, &builtIn)
			})).To(Succeed())
			names := []string{stored(ctx, a.Key()).Name, stored(ctx, b.Key()).Name}
			Expect(names).To(ConsistOf("sy_brk_builtin_2", "sy_brk_builtin_3"))
			Expect(bsvc.IsBuiltIn(builtIn.Key())).To(BeTrue())
		})

	It("Should keep leading and trailing spaces as a distinct name",
		func(ctx SpecContext) {
			a, b := virtual("brk_space"), virtual(" brk_space ")
			Expect(create(ctx, &a)).To(Succeed())
			Expect(create(ctx, &b)).To(Succeed())
			Expect(stored(ctx, b.Key()).Name).To(Equal(" brk_space "))
		})

	It("Should refuse invalid UTF-8 on create and rename", func(ctx SpecContext) {
		bad := virtual("brk_bad\xff")
		Expect(create(ctx, &bad)).To(MatchError(validate.ErrValidation))
		ch := virtual(UniqueChannelName())
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return bsvc.NewWriter(tx).Rename(ctx, ch.Key(), "brk_bad\xff", false)
		})).To(MatchError(validate.ErrValidation))
	})

	It("Should refuse an illegal name renamed through Apply", func(ctx SpecContext) {
		ch := virtual(UniqueChannelName())
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := bsvc.NewWriter(tx).Apply(
				ctx,
				ch.Key(),
				channel.NewRenameAction(channel.RenamePayload{Name: "brk.apply"}),
			)
			return err
		})).To(MatchError(ContainSubstring("no '.' or invisible character")))
	})

	It("Should wrap a clash of Apply in a validation error", func(ctx SpecContext) {
		a, b := virtual(UniqueChannelName()), virtual(UniqueChannelName())
		Expect(create(ctx, &a)).To(Succeed())
		Expect(create(ctx, &b)).To(Succeed())
		err := db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := bsvc.NewWriter(tx).Apply(
				ctx,
				b.Key(),
				channel.NewRenameAction(channel.RenamePayload{Name: a.Name}),
			)
			return err
		})
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).To(MatchError(ContainSubstring(
			"channel with name '" + a.Name + "' already exists",
		)))
	})

	It("Should keep a stored illegal name through an Apply rename to itself",
		func(ctx SpecContext) {
			ch := virtual(UniqueChannelName())
			Expect(create(ctx, &ch)).To(Succeed())
			storeName(ctx, ch.Key(), "brk.kept")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := bsvc.NewWriter(tx).Apply(
					ctx,
					ch.Key(),
					channel.NewRenameAction(channel.RenamePayload{Name: "brk.kept"}),
				)
				return err
			})).To(Succeed())
			Expect(stored(ctx, ch.Key()).Name).To(Equal("brk.kept"))
		})
})
