// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration_test

import (
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	var (
		svc *enumeration.Service
		tx  gorp.Tx
		w   enumeration.Writer
	)
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		svc = MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}))
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx)
	})
	retrieve := func(ctx SpecContext, key enumeration.Key) enumeration.Enumeration {
		GinkgoHelper()
		var e enumeration.Enumeration
		Expect(svc.NewRetrieve().
			Where(enumeration.MatchKeys(key)).
			Entry(&e).
			Exec(ctx, tx)).To(Succeed())
		return e
	}
	breaker := func() enumeration.Enumeration {
		return enumeration.Enumeration{
			Name: "Breaker",
			Entries: []enumeration.Entry{
				{Key: "open", Value: 0, Name: "Open"},
				{Key: "closed", Value: 1, Name: "Closed"},
				{Key: "tripped", Value: 2, Name: "Tripped", Abnormal: true},
			},
		}
	}

	Describe("Create", func() {
		It("Should create an enumeration with a new key", func(ctx SpecContext) {
			e := breaker()
			Expect(createOne(ctx, w, &e)).To(Succeed())
			Expect(e.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, e.Key)).To(Equal(e))
		})
		It("Should create an enumeration with no entries", func(ctx SpecContext) {
			e := enumeration.Enumeration{Name: "Empty"}
			Expect(createOne(ctx, w, &e)).To(Succeed())
			Expect(retrieve(ctx, e.Key).Entries).To(BeEmpty())
		})
		It("Should refuse an enumeration with no name", func(ctx SpecContext) {
			e := breaker()
			e.Name = ""
			Expect(createOne(ctx, w, &e)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("name: required")),
			))
		})
		It("Should refuse an entry with no key", func(ctx SpecContext) {
			e := breaker()
			e.Entries[1].Key = ""
			Expect(createOne(ctx, w, &e)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("entries.1.key: required")),
			))
		})
		It("Should refuse two entries with one key", func(ctx SpecContext) {
			e := breaker()
			e.Entries[2].Key = "open"
			Expect(createOne(ctx, w, &e)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					`entries.2.key: another entry has the key "open"`,
				)),
			))
		})
		It("Should refuse two entries with one value", func(ctx SpecContext) {
			e := breaker()
			e.Entries[2].Value = 1
			Expect(createOne(ctx, w, &e)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"entries.2.value: another entry has the value 1",
				)),
			))
		})
		DescribeTable(
			"Should refuse a value that is not a finite number",
			func(ctx SpecContext, value float64) {
				e := breaker()
				e.Entries[0].Value = value
				Expect(createOne(ctx, w, &e)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("entries.0.value: must be finite")),
				))
			},
			Entry("NaN", math.NaN()),
			Entry("infinity", math.Inf(1)),
			Entry("negative infinity", math.Inf(-1)),
		)
		It("Should refuse an enumeration whose key exists", func(ctx SpecContext) {
			e := breaker()
			Expect(createOne(ctx, w, &e)).To(Succeed())
			again := enumeration.Enumeration{Key: e.Key, Name: "Valve"}
			Expect(createOne(ctx, w, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("enumeration %s already exists", e.Key)),
			))
		})
	})

	Describe("CreateMany", func() {
		It("Should create each enumeration with a new key", func(ctx SpecContext) {
			enumerations := []enumeration.Enumeration{breaker(), {Name: "Valve"}}
			Expect(w.CreateMany(ctx, enumerations)).To(Succeed())
			for _, e := range enumerations {
				Expect(e.Key).ToNot(Equal(uuid.Nil()))
				Expect(retrieve(ctx, e.Key)).To(Equal(e))
			}
		})
		It("Should stop at the first enumeration that is not valid", func(
			ctx SpecContext,
		) {
			enumerations := []enumeration.Enumeration{
				breaker(),
				{},
				{Name: "Valve"},
			}
			Expect(w.CreateMany(ctx, enumerations)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("name: required")),
			))
			Expect(enumerations[2].Key).To(Equal(uuid.Nil()))
		})
	})

	Describe("Apply", func() {
		var e enumeration.Enumeration
		BeforeEach(func(ctx SpecContext) {
			e = breaker()
			Expect(createOne(ctx, w, &e)).To(Succeed())
		})
		It("Should store the result of the actions", func(ctx SpecContext) {
			Expect(w.Apply(
				ctx,
				e.Key,
				enumeration.NewSetAction(enumeration.SetPayload{
					Path:  []string{"name"},
					Value: "Main breaker",
				}),
				enumeration.NewSetAction(enumeration.SetPayload{
					Path:  []string{"entries", "tripped", "value"},
					Value: 3,
				}),
			)).To(Succeed())
			expected := breaker()
			expected.Key = e.Key
			expected.Name = "Main breaker"
			expected.Entries[2].Value = 3
			Expect(retrieve(ctx, e.Key)).To(Equal(expected))
		})
		It("Should refuse a result with two entries of one value", func(
			ctx SpecContext,
		) {
			Expect(w.Apply(
				ctx,
				e.Key,
				enumeration.NewSetAction(enumeration.SetPayload{
					Path:  []string{"entries", "tripped", "value"},
					Value: 0,
				}),
			)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"entries.2.value: another entry has the value 0",
				)),
			))
			Expect(retrieve(ctx, e.Key)).To(Equal(e))
		})
		It("Should refuse an inserted entry with the value of another", func(
			ctx SpecContext,
		) {
			Expect(w.Apply(
				ctx,
				e.Key,
				enumeration.NewInsertAction(enumeration.InsertPayload{
					Path: []string{"entries"},
					Value: map[string]any{
						"key":      "fault",
						"value":    2,
						"name":     "Fault",
						"abnormal": false,
					},
				}),
			)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("another entry has the value 2")),
			))
		})
		It("Should refuse a create action", func(ctx SpecContext) {
			Expect(w.Apply(
				ctx,
				e.Key,
				enumeration.NewCreateAction(enumeration.CreatePayload{Enumeration: e}),
			)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"only the Core dispatches an enumeration create",
				)),
			))
		})
		It("Should return an error for a key with no enumeration", func(
			ctx SpecContext,
		) {
			Expect(w.Apply(
				ctx,
				uuid.New(),
				enumeration.NewSetAction(
					enumeration.SetPayload{Path: []string{"name"}, Value: "A"},
				),
			)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Replace", func() {
		It("Should delete the first enumeration and keep the other", func(
			ctx SpecContext,
		) {
			from, to := breaker(), enumeration.Enumeration{Name: "Valve"}
			Expect(createOne(ctx, w, &from)).To(Succeed())
			Expect(createOne(ctx, w, &to)).To(Succeed())
			Expect(w.Replace(ctx, from.Key, to.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(enumeration.MatchKeys(from.Key)).
				Exists(ctx, tx)).To(BeFalse())
			Expect(retrieve(ctx, to.Key).Name).To(Equal("Valve"))
		})
		It("Should refuse a built-in enumeration", func(ctx SpecContext) {
			to := breaker()
			Expect(createOne(ctx, w, &to)).To(Succeed())
			Expect(w.Replace(ctx, enumeration.Builtins()[0].Key, to.Key)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("cannot delete built-in enumeration")),
			))
		})
		It("Should refuse an enumeration replaced by itself", func(ctx SpecContext) {
			e := breaker()
			Expect(createOne(ctx, w, &e)).To(Succeed())
			Expect(w.Replace(ctx, e.Key, e.Key)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("to itself")),
			))
		})
		It("Should return an error for a key with no enumeration", func(
			ctx SpecContext,
		) {
			e := breaker()
			Expect(createOne(ctx, w, &e)).To(Succeed())
			Expect(w.Replace(ctx, e.Key, uuid.New())).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should delete the enumerations", func(ctx SpecContext) {
			a, b := breaker(), enumeration.Enumeration{Name: "Valve"}
			Expect(createOne(ctx, w, &a)).To(Succeed())
			Expect(createOne(ctx, w, &b)).To(Succeed())
			Expect(w.Delete(ctx, a.Key, b.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(enumeration.MatchKeys(a.Key, b.Key)).
				Exists(ctx, tx)).To(BeFalse())
		})
		It("Should not return an error for a key with no enumeration", func(
			ctx SpecContext,
		) {
			Expect(w.Delete(ctx, uuid.New())).To(Succeed())
		})
	})
})
