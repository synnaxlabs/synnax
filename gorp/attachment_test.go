// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Attachment", func() {
	var (
		db      *gorp.DB
		records *gorp.Table[int32, entry]
		notes   *gorp.Attachment[string]
		create  = func(ctx context.Context, tx gorp.Tx, key int32) error {
			return records.NewCreate().
				Entry(&entry{ID: key, Data: "data"}).
				Exec(ctx, tx)
		}
		collect = func() *[]gorp.Commit {
			var commits []gorp.Commit
			DeferCleanup(db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
				commits = append(commits, c)
			}))
			return &commits
		}
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		records = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
			DB:   db,
			Name: "record",
		}))
		notes = gorp.NewAttachment[string]()
	})

	It("Should deliver values in the order they were added", func(ctx SpecContext) {
		commits := collect()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			notes.Add(tx, "first")
			if err := create(ctx, tx, 1); err != nil {
				return err
			}
			notes.Add(tx, "second", "third")
			return nil
		})).To(Succeed())
		Expect(*commits).To(HaveLen(1))
		Expect(notes.Of((*commits)[0])).To(Equal([]string{"first", "second", "third"}))
	})

	It("Should keep two attachments of one type apart", func(ctx SpecContext) {
		var (
			commits = collect()
			other   = gorp.NewAttachment[string]()
			unused  = gorp.NewAttachment[string]()
		)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			notes.Add(tx, "a")
			other.Add(tx, "b")
			notes.Add(tx, "c")
			return nil
		})).To(Succeed())
		Expect(*commits).To(HaveLen(1))
		Expect(notes.Of((*commits)[0])).To(Equal([]string{"a", "c"}))
		Expect(other.Of((*commits)[0])).To(Equal([]string{"b"}))
		Expect(unused.Of((*commits)[0])).To(BeEmpty())
	})

	It("Should not carry values into the next transaction", func(ctx SpecContext) {
		commits := collect()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			notes.Add(tx, "first")
			return create(ctx, tx, 1)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return create(ctx, tx, 2)
		})).To(Succeed())
		Expect(*commits).To(HaveLen(2))
		Expect(notes.Of((*commits)[0])).To(Equal([]string{"first"}))
		Expect(notes.Of((*commits)[1])).To(BeEmpty())
	})

	It(
		"Should deliver no values for a transaction that closes without a commit",
		func(ctx SpecContext) {
			commits := collect()
			tx := db.OpenTx()
			notes.Add(tx, "closed")
			Expect(create(ctx, tx, 1)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
			// A ticket that the closed transaction took would block this commit.
			committed := make(chan error, 1)
			go func() {
				committed <- db.WithTx(ctx, func(tx gorp.Tx) error {
					notes.Add(tx, "committed")
					return nil
				})
			}()
			Eventually(committed).Should(Receive(Succeed()))
			Expect(*commits).To(HaveLen(1))
			Expect((*commits)[0].Seq).To(Equal(uint64(1)))
			Expect(notes.Of((*commits)[0])).To(Equal([]string{"committed"}))
		},
	)

	It(
		"Should deliver and number a commit that carries values and changes no record",
		func(ctx SpecContext) {
			commits := collect()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return create(ctx, tx, 1)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				notes.Add(tx, "only")
				return nil
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return create(ctx, tx, 2)
			})).To(Succeed())
			Expect(*commits).To(HaveLen(3))
			Expect((*commits)[1].Seq).To(Equal(uint64(2)))
			Expect((*commits)[1].Changes).To(BeEmpty())
			Expect(notes.Of((*commits)[1])).To(Equal([]string{"only"}))
			Expect((*commits)[2].Seq).To(Equal(uint64(3)))
		},
	)

	Describe("Pending", func() {
		It(
			"Should return the values attached to a transaction so far",
			func(ctx SpecContext) {
				other := gorp.NewAttachment[string]()
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					Expect(notes.Pending(tx)).To(BeEmpty())
					notes.Add(tx, "first")
					Expect(notes.Pending(tx)).To(Equal([]string{"first"}))
					other.Add(tx, "other")
					notes.Add(tx, "second", "third")
					Expect(notes.Pending(tx)).
						To(Equal([]string{"first", "second", "third"}))
					Expect(other.Pending(tx)).To(Equal([]string{"other"}))
					return nil
				})).To(Succeed())
			},
		)

		It(
			"Should give work registered before a value is added every value",
			func(ctx SpecContext) {
				var pending []string
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					notes.Add(tx, "first")
					gorp.BeforeCommit(tx, func(context.Context) error {
						pending = notes.Pending(tx)
						return nil
					})
					notes.Add(tx, "second")
					return nil
				})).To(Succeed())
				Expect(pending).To(Equal([]string{"first", "second"}))
			},
		)

		It("Should panic when reading through a DB", func() {
			Expect(func() { notes.Pending(db) }).
				To(PanicWith("[gorp] - Attachment.Pending needs a transaction, not a DB"))
		})
	})

	It("Should panic when adding through a DB", func() {
		Expect(func() { notes.Add(db, "value") }).
			To(PanicWith("[gorp] - Attachment.Add needs a transaction, not a DB"))
	})
})
