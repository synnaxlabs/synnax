// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"context"
	"iter"
	"slices"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

// newAuthor creates a fresh user and returns its key. Committed-write specs
// (OnChange, OpenNexter) use a unique author per spec so they never collide with
// other specs that query projects by author.
func newAuthor(ctx SpecContext) user.Key {
	return MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
		Username: uuid.NewString(),
	})).Key
}

var _ = Describe("Ontology", func() {
	Describe("OntologyID", func() {
		It("Should build an ontology ID with the project resource type and the key", func() {
			key := uuid.New()
			id := project.OntologyID(key)
			Expect(id.Type).To(Equal(ontology.ResourceTypeProject))
			Expect(id.Key).To(Equal(key.String()))
		})
	})
	Describe("OntologyIDs", func() {
		It("Should map a slice of keys to ontology IDs", func() {
			a, b := uuid.New(), uuid.New()
			Expect(project.OntologyIDs([]project.Key{a, b})).
				To(ConsistOf(project.OntologyID(a), project.OntologyID(b)))
		})
	})
	Describe("OntologyIDsFromProjects", func() {
		It("Should map a slice of projects to ontology IDs", func() {
			a := project.Project{Key: uuid.New(), Name: "a"}
			b := project.Project{Key: uuid.New(), Name: "b"}
			Expect(project.OntologyIDsFromProjects([]project.Project{a, b})).
				To(ConsistOf(a.OntologyID(), b.OntologyID()))
		})
	})
	Describe("Project.OntologyID", func() {
		It("Should return an ontology ID populated from the project's key", func() {
			p := project.Project{Key: uuid.New(), Name: "x"}
			Expect(p.OntologyID()).To(Equal(project.OntologyID(p.Key)))
		})
	})
	Describe("KeysFromOntologyIDs", func() {
		It("Should round-trip keys through ontology IDs", func() {
			a, b := uuid.New(), uuid.New()
			Expect(project.KeysFromOntologyIDs([]ontology.ID{
				project.OntologyID(a), project.OntologyID(b),
			})).To(Equal([]project.Key{a, b}))
		})
		It("Should return an error when an ID key is not a valid UUID", func() {
			Expect(project.KeysFromOntologyIDs([]ontology.ID{{
				Type: ontology.ResourceTypeProject, Key: "not-a-uuid",
			}})).Error().To(MatchError(ContainSubstring("invalid UUID")))
		})
	})
	Describe("Schema", func() {
		It("Should return an object schema with key and name fields", func() {
			shape := svc.Schema().Shape()
			Expect(shape.DataType()).To(Equal(zyn.ObjectT))
			Expect(shape.Fields()).To(SatisfyAll(HaveKey("key"), HaveKey("name")))
		})
	})
	Describe("Type", func() {
		It("Should return the project resource type", func() {
			Expect(svc.Type()).To(Equal(ontology.ResourceTypeProject))
		})
	})
	Describe("RetrieveResource", func() {
		It("Should retrieve a project as an ontology resource", func(ctx SpecContext) {
			p := project.Project{Key: uuid.New(), Name: "resource", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			resource := MustSucceed(svc.RetrieveResource(ctx, p.Key.String(), tx))
			Expect(resource.ID).To(Equal(project.OntologyID(p.Key)))
			Expect(resource.Name).To(Equal("resource"))
		})
		It("Should return an error when the key is not a valid UUID", func(ctx SpecContext) {
			Expect(svc.RetrieveResource(ctx, "not-a-uuid", nil)).Error().
				To(MatchError(ContainSubstring("invalid UUID")))
		})
		It("Should return query.ErrNotFound when no project has the given key", func(ctx SpecContext) {
			Expect(svc.RetrieveResource(ctx, uuid.NewString(), nil)).Error().
				To(MatchError(query.ErrNotFound))
		})
	})
	Describe("OnChange", func() {
		It("Should publish set and delete changes translated into ontology changes", func(ctx SpecContext) {
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

			p := project.Project{Key: uuid.New(), Name: "observed", Author: newAuthor(ctx)}
			Expect(svc.NewWriter(nil).Create(ctx, &p)).To(Succeed())
			expectedID := project.OntologyID(p.Key).String()

			Eventually(func(g Gomega) {
				mu.Lock()
				defer mu.Unlock()
				idx := slices.IndexFunc(changes, func(c ontology.Change) bool {
					return c.Key == expectedID && c.Variant == xchange.VariantSet
				})
				g.Expect(idx).ToNot(Equal(-1))
				g.Expect(changes[idx].Value.Name).To(Equal("observed"))
			}).Should(Succeed())

			Expect(svc.NewWriter(nil).Delete(ctx, p.Key)).To(Succeed())

			Eventually(func(g Gomega) {
				mu.Lock()
				defer mu.Unlock()
				g.Expect(changes).To(ContainElement(SatisfyAll(
					HaveField("Key", expectedID),
					HaveField("Variant", xchange.VariantDelete),
				)))
			}).Should(Succeed())
		})
	})
	Describe("OpenNexter", func() {
		It("Should iterate over all projects currently stored", func(ctx SpecContext) {
			a := project.Project{Key: uuid.New(), Name: "a", Author: newAuthor(ctx)}
			b := project.Project{Key: uuid.New(), Name: "b", Author: newAuthor(ctx)}
			Expect(svc.NewWriter(nil).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(nil).Create(ctx, &b)).To(Succeed())

			seq, closer := MustSucceed2(svc.OpenNexter(ctx))
			DeferClose(closer)

			seen := make(map[string]struct{})
			for resource := range seq {
				seen[resource.ID.String()] = struct{}{}
			}
			Expect(seen).To(SatisfyAll(
				HaveKey(a.OntologyID().String()),
				HaveKey(b.OntologyID().String()),
			))
		})
	})
})
