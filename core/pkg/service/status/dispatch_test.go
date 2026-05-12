// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Dispatch", Ordered, func() {
	var (
		db  *gorp.DB
		svc *status.Service
	)
	BeforeAll(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.Open())
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB: db, Ontology: otg, Search: searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db, Ontology: otg, Group: g, Search: searchIdx,
		}))
		svc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db, Ontology: otg, Group: g, Label: labelSvc, Search: searchIdx,
		}))
		Expect(searchIdx.Initialize(ctx)).To(Succeed())
	})

	Describe("AllowedVariants", func() {
		It("Should contain exactly the six allowed variant strings", func() {
			Expect(status.AllowedVariants).To(HaveLen(6))
			Expect(status.AllowedVariants).To(ContainElements(
				string(xstatus.VariantSuccess),
				string(xstatus.VariantInfo),
				string(xstatus.VariantWarning),
				string(xstatus.VariantError),
				string(xstatus.VariantLoading),
				string(xstatus.VariantDisabled),
			))
		})
	})

	Describe("SetByKeyOrName", func() {
		Describe("Invalid variant", func() {
			It("Should return ErrInvalidVariant for an unknown variant", func(ctx SpecContext) {
				key, multi, err := svc.SetByKeyOrName(ctx, "dispatch_iv_a", "msg", "bogus")
				Expect(err).To(MatchError(status.ErrInvalidVariant))
				Expect(key).To(BeEmpty())
				Expect(multi).To(BeFalse())
			})

			It("Should reject upper-cased variants (case-sensitive)", func(ctx SpecContext) {
				_, _, err := svc.SetByKeyOrName(ctx, "dispatch_iv_b", "msg", "SUCCESS")
				Expect(err).To(MatchError(status.ErrInvalidVariant))
			})

			It("Should reject the empty variant", func(ctx SpecContext) {
				_, _, err := svc.SetByKeyOrName(ctx, "dispatch_iv_c", "msg", "")
				Expect(err).To(MatchError(status.ErrInvalidVariant))
			})

			It("Should not write anything to the store when the variant is invalid", func(ctx SpecContext) {
				name := "dispatch_iv_unwritten"
				_, _, _ = svc.SetByKeyOrName(ctx, name, "msg", "bogus")
				var rows []status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchNames[any](name)).Entries(&rows).Exec(ctx, nil)).To(Succeed())
				Expect(rows).To(BeEmpty())
			})
		})

		Describe("By-key path (UUID input)", func() {
			It("Should update an existing row by UUID", func(ctx SpecContext) {
				key := uuid.NewString()
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: key, Name: "by_key_orig", Variant: xstatus.VariantInfo,
					Message: "old", Time: telem.Now(),
				})).To(Succeed())

				gotKey, multi, err := svc.SetByKeyOrName(ctx, key, "new", string(xstatus.VariantWarning))
				Expect(err).ToNot(HaveOccurred())
				Expect(gotKey).To(Equal(key))
				Expect(multi).To(BeFalse())

				var s status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](key)).Entry(&s).Exec(ctx, nil)).To(Succeed())
				Expect(s.Variant).To(Equal(xstatus.VariantWarning))
				Expect(s.Message).To(Equal("new"))
				Expect(s.Time).ToNot(BeZero())
			})

			// By-key path returns the input UUID alongside the error (not "");
			// pinned so a future "normalize on error" refactor trips this spec.
			It("Should propagate query.ErrNotFound when the UUID is unknown", func(ctx SpecContext) {
				missing := uuid.NewString()
				gotKey, multi, err := svc.SetByKeyOrName(ctx, missing, "x", string(xstatus.VariantInfo))
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, query.ErrNotFound)).To(BeTrue())
				Expect(gotKey).To(Equal(missing))
				Expect(multi).To(BeFalse())
			})
		})

		Describe("By-name path", func() {
			It("Should create a fresh row when no match exists", func(ctx SpecContext) {
				name := "by_name_fresh_a"
				gotKey, multi, err := svc.SetByKeyOrName(ctx, name, "hello", string(xstatus.VariantInfo))
				Expect(err).ToNot(HaveOccurred())
				Expect(gotKey).ToNot(BeEmpty())
				_, parseErr := uuid.Parse(gotKey)
				Expect(parseErr).ToNot(HaveOccurred())
				Expect(multi).To(BeFalse())

				var s status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](gotKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
				Expect(s.Name).To(Equal(name))
				Expect(s.Variant).To(Equal(xstatus.VariantInfo))
				Expect(s.Message).To(Equal("hello"))
			})

			It("Should update in place when there is a single match", func(ctx SpecContext) {
				name := "by_name_single"
				existingKey := uuid.NewString()
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: existingKey, Name: name, Variant: xstatus.VariantSuccess,
					Message: "ok", Time: telem.Now(),
				})).To(Succeed())

				gotKey, multi, err := svc.SetByKeyOrName(ctx, name, "now bad", string(xstatus.VariantError))
				Expect(err).ToNot(HaveOccurred())
				Expect(gotKey).To(Equal(existingKey))
				Expect(multi).To(BeFalse())

				var s status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](existingKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
				Expect(s.Variant).To(Equal(xstatus.VariantError))
				Expect(s.Message).To(Equal("now bad"))
			})

			// matches come out of writer.retrieveByName in gorp insert order; the
			// first key returned drives the upsert.
			It("Should report multipleMatches and update only the first match", func(ctx SpecContext) {
				name := "by_name_multi"
				firstKey := uuid.NewString()
				secondKey := uuid.NewString()
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: firstKey, Name: name, Variant: xstatus.VariantInfo,
					Message: "first", Time: telem.Now(),
				})).To(Succeed())
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: secondKey, Name: name, Variant: xstatus.VariantInfo,
					Message: "second", Time: telem.Now(),
				})).To(Succeed())

				gotKey, multi, err := svc.SetByKeyOrName(ctx, name, "updated", string(xstatus.VariantWarning))
				Expect(err).ToNot(HaveOccurred())
				Expect(multi).To(BeTrue())
				Expect(gotKey).To(SatisfyAny(Equal(firstKey), Equal(secondKey)))

				// One row carries the new variant; the other still has the original.
				var rows []status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](firstKey, secondKey)).Entries(&rows).Exec(ctx, nil)).To(Succeed())
				Expect(rows).To(HaveLen(2))
				warning, info := 0, 0
				for _, r := range rows {
					switch r.Variant {
					case xstatus.VariantWarning:
						warning++
					case xstatus.VariantInfo:
						info++
					}
				}
				Expect(warning).To(Equal(1))
				Expect(info).To(Equal(1))
			})

			// Empty keyOrName routes to by-name path; writer.validate only
			// forbids empty Key, so name="" is accepted today.
			It("Should create a row with empty name when keyOrName is empty", func(ctx SpecContext) {
				gotKey, multi, err := svc.SetByKeyOrName(ctx, "", "x", string(xstatus.VariantInfo))
				Expect(err).ToNot(HaveOccurred())
				Expect(gotKey).ToNot(BeEmpty())
				Expect(multi).To(BeFalse())
				var s status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](gotKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
				Expect(s.Name).To(BeEmpty())
			})

			// writer.validate doesn't constrain Message; pin current behavior
			// so a future tighten-up trips this spec.
			It("Should accept an empty message", func(ctx SpecContext) {
				name := "by_name_empty_msg"
				gotKey, _, err := svc.SetByKeyOrName(ctx, name, "", string(xstatus.VariantInfo))
				Expect(err).ToNot(HaveOccurred())
				var s status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](gotKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
				Expect(s.Message).To(BeEmpty())
			})
		})
	})

	Describe("DeleteByKeyOrName", func() {
		Describe("By-key path", func() {
			It("Should delete a row by UUID and return count=1", func(ctx SpecContext) {
				key := uuid.NewString()
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: key, Name: "del_by_key", Variant: xstatus.VariantInfo,
					Message: "x", Time: telem.Now(),
				})).To(Succeed())

				count, err := svc.DeleteByKeyOrName(ctx, key)
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(1))

				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](key)).Entry(&status.Status[any]{}).Exec(ctx, nil)).
					To(MatchError(query.ErrNotFound))
			})

			It("Should return count=0 with no error when the UUID is unknown", func(ctx SpecContext) {
				count, err := svc.DeleteByKeyOrName(ctx, uuid.NewString())
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(0))
			})
		})

		Describe("By-name path", func() {
			It("Should delete a single match and return count=1", func(ctx SpecContext) {
				name := "del_by_name_single"
				key := uuid.NewString()
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: key, Name: name, Variant: xstatus.VariantInfo,
					Message: "x", Time: telem.Now(),
				})).To(Succeed())

				count, err := svc.DeleteByKeyOrName(ctx, name)
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(1))

				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](key)).Entry(&status.Status[any]{}).Exec(ctx, nil)).
					To(MatchError(query.ErrNotFound))
			})

			It("Should delete all matches on multi-match", func(ctx SpecContext) {
				name := "del_by_name_multi"
				k1, k2 := uuid.NewString(), uuid.NewString()
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: k1, Name: name, Variant: xstatus.VariantInfo, Message: "a", Time: telem.Now(),
				})).To(Succeed())
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: k2, Name: name, Variant: xstatus.VariantInfo, Message: "b", Time: telem.Now(),
				})).To(Succeed())

				count, err := svc.DeleteByKeyOrName(ctx, name)
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(2))

				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](k1, k2)).Entry(&status.Status[any]{}).Exec(ctx, nil)).
					To(MatchError(query.ErrNotFound))
			})

			It("Should return count=0 with no error when no row matches", func(ctx SpecContext) {
				count, err := svc.DeleteByKeyOrName(ctx, "del_by_name_missing")
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(0))
			})
		})
	})
})
