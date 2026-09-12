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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Delete", func() {
	var tx gorp.Tx
	BeforeEach(func() {
		tx = DeferClose(db.OpenTx())
	})

	Describe("ExecKeys", func() {
		It("Should return the keys it deleted", func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, entry]().
				Entries(&[]entry{{ID: 1, Data: "one"}, {ID: 2, Data: "two"}}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int32, entry]().
				Where(gorp.MatchKeys[int32, entry](1, 2)).
				ExecKeys(ctx, tx)).To(ConsistOf(int32(1), int32(2)))
		})

		It("Should omit keys that match no entry", func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 3, Data: "three"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int32, entry]().
				Where(gorp.MatchKeys[int32, entry](3, 404)).
				ExecKeys(ctx, tx)).To(ConsistOf(int32(3)))
		})

		It("Should return no keys when nothing matches", func(ctx SpecContext) {
			Expect(gorp.NewDelete[int32, entry]().
				Where(gorp.MatchKeys[int32, entry](909)).
				ExecKeys(ctx, tx)).To(BeEmpty())
		})

		It("Should return the guard's error and delete nothing", func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 4, Data: "guarded"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewDelete[int32, entry]().
				Where(gorp.MatchKeys[int32, entry](4)).
				Guard(func(_ gorp.Context, e entry) error {
					return validate.ErrValidation
				}).
				ExecKeys(ctx, tx)).Error().To(MatchError(validate.ErrValidation))
			Expect(gorp.NewRetrieve[int32, entry]().
				Where(gorp.MatchKeys[int32, entry](4)).
				Exists(ctx, tx)).To(BeTrue())
		})
	})

	Describe("WhereKeys", func() {
		It("Should delete an entry by key in the db", func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 1, Data: "Synnax"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(
				gorp.NewDelete[int32, entry]().Where(gorp.MatchKeys[int32, entry](1)).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(
				gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](1)).
					Exists(ctx, tx),
			).To(BeFalse())
		})
		It(
			"Should NOT return an error if the entry does not exist",
			func(ctx SpecContext) {
				Expect(
					gorp.NewDelete[int32, entry]().Where(gorp.MatchKeys[int32, entry](1)).
						Exec(ctx, tx),
				).To(Succeed())
			},
		)
	})

	Describe("Where", func() {
		It("Should delete an entry by predicate in the db", func(ctx SpecContext) {
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&entry{ID: 1, Data: "Synnax"}).
				Exec(ctx, tx)).To(Succeed())
			Expect(
				gorp.NewDelete[int32, entry]().Where(gorp.Match(func(_ gorp.Context, e *entry) (bool, error) {
					return e.Data == "Synnax", nil
				})).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(
				gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](1)).
					Exists(ctx, tx),
			).To(BeFalse())
		})

		It(
			"Should not return an error if the entry does not exist",
			func(ctx SpecContext) {
				Expect(
					gorp.NewDelete[int32, entry]().Where(gorp.Match(func(_ gorp.Context, e *entry) (bool, error) {
						return e.Data == "Synnax", nil
					})).
						Exec(ctx, tx),
				).To(Succeed())
			},
		)
	})

	Describe("Guard", func() {
		It(
			"Should prevent deletion if any of the guard functions fail",
			func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, entry]().
					Entry(&entry{ID: 1, Data: "Synnax"}).
					Exec(ctx, tx)).To(Succeed())
				Expect(gorp.NewDelete[int32, entry]().
					Where(gorp.MatchKeys[int32, entry](1)).
					Guard(func(_ gorp.Context, e entry) error {
						return validate.ErrValidation
					}).Exec(ctx, tx)).To(MatchError(validate.ErrValidation))
				Expect(
					gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](1)).
						Exists(ctx, tx),
				).To(BeTrue())
			},
		)

		It(
			"Should pass the correct transaction to the gorp context of the guard clause",
			func(ctx SpecContext) {
				Expect(gorp.NewCreate[int32, entry]().
					Entry(&entry{ID: 22, Data: "Synnax"}).
					Exec(ctx, tx)).To(Succeed())
				Expect(gorp.NewDelete[int32, entry]().
					Where(gorp.MatchKeys[int32, entry](22)).
					Guard(func(gCtx gorp.Context, _ entry) error {
						Expect(gCtx.Tx).To(BeIdenticalTo(tx))
						Expect(gCtx.Context).To(BeIdenticalTo(ctx))
						return validate.ErrValidation
					}).Exec(ctx, tx)).To(MatchError(validate.ErrValidation))
				Expect(
					gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](22)).
						Exists(ctx, tx),
				).To(BeTrue())
			},
		)
	})
})
