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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
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

	Describe("SetByKeyOrName", func() {
		Describe("Input validation", func() {
			It("Should return a validation error for an unknown variant", func(ctx SpecContext) {
				Expect(svc.SetByKeyOrName(ctx, "dispatch_iv_a", "msg", "bogus")).
					Error().To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
			})

			It("Should reject upper-cased variants (case-sensitive)", func(ctx SpecContext) {
				Expect(svc.SetByKeyOrName(ctx, "dispatch_iv_b", "msg", "SUCCESS")).
					Error().To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
			})

			It("Should reject the empty variant", func(ctx SpecContext) {
				Expect(svc.SetByKeyOrName(ctx, "dispatch_iv_c", "msg", "")).
					Error().To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
			})

			It("Should return a validation error for empty input", func(ctx SpecContext) {
				Expect(svc.SetByKeyOrName(ctx, "", "msg", string(xstatus.VariantInfo))).
					Error().To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("key_or_name is required"))))
			})

			It("Should not write anything to the store when the variant is invalid", func(ctx SpecContext) {
				name := "dispatch_iv_unwritten"
				Expect(svc.SetByKeyOrName(ctx, name, "msg", "bogus")).
					Error().To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](name)).
					Entry(&status.Status[any]{}).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			})
		})

		Describe("By-key path", func() {
			It("Should update an existing row whose Key matches the input", func(ctx SpecContext) {
				key := uuid.NewString()
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: key, Name: "by_key_orig",
					Status: xstatus.Status[any]{
						Variant: xstatus.VariantInfo, Message: "old", Time: telem.Now(),
					},
				})).To(Succeed())

				gotKey, multi := MustSucceed2(svc.SetByKeyOrName(ctx, key, "new", string(xstatus.VariantWarning)))
				Expect(gotKey).To(Equal(key))
				Expect(multi).To(BeFalse())

				var s status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](key)).Entry(&s).Exec(ctx, nil)).To(Succeed())
				Expect(s.Variant).To(Equal(xstatus.VariantWarning))
				Expect(s.Message).To(Equal("new"))
				Expect(s.Time).ToNot(BeZero())
			})

			It("Should match arbitrary (non-UUID) string keys", func(ctx SpecContext) {
				key := "by_key_plain_string"
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: key, Name: "by_key_plain_orig",
					Status: xstatus.Status[any]{
						Variant: xstatus.VariantInfo, Message: "old", Time: telem.Now(),
					},
				})).To(Succeed())

				gotKey, multi := MustSucceed2(svc.SetByKeyOrName(ctx, key, "new", string(xstatus.VariantSuccess)))
				Expect(gotKey).To(Equal(key))
				Expect(multi).To(BeFalse())
			})

			It("Should prefer the by-key match over a by-name match for the same input", func(ctx SpecContext) {
				shared := "shared_token"
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: shared, Name: "by_key_winner",
					Status: xstatus.Status[any]{
						Variant: xstatus.VariantInfo, Message: "key", Time: telem.Now(),
					},
				})).To(Succeed())
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: uuid.NewString(), Name: shared,
					Status: xstatus.Status[any]{
						Variant: xstatus.VariantInfo, Message: "name", Time: telem.Now(),
					},
				})).To(Succeed())

				gotKey, multi := MustSucceed2(svc.SetByKeyOrName(ctx, shared, "updated", string(xstatus.VariantWarning)))
				Expect(gotKey).To(Equal(shared))
				Expect(multi).To(BeFalse())
			})
		})

		Describe("By-name path", func() {
			It("Should update in place when there is a single name match", func(ctx SpecContext) {
				name := "by_name_single"
				existingKey := uuid.NewString()
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: existingKey, Name: name,
					Status: xstatus.Status[any]{
						Variant: xstatus.VariantSuccess, Message: "ok", Time: telem.Now(),
					},
				})).To(Succeed())

				gotKey, multi := MustSucceed2(svc.SetByKeyOrName(ctx, name, "now bad", string(xstatus.VariantError)))
				Expect(gotKey).To(Equal(existingKey))
				Expect(multi).To(BeFalse())

				var s status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](existingKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
				Expect(s.Variant).To(Equal(xstatus.VariantError))
				Expect(s.Message).To(Equal("now bad"))
			})

			It("Should report multipleMatches and update only the first match", func(ctx SpecContext) {
				name := "by_name_multi"
				firstKey := uuid.NewString()
				secondKey := uuid.NewString()
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: firstKey, Name: name,
					Status: xstatus.Status[any]{
						Variant: xstatus.VariantInfo, Message: "first", Time: telem.Now(),
					},
				})).To(Succeed())
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: secondKey, Name: name,
					Status: xstatus.Status[any]{
						Variant: xstatus.VariantInfo, Message: "second", Time: telem.Now(),
					},
				})).To(Succeed())

				gotKey, multi := MustSucceed2(svc.SetByKeyOrName(ctx, name, "updated", string(xstatus.VariantWarning)))
				Expect(multi).To(BeTrue())
				Expect(gotKey).To(SatisfyAny(Equal(firstKey), Equal(secondKey)))

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

			It("Should accept an empty message on the by-name path", func(ctx SpecContext) {
				name := "by_name_empty_msg"
				existingKey := uuid.NewString()
				Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
					Key: existingKey, Name: name,
					Status: xstatus.Status[any]{
						Variant: xstatus.VariantInfo, Message: "old", Time: telem.Now(),
					},
				})).To(Succeed())
				gotKey, _ := MustSucceed2(svc.SetByKeyOrName(ctx, name, "", string(xstatus.VariantInfo)))
				Expect(gotKey).To(Equal(existingKey))
				var s status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](gotKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
				Expect(s.Message).To(BeEmpty())
			})
		})

		Describe("Create path", func() {
			It("Should create a UUID-keyed row named after the input when nothing matches", func(ctx SpecContext) {
				input := "fresh_row_a"
				gotKey, multi := MustSucceed2(svc.SetByKeyOrName(ctx, input, "hello", string(xstatus.VariantInfo)))
				MustSucceed(uuid.Parse(gotKey))
				Expect(gotKey).ToNot(Equal(input))
				Expect(multi).To(BeFalse())

				var s status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](gotKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
				Expect(s.Key).To(Equal(gotKey))
				Expect(s.Name).To(Equal(input))
				Expect(s.Variant).To(Equal(xstatus.VariantInfo))
				Expect(s.Message).To(Equal("hello"))
			})

			It("Should match a created row by name on subsequent SetByKeyOrName calls", func(ctx SpecContext) {
				input := "fresh_row_then_update"
				firstKey, _ := MustSucceed2(svc.SetByKeyOrName(ctx, input, "first", string(xstatus.VariantInfo)))
				gotKey, multi := MustSucceed2(svc.SetByKeyOrName(ctx, input, "second", string(xstatus.VariantWarning)))
				Expect(gotKey).To(Equal(firstKey))
				Expect(multi).To(BeFalse())

				var s status.Status[any]
				Expect(svc.NewRetrieve().Where(status.MatchKeys[any](gotKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
				Expect(s.Message).To(Equal("second"))
				Expect(s.Variant).To(Equal(xstatus.VariantWarning))
			})
		})
	})

})
