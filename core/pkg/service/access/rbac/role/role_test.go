// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Role", Ordered, func() {
	var (
		db    *gorp.DB
		svc   *role.Service
		otg   *ontology.Ontology
		tx    gorp.Tx
		roles []role.Role
	)

	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		svc = MustSucceed(role.OpenService(ctx, role.Config{
			DB:       db,
			Ontology: otg,
		}))
		tx = db.OpenTx()
	})

	AfterAll(func() {
		Expect(tx.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})

	Describe("Writer", func() {
		Describe("Create", func() {
			It("Should create a new role with an explicit key", func() {
				key := uuid.New()
				r := &role.Role{
					Key:         key,
					Name:        "Engineer",
					Description: "Engineering team role",
					Policies:    []uuid.UUID{},
					Internal:    false,
				}
				w := svc.NewWriter(tx)
				Expect(w.Create(ctx, r)).To(Succeed())
				Expect(r.Key).To(Equal(key))
				Expect(r.Name).To(Equal("Engineer"))
				Expect(r.Description).To(Equal("Engineering team role"))
				Expect(r.Internal).To(BeFalse())
				roles = append(roles, *r)
			})

			It("Should create a new role and generate a key when not provided", func() {
				r := &role.Role{
					Name:        "Viewer",
					Description: "Read-only access",
					Policies:    []uuid.UUID{},
					Internal:    false,
				}
				w := svc.NewWriter(tx)
				Expect(w.Create(ctx, r)).To(Succeed())
				Expect(r.Key).ToNot(Equal(uuid.Nil))
				Expect(r.Name).To(Equal("Viewer"))
				roles = append(roles, *r)
			})

			It("Should create a builtin role", func() {
				r := &role.Role{
					Name:        "System",
					Description: "System administrator",
					Policies:    []uuid.UUID{uuid.New()},
					Internal:    true,
				}
				w := svc.NewWriter(tx)
				Expect(w.Create(ctx, r)).To(Succeed())
				Expect(r.Internal).To(BeTrue())
				Expect(r.Policies).To(HaveLen(1))
				roles = append(roles, *r)
			})
		})

		Describe("Delete", func() {
			It("Should delete a non-builtin role", func() {
				r := &role.Role{
					Name:        "Temporary",
					Description: "Temporary role for testing",
					Internal:    false,
				}
				w := svc.NewWriter(tx)
				Expect(w.Create(ctx, r)).To(Succeed())

				// Delete the role
				Expect(w.Delete(ctx, r.Key)).To(Succeed())

				// Verify deletion
				var retrieved role.Role
				err := svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieved).Exec(ctx, tx)
				Expect(err).To(MatchError(query.NotFound))
			})

			It("Should fail to delete a builtin role", func() {
				r := &role.Role{
					Name:        "BuiltinAdmin",
					Description: "Built-in administrator",
					Internal:    true,
				}
				w := svc.NewWriter(tx)
				Expect(w.Create(ctx, r)).To(Succeed())

				// Attempt to delete - should fail
				err := w.Delete(ctx, r.Key)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot delete builtin role"))

				// Verify role still exists
				var retrieved role.Role
				Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
				Expect(retrieved.Key).To(Equal(r.Key))
			})
		})
	})

	Describe("Retrieve", func() {
		Describe("WhereKeys", func() {
			It("Should retrieve a role by its key", func() {
				var retrieved role.Role
				err := svc.NewRetrieve().WhereKeys(roles[0].Key).Entry(&retrieved).Exec(ctx, tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(retrieved.Key).To(Equal(roles[0].Key))
				Expect(retrieved.Name).To(Equal(roles[0].Name))
				Expect(retrieved.Description).To(Equal(roles[0].Description))
			})

			It("Should retrieve multiple roles by keys", func() {
				var retrieved []role.Role
				keys := []uuid.UUID{roles[0].Key, roles[1].Key}
				err := svc.NewRetrieve().WhereKeys(keys...).Entries(&retrieved).Exec(ctx, tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(retrieved).To(HaveLen(2))
				Expect(retrieved).To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Key":  Equal(roles[0].Key),
					"Name": Equal(roles[0].Name),
				})))
				Expect(retrieved).To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Key":  Equal(roles[1].Key),
					"Name": Equal(roles[1].Name),
				})))
			})

			It("Should return NotFound when role doesn't exist", func() {
				var retrieved role.Role
				nonExistentKey := uuid.New()
				err := svc.NewRetrieve().WhereKeys(nonExistentKey).Entry(&retrieved).Exec(ctx, tx)
				Expect(err).To(MatchError(query.NotFound))
			})

			It("Should retrieve all roles when no filter is applied", func() {
				var retrieved []role.Role
				err := svc.NewRetrieve().Entries(&retrieved).Exec(ctx, tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(retrieved)).To(BeNumerically(">=", len(roles)))
			})
		})

		Describe("WhereName", func() {
			It("Should retrieve a role by its name", func() {
				var retrieved role.Role
				err := svc.NewRetrieve().WhereName("Engineer").Entry(&retrieved).Exec(ctx, tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(retrieved.Name).To(Equal("Engineer"))
				Expect(retrieved.Key).To(Equal(roles[0].Key))
			})

			It("Should return NotFound when name doesn't exist", func() {
				var retrieved role.Role
				err := svc.NewRetrieve().WhereName("NonExistent").Entry(&retrieved).Exec(ctx, tx)
				Expect(err).To(MatchError(query.NotFound))
			})

			It("Should be case-sensitive", func() {
				var retrieved role.Role
				err := svc.NewRetrieve().WhereName("engineer").Entry(&retrieved).Exec(ctx, tx)
				Expect(err).To(MatchError(query.NotFound))
			})
		})

		Describe("WhereInternal", func() {
			It("Should retrieve only builtin roles", func() {
				var retrieved []role.Role
				err := svc.NewRetrieve().WhereInternal(true).Entries(&retrieved).Exec(ctx, tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(retrieved)).To(BeNumerically(">=", 1))
				for _, r := range retrieved {
					Expect(r.Internal).To(BeTrue())
				}
			})

			It("Should retrieve only custom (non-builtin) roles", func() {
				var retrieved []role.Role
				err := svc.NewRetrieve().WhereInternal(false).Entries(&retrieved).Exec(ctx, tx)
				Expect(err).ToNot(HaveOccurred())
				// Should have Engineer and Viewer (2 non-builtin roles created earlier)
				Expect(len(retrieved)).To(BeNumerically(">=", 2))
				for _, r := range retrieved {
					Expect(r.Internal).To(BeFalse())
				}
			})
		})

		Describe("Filter Combinations", func() {
			It("Should combine WhereKeys and WhereInternal", func() {
				var retrieved []role.Role
				err := svc.NewRetrieve().
					WhereKeys(roles[2].Key).
					WhereInternal(true).
					Entries(&retrieved).
					Exec(ctx, tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(retrieved).To(HaveLen(1))
				Expect(retrieved[0].Key).To(Equal(roles[2].Key))
				Expect(retrieved[0].Internal).To(BeTrue())
			})
		})

		Describe("Entry vs Entries", func() {
			It("Should use Entry for single role retrieval", func() {
				var retrieved role.Role
				err := svc.NewRetrieve().WhereKeys(roles[0].Key).Entry(&retrieved).Exec(ctx, tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(retrieved.Key).To(Equal(roles[0].Key))
			})

			It("Should use Entries for multiple role retrieval", func() {
				var retrieved []role.Role
				err := svc.NewRetrieve().Entries(&retrieved).Exec(ctx, tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(retrieved)).To(BeNumerically(">=", 3))
			})
		})
	})

	Describe("Role Entity", func() {
		Describe("GorpKey", func() {
			It("Should return the role's UUID", func() {
				r := role.Role{Key: uuid.New()}
				Expect(r.GorpKey()).To(Equal(r.Key))
			})
		})

		Describe("OntologyID", func() {
			It("Should return the correct ontology ID", func() {
				key := uuid.New()
				r := role.Role{Key: key}
				ontologyID := r.OntologyID()
				Expect(ontologyID.Type).To(Equal(ontology.Type("role")))
				Expect(ontologyID.Key).To(Equal(key.String()))
			})
		})

		Describe("SetOptions", func() {
			It("Should return nil", func() {
				r := role.Role{}
				Expect(r.SetOptions()).To(BeNil())
			})
		})
	})

	Describe("Ontology Functions", func() {
		Describe("OntologyID", func() {
			It("Should construct an ontology ID from a UUID", func() {
				key := uuid.New()
				id := role.OntologyID(key)
				Expect(id.Type).To(Equal(ontology.Type("role")))
				Expect(id.Key).To(Equal(key.String()))
			})
		})

		Describe("OntologyIDs", func() {
			It("Should construct ontology ResourceIDs from a slice of UUIDs", func() {
				keys := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
				ids := role.OntologyIDs(keys)
				Expect(ids).To(HaveLen(3))
				for i, id := range ids {
					Expect(id.Type).To(Equal(ontology.Type("role")))
					Expect(id.Key).To(Equal(keys[i].String()))
				}
			})

			It("Should return empty slice for empty input", func() {
				ids := role.OntologyIDs([]uuid.UUID{})
				Expect(ids).To(BeEmpty())
			})
		})

		Describe("OntologyIDsFromPolicies", func() {
			It("Should construct ontology ResourceIDs from a slice of Roles", func() {
				testRoles := []role.Role{
					{Key: uuid.New(), Name: "Role1"},
					{Key: uuid.New(), Name: "Role2"},
				}
				ids := role.OntologyIDsFromPolicies(testRoles)
				Expect(ids).To(HaveLen(2))
				for i, id := range ids {
					Expect(id.Type).To(Equal(ontology.Type("role")))
					Expect(id.Key).To(Equal(testRoles[i].Key.String()))
				}
			})
		})

		Describe("KeysFromOntologyIds", func() {
			It("Should extract UUIDs from ontology ResourceIDs", func() {
				keys := []uuid.UUID{uuid.New(), uuid.New()}
				ids := role.OntologyIDs(keys)
				extractedKeys, err := role.KeysFromOntologyIds(ids)
				Expect(err).ToNot(HaveOccurred())
				Expect(extractedKeys).To(Equal(keys))
			})

			It("Should return an error for invalid UUID strings", func() {
				ids := []ontology.ID{{Type: "role", Key: "invalid-uuid"}}
				_, err := role.KeysFromOntologyIds(ids)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Policy Management", func() {
		It("Should store and retrieve policy UUIDs in a role", func() {
			policyKeys := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
			r := &role.Role{
				Name:        "PolicyHolder",
				Description: "Role with multiple policies",
				Policies:    policyKeys,
				Internal:    false,
			}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, r)).To(Succeed())

			// Retrieve and verify policies
			var retrieved role.Role
			err := svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieved).Exec(ctx, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(retrieved.Policies).To(HaveLen(3))
			Expect(retrieved.Policies).To(ConsistOf(policyKeys))
		})

		It("Should handle empty policy list", func() {
			r := &role.Role{
				Name:        "NoPolicies",
				Description: "Role with no policies",
				Policies:    []uuid.UUID{},
				Internal:    false,
			}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, r)).To(Succeed())

			var retrieved role.Role
			err := svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieved).Exec(ctx, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(retrieved.Policies).To(BeEmpty())
		})
	})

	Describe("Transaction Behavior", func() {
		It("Should isolate writes in separate transactions", func() {
			tx1 := db.OpenTx()
			tx2 := db.OpenTx()

			// Create role in tx1
			r := &role.Role{Name: "TxTest", Description: "Transaction test"}
			w1 := svc.NewWriter(tx1)
			Expect(w1.Create(ctx, r)).To(Succeed())

			// Should not be visible in tx2
			var retrieved role.Role
			err := svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieved).Exec(ctx, tx2)
			Expect(err).To(MatchError(query.NotFound))

			// Close transactions
			Expect(tx1.Close()).To(Succeed())
			Expect(tx2.Close()).To(Succeed())
		})
	})

	Describe("Role Assignment", Ordered, func() {
		var (
			testRole    role.Role
			userID      ontology.ID
			anotherRole role.Role
		)

		BeforeAll(func() {
			// Create a test role
			testRole = role.Role{
				Name:        "TestAssignment",
				Description: "Role for testing assignment",
				Policies:    []uuid.UUID{uuid.New()},
				Internal:    false,
			}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &testRole)).To(Succeed())

			// Create another role
			anotherRole = role.Role{
				Name:        "AnotherRole",
				Description: "Another role for testing",
				Policies:    []uuid.UUID{},
				Internal:    false,
			}
			Expect(w.Create(ctx, &anotherRole)).To(Succeed())

			// Create a mock user ID and define it in the ontology
			userID = ontology.ID{Type: "user", Key: uuid.New().String()}
			otgWriter := otg.NewWriter(tx)
			Expect(otgWriter.DefineResource(ctx, userID)).To(Succeed())
		})

		Describe("AssignRole", func() {
			It("Should assign a role to a subject", func() {
				w := svc.NewWriter(tx)
				err := w.AssignRole(ctx, userID, testRole.Key)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Should be idempotent - assigning the same role multiple times should not error", func() {
				w := svc.NewWriter(tx)
				err := w.AssignRole(ctx, userID, testRole.Key)
				Expect(err).ToNot(HaveOccurred())

				// Assign again
				err = w.AssignRole(ctx, userID, testRole.Key)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Should allow assigning multiple roles to the same subject", func() {
				w := svc.NewWriter(tx)
				err := w.AssignRole(ctx, userID, anotherRole.Key)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("UnassignRole", func() {
			It("Should unassign a role from a subject", func() {
				w := svc.NewWriter(tx)
				err := w.UnassignRole(ctx, userID, anotherRole.Key)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Should be a no-op when unassigning a non-existent relationship", func() {
				w := svc.NewWriter(tx)
				nonExistentUserID := ontology.ID{Type: "user", Key: uuid.New().String()}
				err := w.UnassignRole(ctx, nonExistentUserID, testRole.Key)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Should allow unassigning the same role multiple times", func() {
				w := svc.NewWriter(tx)
				err := w.UnassignRole(ctx, userID, anotherRole.Key)
				Expect(err).ToNot(HaveOccurred())

				// Unassign again
				err = w.UnassignRole(ctx, userID, anotherRole.Key)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
