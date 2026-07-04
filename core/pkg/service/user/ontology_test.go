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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology", func() {
	Describe("OntologyID", func() {
		It("Should build an ontology ID with the user resource type and the key", func() {
			key := uuid.New()
			id := user.OntologyID(key)
			Expect(id.Type).To(Equal(ontology.ResourceTypeUser))
			Expect(id.Key).To(Equal(key.String()))
		})
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
			u := user.User{Key: uuid.New(), Username: "alice"}
			Expect(u.OntologyID()).To(Equal(user.OntologyID(u.Key)))
		})
	})
	Describe("OntologyIDsFromUsers", func() {
		It("Should map a slice of users to a slice of ontology IDs", func() {
			a := user.User{Key: uuid.New(), Username: "a"}
			b := user.User{Key: uuid.New(), Username: "b"}
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
			})).Error().To(HaveOccurred())
		})
	})
	Describe("Type", func() {
		It("Should return the user resource type", func() {
			Expect(svc.Type()).To(Equal(ontology.ResourceTypeUser))
		})
	})
	Describe("SearchableFields", func() {
		It("Should return username, first_name, and last_name", func() {
			Expect(svc.SearchableFields()).To(ConsistOf("username", "first_name", "last_name"))
		})
		It("Should index first_name for fuzzy search via the suite's search index", func(ctx SpecContext) {
			created := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{
				Username:  uuid.NewString(),
				FirstName: "Persephone",
				LastName:  "Quintarelli",
			}))
			Eventually(func(g Gomega) {
				ids, err := searchIdx.Search(ctx, search.Request{
					Type: ontology.ResourceTypeUser,
					Term: "Persephone",
				})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(ids).To(ContainElement(created.OntologyID()))
			}).Should(Succeed())
		})
		It("Should index last_name for fuzzy search via the suite's search index", func(ctx SpecContext) {
			created := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{
				Username:  uuid.NewString(),
				FirstName: "Marigold",
				LastName:  "Ravenscroft",
			}))
			Eventually(func(g Gomega) {
				ids, err := searchIdx.Search(ctx, search.Request{
					Type: ontology.ResourceTypeUser,
					Term: "Ravenscroft",
				})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(ids).To(ContainElement(created.OntologyID()))
			}).Should(Succeed())
		})
	})
	Describe("RetrieveResource", func() {
		It("Should retrieve a user's schema entity by its key", func(ctx SpecContext) {
			key := uuid.New()
			created := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{
				Username: uuid.NewString(),
				Key:      key,
			}))
			Expect(svc.RetrieveResource(ctx, key.String(), nil)).To(
				Equal(ontology.Resource{
					ID:   created.OntologyID(),
					Name: created.Username,
					Data: map[string]any{
						"key":        key.String(),
						"username":   created.Username,
						"first_name": "",
						"last_name":  "",
						"root_user":  false,
					},
				}))
		})
		It("Should return an error when the key is not a valid UUID", func(ctx SpecContext) {
			Expect(svc.RetrieveResource(ctx, "not-a-uuid", nil)).Error().
				To(MatchError(ContainSubstring("invalid UUID")))
		})
		It("Should return query.ErrNotFound when no user has the given key", func(ctx SpecContext) {
			Expect(svc.RetrieveResource(ctx, uuid.NewString(), nil)).Error().
				To(MatchError(query.ErrNotFound))
		})
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
				disconnect := svc.OnChange(func(_ context.Context, it iter.Seq[ontology.Change]) {
					mu.Lock()
					defer mu.Unlock()
					changes = append(changes, slices.Collect(it)...)
				})
				DeferCleanup(disconnect)

				created := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{
					Username:  uuid.NewString(),
					FirstName: "Octavian",
				}))
				expectedID := created.OntologyID().String()

				Eventually(func(g Gomega) {
					mu.Lock()
					defer mu.Unlock()
					setIdx := slices.IndexFunc(changes, func(c ontology.Change) bool {
						return c.Key == expectedID && c.Variant == change.VariantSet
					})
					g.Expect(setIdx).ToNot(Equal(-1))
					g.Expect(changes[setIdx].Value.ID).To(Equal(created.OntologyID()))
					g.Expect(changes[setIdx].Value.Name).To(Equal(created.Username))
				}).Should(Succeed())

				Expect(svc.NewWriter(nil).Delete(ctx, created.Key)).To(Succeed())

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
	Describe("OpenNexter", func() {
		It("Should iterate over all users currently stored in the service", func(ctx SpecContext) {
			a := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{
				Username: uuid.NewString(),
			}))
			b := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{
				Username: uuid.NewString(),
			}))

			seq, closer := MustSucceed2(svc.OpenNexter(ctx))
			DeferClose(closer)

			seen := make(map[string]string)
			for resource := range seq {
				seen[resource.ID.String()] = resource.Name
			}
			Expect(seen).To(HaveKeyWithValue(a.OntologyID().String(), a.Username))
			Expect(seen).To(HaveKeyWithValue(b.OntologyID().String(), b.Username))
		})
	})
})
