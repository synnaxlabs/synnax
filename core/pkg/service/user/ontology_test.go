// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user_test

import (
	"context"
	"iter"
	"slices"
	"sync"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology", func() {
	Describe("OntologyID", func() {
		It(
			"Should build an ontology ID with the user resource type and the key",
			func() {
				key := uuid.New()
				id := user.OntologyID(key)
				Expect(id.Type).To(Equal(ontology.ResourceTypeUser))
				Expect(id.Key).To(Equal(key.String()))
			},
		)
	})
	Describe("OntologyIDsFromKeys", func() {
		It("Should map a slice of keys to a slice of ontology IDs", func() {
			a, b := uuid.New(), uuid.New()
			ids := user.OntologyIDsFromKeys([]user.Key{a, b})
			Expect(ids).To(HaveLen(2))
			Expect(ids[0]).To(Equal(user.OntologyID(a)))
			Expect(ids[1]).To(Equal(user.OntologyID(b)))
		})
		It("Should return an empty slice when given no keys", func() {
			Expect(user.OntologyIDsFromKeys(nil)).To(BeEmpty())
		})
	})
	Describe("User.OntologyID", func() {
		It("Should return an ontology ID populated from the user's key", func() {
			u := user.User{Key: uuid.New(), Name: "alice"}
			Expect(u.OntologyID()).To(Equal(u.OntologyID()))
		})
	})
	Describe("OntologyIDsFromUsers", func() {
		It("Should map a slice of users to a slice of ontology IDs", func() {
			a := user.User{Key: uuid.New(), Name: "a"}
			b := user.User{Key: uuid.New(), Name: "b"}
			ids := user.OntologyIDsFromUsers([]user.User{a, b})
			Expect(ids).To(ConsistOf(a.OntologyID(), b.OntologyID()))
		})
		It("Should return an empty slice when given no users", func() {
			Expect(user.OntologyIDsFromUsers(nil)).To(BeEmpty())
		})
	})
	Describe("KeyFromOntologyID", func() {
		It("Should round-trip a key through OntologyID", func() {
			key := uuid.New()
			Expect(user.KeyFromOntologyID(user.OntologyID(key))).To(Equal(key))
		})
		It("Should return an error when the ID key is not a valid UUID", func() {
			Expect(user.KeyFromOntologyID(ontology.ID{
				Type: ontology.ResourceTypeUser,
				Key:  "not-a-uuid",
			})).Error().To(MatchError("invalid uuid"))
		})
	})
	Describe("Type", func() {
		It("Should return the user resource type", func() {
			Expect(svc.Type()).To(Equal(ontology.ResourceTypeUser))
		})
	})
	Describe("Search", func() {
		DescribeTable("Should search users by each word of their name",
			func(ctx SpecContext, term string) {
				created := MustSucceed(writer.Create(ctx, user.User{
					Name: "Persephone Quintarelli",
				}))
				DeferCleanup(func(ctx SpecContext) {
					Expect(writer.Delete(ctx, created.Key)).To(Succeed())
				})
				Expect(MustSucceed(svc.Search(ctx, db, term, 10)).Keys).
					To(ContainElement(created.OntologyID()))
			},
			Entry("first word", "Persephone"),
			Entry("last word", "Quintarelli"),
		)
	})
	Describe("RetrieveResource", func() {
		It("Should retrieve a user's schema entity by its key", func(ctx SpecContext) {
			key := uuid.New()
			created := MustSucceed(writer.Create(ctx, user.User{
				Name: "Patrick Star",
				Key:  key,
			}))
			Expect(svc.RetrieveResource(ctx, key.String(), nil)).To(
				Equal(ontology.Resource{
					ID:   created.OntologyID(),
					Name: "Patrick Star",
					Data: map[string]any{
						"key":       key.String(),
						"name":      "Patrick Star",
						"root_user": false,
					},
				}),
			)
		})
		It(
			"Should return an error when the key is not a valid UUID",
			func(ctx SpecContext) {
				Expect(svc.RetrieveResource(ctx, "not-a-uuid", nil)).Error().
					To(MatchError(ContainSubstring("invalid uuid")))
			},
		)
		It(
			"Should return query.ErrNotFound when no user has the given key",
			func(ctx SpecContext) {
				Expect(svc.RetrieveResource(ctx, uuid.New().String(), nil)).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
	})
	Describe("OnChange", func() {
		// OnChange publishes the gorp-table change stream through translateChange so
		// each ontology.Change carries the ontology ID built from the user key and a
		// Resource parseable back to the User. We assert on the produced changes for
		// both Set and Delete variants — covering the translateChange function
		// end-to-end.
		It("Should publish set and delete changes translated into ontology changes",
			func(ctx SpecContext) {
				var (
					mu      sync.Mutex
					changes []ontology.Change
				)
				disconnect := svc.OnChange(
					func(_ context.Context, it iter.Seq[ontology.Change]) {
						mu.Lock()
						defer mu.Unlock()
						changes = append(changes, slices.Collect(it)...)
					},
				)
				DeferCleanup(disconnect)

				created := MustSucceed(writer.Create(ctx, user.User{Name: "Octavian"}))
				expectedID := created.OntologyID().String()

				Eventually(func(g Gomega) {
					mu.Lock()
					defer mu.Unlock()
					setIdx := slices.IndexFunc(changes, func(c ontology.Change) bool {
						return c.Key == expectedID && c.Variant == change.VariantSet
					})
					g.Expect(setIdx).ToNot(Equal(-1))
					g.Expect(changes[setIdx].Value.ID).To(Equal(created.OntologyID()))
					g.Expect(changes[setIdx].Value.Name).To(Equal("Octavian"))
				}).Should(Succeed())

				Expect(writer.Delete(ctx, created.Key)).To(Succeed())

				Eventually(func(g Gomega) {
					mu.Lock()
					defer mu.Unlock()
					g.Expect(changes).To(ContainElement(SatisfyAll(
						HaveField("Key", expectedID),
						HaveField("Variant", change.VariantDelete),
					)))
				}).Should(Succeed())
			})
	})
})
