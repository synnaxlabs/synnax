// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role_test

import (
	"context"
	"iter"
	"slices"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = db.OpenTx() })
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("OntologyID", func() {
		It("Should return the correct ontology ID", func() {
			key := uuid.New()
			Expect(role.OntologyID(key)).To(Equal(ontology.ID{
				Type: role.OntologyType,
				Key:  key.String(),
			}))
		})
	})

	Describe("Type", func() {
		It("Should return the correct ontology type", func() {
			Expect(svc.Type()).To(Equal(role.OntologyType))
		})
	})

	Describe("Schema", func() {
		It("Should validate properly shaped roles", func() {
			schema := svc.Schema()
			key := uuid.New()
			data := map[string]any{"key": key, "name": "test-role"}
			var r role.Role
			Expect(schema.Parse(data, &r)).To(Succeed())
			Expect(r.Key).To(Equal(key))
			Expect(r.Name).To(Equal("test-role"))
		})
		It("Should fail to validate roles with invalid keys", func() {
			schema := svc.Schema()
			data := map[string]any{"key": "invalid-key", "name": "test-role"}
			Expect(schema.Parse(data, &role.Role{})).
				To(MatchError(ContainSubstring("invalid UUID format")))
		})
		It("Should fail to validate roles with invalid names", func() {
			schema := svc.Schema()
			data := map[string]any{"key": uuid.New()}
			Expect(schema.Parse(data, &role.Role{})).
				To(MatchError(ContainSubstring("name: required: validation error")))
		})
	})

	Describe("RetrieveResource", func() {
		It("Should retrieve a role as an ontology resource", func() {
			w := svc.NewWriter(tx)
			r := &role.Role{Name: "resource-test", Description: "Resource test role"}
			Expect(w.Create(ctx, r)).To(Succeed())
			res := MustSucceed(svc.RetrieveResource(ctx, r.Key.String(), tx))
			Expect(res.ID.Key).To(Equal(r.Key.String()))
			Expect(res.Name).To(Equal(r.Name))
		})

		It("Should return error for invalid UUID", func() {
			Expect(svc.RetrieveResource(ctx, "invalid-uuid", tx)).Error().
				To(HaveOccurred())
		})

		It("Should return error for non-existent role", func() {
			Expect(svc.RetrieveResource(ctx, uuid.New().String(), tx)).Error().
				To(MatchError(query.NotFound))
		})
	})

	Describe("OnChange", func() {
		It("Should listen for changes to roles", func() {
			changeChan := make(chan ontology.Change, 1)
			disconnect := svc.OnChange(func(
				_ context.Context,
				changes iter.Seq[ontology.Change],
			) {
				for change := range changes {
					changeChan <- change
				}
			})
			defer disconnect()
			w := svc.NewWriter(tx)
			r := &role.Role{Name: "test-role", Description: "Test role"}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			cg := <-changeChan
			Expect(cg.Variant).To(Equal(change.Set))
			Expect(cg.Key).To(Equal(r.OntologyID()))
			Expect(cg.Value.Name).To(Equal(r.Name))
			tx := db.OpenTx()
			Expect(svc.NewWriter(tx).Delete(ctx, r.Key)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
			cg = <-changeChan
			Expect(cg.Variant).To(Equal(change.Delete))
			Expect(cg.Key).To(Equal(r.OntologyID()))
		})
	})

	Describe("OpenNexter", func() {
		It("Should iterate over all roles", func() {
			keys := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
			w := svc.NewWriter(tx)
			for i := range 3 {
				r := &role.Role{
					Key:         keys[i],
					Name:        "nexter-test",
					Description: "Nexter test role",
				}
				Expect(w.Create(ctx, r)).To(Succeed())
			}
			Expect(tx.Commit(ctx)).To(Succeed())

			nexter, closer := MustSucceed2(svc.OpenNexter(ctx))
			Expect(slices.Collect(nexter)).To(HaveLen(3))
			Expect(closer.Close()).To(Succeed())
			tx := db.OpenTx()
			w = svc.NewWriter(tx)
			for _, key := range keys {
				Expect(w.Delete(ctx, key)).To(Succeed())
			}
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
		})
	})
})
