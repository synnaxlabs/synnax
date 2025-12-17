// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Status", Ordered, func() {
	var (
		db       *gorp.DB
		svc      *status.Service
		w        status.Writer[any]
		labelSvc *label.Service
		otg      *ontology.Ontology
		tx       gorp.Tx
		closer   io.Closer
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{
			DB:           db,
			EnableSearch: config.True(),
		}))
		g := MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))
		labelSvc = MustSucceed(label.OpenService(ctx, label.Config{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		svc = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Label:    labelSvc,
			Group:    g,
		}))
		Expect(otg.InitializeSearchIndex(ctx)).To(Succeed())

		closer = xio.MultiCloser{db, otg, g, svc}
	})
	AfterAll(func() {
		Expect(labelSvc.Close()).To(Succeed())
		Expect(svc.Close()).To(Succeed())
		Expect(closer.Close()).To(Succeed())
	})
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})

	Describe("Writer", func() {
		Describe("Set", func() {
			It("Should create a new status", func() {
				s := &status.Status[any]{
					Name:    "Test Status",
					Key:     "test-key",
					Variant: "success",
					Message: "Test message",
					Time:    telem.Now(),
				}
				Expect(w.Set(ctx, s)).To(Succeed())
				Expect(s.Key).To(Equal("test-key"))
			})
			It("Should update an existing status", func() {
				s := &status.Status[any]{
					Name:    "Test Status",
					Key:     "update-key",
					Variant: "info",
					Message: "Initial message",
					Time:    telem.Now(),
				}
				Expect(w.Set(ctx, s)).To(Succeed())
				s.Message = "Updated message"
				s.Variant = "warning"
				Expect(w.Set(ctx, s)).To(Succeed())

				var retrieved status.Status[any]
				Expect(svc.NewRetrieve().WhereKeys("update-key").Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
				Expect(retrieved.Message).To(Equal("Updated message"))
				Expect(retrieved.Variant).To(Equal(xstatus.Variant("warning")))
			})
			Context("Parent Management", func() {
				It("Should set a custom parent for the status", func() {
					parent := status.Status[any]{
						Name:    "Parent Status",
						Key:     "parent-key",
						Variant: "info",
						Message: "Parent status",
						Time:    telem.Now(),
					}
					Expect(w.Set(ctx, &parent)).To(Succeed())

					child := status.Status[any]{
						Name:    "Child Status",
						Key:     "child-key",
						Variant: "info",
						Message: "Child status",
						Time:    telem.Now(),
					}
					Expect(w.SetWithParent(ctx, &child, status.OntologyID(parent.Key))).To(Succeed())

					var res ontology.Resource
					Expect(otg.NewRetrieve().
						WhereIDs(status.OntologyID(child.Key)).
						TraverseTo(ontology.Parents).
						Entry(&res).
						Exec(ctx, tx)).To(Succeed())
					Expect(res.ID).To(Equal(status.OntologyID(parent.Key)))
				})
			})
		})

		Describe("SetMany", func() {
			It("Should create multiple statuses", func() {
				statuses := []status.Status[any]{
					{
						Name:    "Status 1",
						Key:     "key1",
						Variant: "info",
						Message: "Message 1",
						Time:    telem.Now(),
					},
					{
						Name:    "Status 2",
						Key:     "key2",
						Variant: "warning",
						Message: "Message 2",
						Time:    telem.Now(),
					},
				}
				Expect(w.SetMany(ctx, &statuses)).To(Succeed())

				var retrieved []status.Status[any]
				Expect(svc.NewRetrieve().WhereKeys("key1", "key2").Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
				Expect(retrieved).To(HaveLen(2))
			})
		})

		Describe("Delete", func() {
			It("Should delete a status", func() {
				s := &status.Status[any]{
					Name:    "To Delete",
					Key:     "delete-key",
					Variant: "info",
					Message: "Will be deleted",
					Time:    telem.Now(),
				}
				Expect(w.Set(ctx, s)).To(Succeed())
				Expect(w.Delete(ctx, "delete-key")).To(Succeed())

				err := svc.NewRetrieve().WhereKeys("delete-key").Entry(&status.Status[any]{}).Exec(ctx, tx)
				Expect(err).To(MatchError(query.NotFound))
			})

			It("Should be idempotent", func() {
				Expect(w.Delete(ctx, "non-existent-key")).To(Succeed())
			})
		})

		Describe("DeleteMany", func() {
			It("Should delete multiple statuses", func() {
				statuses := []status.Status[any]{
					{
						Name:    "Del 1",
						Key:     "del1",
						Variant: "info",
						Time:    telem.Now(),
					},
					{
						Name:    "Del 2",
						Key:     "del2",
						Variant: "info",
						Time:    telem.Now(),
					},
				}
				Expect(w.SetMany(ctx, &statuses)).To(Succeed())
				Expect(w.DeleteMany(ctx, "del1", "del2")).To(Succeed())

				Expect(svc.NewRetrieve().WhereKeys("del1", "del2").Exec(ctx, tx)).To(HaveOccurredAs(query.NotFound))
			})
		})
	})

	Describe("Retrieve", func() {
		BeforeEach(func() {
			statuses := []status.Status[any]{
				{
					Name:    "Status A",
					Key:     "retrieve-a",
					Variant: "info",
					Message: "Status A message",
					Time:    telem.Now(),
				},
				{
					Name:    "Status B",
					Key:     "retrieve-b",
					Variant: "warning",
					Message: "Status B message",
					Time:    telem.Now(),
				},
				{
					Name:    "Status C",
					Key:     "retrieve-c",
					Variant: "error",
					Message: "Status C message",
					Time:    telem.Now(),
				},
				{
					Name:    "Device 1 Status",
					Key:     "device-001-status",
					Variant: "info",
					Message: "Device 1 OK",
					Time:    telem.Now(),
				},
				{
					Name:    "Device 2 Status",
					Key:     "device-002-status",
					Variant: "warning",
					Message: "Device 2 Warning",
					Time:    telem.Now(),
				},
				{
					Name:    "Sensor Status",
					Key:     "sensor-001-status",
					Variant: "info",
					Message: "Sensor OK",
					Time:    telem.Now(),
				},
			}
			Expect(w.SetMany(ctx, &statuses)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			tx = db.OpenTx()
			w = svc.NewWriter(tx)
		})

		Describe("WhereKeys", func() {
			It("Should retrieve status by key", func() {
				var s status.Status[any]
				Expect(svc.NewRetrieve().WhereKeys("retrieve-a").Entry(&s).Exec(ctx, tx)).To(Succeed())
				Expect(s.Key).To(Equal("retrieve-a"))
				Expect(s.Name).To(Equal("Status A"))
			})

			It("Should retrieve multiple statuses by keys", func() {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve().WhereKeys("retrieve-a", "retrieve-b").Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(2))
			})
		})

		Describe("Limit and Offset", func() {
			It("Should limit results", func() {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve().Limit(2).Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(2))
			})

			It("Should offset results", func() {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve().Offset(1).Limit(2).Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(2))
			})
		})

		Describe("Search", func() {
			It("Should search for statuses", func() {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve().Search("Status A").Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(len(statuses)).To(BeNumerically(">", 1))
				Expect(statuses[0].Key).To(Equal("retrieve-a"))
			})
		})

		Describe("WhereKeyPrefix", func() {
			It("Should retrieve statuses with matching key prefix", func() {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve().WhereKeyPrefix("device-").Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(2))
				for _, s := range statuses {
					Expect(s.Key).To(HavePrefix("device-"))
				}
			})

			It("Should retrieve statuses with different prefix", func() {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve().WhereKeyPrefix("sensor-").Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(1))
				Expect(statuses[0].Key).To(Equal("sensor-001-status"))
			})

			It("Should return empty when no statuses match prefix", func() {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve().WhereKeyPrefix("nonexistent-").Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(BeEmpty())
			})

			It("Should retrieve statuses with retrieve- prefix", func() {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve().WhereKeyPrefix("retrieve-").Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(3))
			})
		})
	})

	Describe("Generic Type Behavior", func() {
		// This test suite documents how generic types work with Status[D] in gorp.
		// All Status[D] types share the same gorp namespace because CustomTypeName()
		// returns "Status" regardless of the type parameter D. This means:
		// 1. Statuses stored with different D types are stored in the same namespace
		// 2. Retrieval does not filter by the generic type parameter
		// 3. The generic type only affects how the Details field is decoded

		It("Should store and retrieve statuses with typed details", func() {
			type IntDetails struct {
				Count int
			}
			intWriter := status.NewWriter[IntDetails](svc, tx)
			s := &status.Status[IntDetails]{
				Key:     "typed-int-status",
				Name:    "Typed Int Status",
				Variant: "info",
				Details: IntDetails{Count: 42},
				Time:    telem.Now(),
			}
			Expect(intWriter.Set(ctx, s)).To(Succeed())

			var retrieved status.Status[IntDetails]
			intRetrieve := status.NewRetrieve[IntDetails](svc)
			Expect(intRetrieve.WhereKeys("typed-int-status").Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Details.Count).To(Equal(42))
		})

		It("Should retrieve statuses stored with different detail types using any", func() {
			// Store a status with specific typed details
			type StringDetails struct {
				Message string
			}
			typedWriter := status.NewWriter[StringDetails](svc, tx)
			s := &status.Status[StringDetails]{
				Key:     "typed-string-status",
				Name:    "Typed String Status",
				Variant: "info",
				Details: StringDetails{Message: "hello"},
				Time:    telem.Now(),
			}
			Expect(typedWriter.Set(ctx, s)).To(Succeed())

			// Retrieve using any type - this works because all Status[D] share
			// the same gorp namespace
			var retrieved status.Status[any]
			Expect(svc.NewRetrieve().WhereKeys("typed-string-status").Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Key).To(Equal("typed-string-status"))
			// Details will be decoded as map[string]interface{} when using any
			details, ok := retrieved.Details.(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(details["Message"]).To(Equal("hello"))
		})

		It("Should allow retrieval of all statuses regardless of detail type", func() {
			// Store statuses with different detail types
			type TypeA struct{ ValueA int }
			type TypeB struct{ ValueB string }

			writerA := status.NewWriter[TypeA](svc, tx)
			writerB := status.NewWriter[TypeB](svc, tx)

			Expect(writerA.Set(ctx, &status.Status[TypeA]{
				Key: "generic-type-a", Name: "Type A", Variant: "info",
				Details: TypeA{ValueA: 100}, Time: telem.Now(),
			})).To(Succeed())

			Expect(writerB.Set(ctx, &status.Status[TypeB]{
				Key: "generic-type-b", Name: "Type B", Variant: "info",
				Details: TypeB{ValueB: "test"}, Time: telem.Now(),
			})).To(Succeed())

			// Retrieve both using any - demonstrates that gorp doesn't filter by
			// generic type parameter
			var statuses []status.Status[any]
			Expect(svc.NewRetrieve().WhereKeys("generic-type-a", "generic-type-b").
				Entries(&statuses).Exec(ctx, tx)).To(Succeed())
			Expect(statuses).To(HaveLen(2))
		})

		It("Should decode mismatched types with zero values for missing fields", func() {
			// Store a status with TypeA details
			type TypeA struct {
				FieldA int
				FieldB string
			}
			writerA := status.NewWriter[TypeA](svc, tx)
			Expect(writerA.Set(ctx, &status.Status[TypeA]{
				Key: "mismatch-test", Name: "Mismatch Test", Variant: "info",
				Details: TypeA{FieldA: 42, FieldB: "hello"}, Time: telem.Now(),
			})).To(Succeed())

			// Retrieve with a different type - MsgPack will decode what it can
			// and use zero values for fields that don't match
			type TypeC struct {
				FieldC float64
				FieldD bool
			}
			var retrieved status.Status[TypeC]
			retrieveC := status.NewRetrieve[TypeC](svc)
			Expect(retrieveC.WhereKeys("mismatch-test").Entry(&retrieved).Exec(ctx, tx)).To(Succeed())

			// The status is retrieved successfully, but Details has zero values
			// because TypeC's fields don't match TypeA's fields
			Expect(retrieved.Key).To(Equal("mismatch-test"))
			Expect(retrieved.Name).To(Equal("Mismatch Test"))
			Expect(retrieved.Details.FieldC).To(Equal(float64(0)))
			Expect(retrieved.Details.FieldD).To(BeFalse())
		})
		It("should not work when details are not a struct", func() {
			type DetailsA int
			type DetailsB string
			writerB := status.NewWriter[DetailsB](svc, tx)
			Expect(writerB.Set(ctx, &status.Status[DetailsB]{
				Key: "details-b", Name: "Details B", Variant: "info",
				Details: DetailsB("hello"), Time: telem.Now(),
			})).To(Succeed())
			var retrieved status.Status[DetailsA]
			retrieveA := status.NewRetrieve[DetailsA](svc)
			Expect(retrieveA.Entry(&retrieved).Exec(ctx, tx)).To(Not(Succeed()))
		})
	})
})
