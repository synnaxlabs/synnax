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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Status", Ordered, func() {
	var (
		db       *gorp.DB
		svc      *status.Service
		w        status.Writer
		labelSvc *label.Service
		otg      *ontology.Ontology
		tx       gorp.Tx
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		// Like the Core on the node that leases each write, the indexes see only the
		// writes made through their table.
		db = DeferClose(gorp.Wrap(
			memkv.New(),
			gorp.WithIndexObservable(observe.Noop[kv.TxReader]{}),
		))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{
			DB: db,
		}))
		g := MustOpen(
			group.OpenService(
				ctx,
				group.ServiceConfig{DB: db, Ontology: otg},
			),
		)
		labelSvc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		svc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Label:    labelSvc,
			Group:    g,
		}))
	})
	BeforeEach(func(ctx SpecContext) {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func(ctx SpecContext) {
		Expect(tx.Close()).To(Succeed())
	})

	Describe("Writer", func() {
		Describe("Set", func() {
			It("Should create a new status", func(ctx SpecContext) {
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
			It("Should update an existing status", func(ctx SpecContext) {
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
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchKeys[any]("update-key")).
						Entry(&retrieved).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(retrieved.Message).To(Equal("Updated message"))
				Expect(retrieved.Variant).To(Equal(status.Variant("warning")))
			})
			Context("Parent Management", func() {
				It("Should set a custom parent for the status", func(ctx SpecContext) {
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
					Expect(
						w.SetWithParent(ctx, &child, parent.OntologyID()),
					).To(Succeed())

					var res ontology.Resource
					Expect(otg.NewRetrieve().
						WhereIDs(child.OntologyID()).
						TraverseTo(ontology.ParentsTraverser).
						Entry(&res).
						Exec(ctx, tx)).To(Succeed())
					Expect(res.ID).To(Equal(parent.OntologyID()))
				})
			})
		})

		Describe("SetMany", func() {
			It("Should create multiple statuses", func(ctx SpecContext) {
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
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchKeys[any]("key1", "key2")).
						Entries(&retrieved).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(retrieved).To(HaveLen(2))
			})
		})

		Describe("Rename", func() {
			It(
				"Should rename a status, keeping every other field",
				func(ctx SpecContext) {
					type details struct {
						// Device is the key of the device the status reports on.
						Device string `json:"device" msgpack:"device"`
						// Count is past float64 precision, so a lossy rename changes
						// it.
						Count int64 `json:"count" msgpack:"count"`
					}
					s := &status.Status[details]{
						Name:        "Before",
						Key:         "rename-key",
						Variant:     status.VariantSuccess,
						Message:     "Connected",
						Description: "Device is connected",
						Time:        telem.Now(),
						Details:     details{Device: "dev1", Count: 9007199254740993},
					}
					Expect(w.Set(ctx, s)).To(Succeed())
					Expect(w.Rename(ctx, "rename-key", "After")).To(Succeed())
					var res status.Status[details]
					Expect(svc.NewRetrieve[details]().
						Where(status.MatchKeys[details]("rename-key")).
						Entry(&res).
						Exec(ctx, tx)).To(Succeed())
					expected := *s
					expected.Name = "After"
					Expect(res).To(Equal(expected))
				},
			)
			It(
				"Should return a validation error for an empty name",
				func(ctx SpecContext) {
					s := &status.Status[any]{
						Name:    "Before",
						Key:     "rename-empty-key",
						Variant: status.VariantSuccess,
						Time:    telem.Now(),
					}
					Expect(w.Set(ctx, s)).To(Succeed())
					Expect(w.Rename(ctx, "rename-empty-key", "")).
						To(MatchError(ContainSubstring("name: required")))
				},
			)
			It(
				"Should return a not found error for a missing status",
				func(ctx SpecContext) {
					Expect(w.Rename(ctx, "missing-rename-key", "After")).
						To(MatchError(query.ErrNotFound))
				},
			)
		})

		Describe("Delete", func() {
			It("Should delete a status", func(ctx SpecContext) {
				s := &status.Status[any]{
					Name:    "To Delete",
					Key:     "delete-key",
					Variant: "info",
					Message: "Will be deleted",
					Time:    telem.Now(),
				}
				Expect(w.Set(ctx, s)).To(Succeed())
				Expect(w.Delete(ctx, "delete-key")).To(Succeed())

				err := svc.NewRetrieve[any]().
					Where(status.MatchKeys[any]("delete-key")).
					Entry(&status.Status[any]{}).
					Exec(ctx, tx)
				Expect(err).To(MatchError(query.ErrNotFound))
			})

			It("Should be idempotent", func(ctx SpecContext) {
				Expect(w.Delete(ctx, "non-existent-key")).To(Succeed())
			})

			It("Should delete multiple statuses", func(ctx SpecContext) {
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
				Expect(w.Delete(ctx, "del1", "del2")).To(Succeed())

				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchKeys[any]("del1", "del2")).
						Exec(ctx, tx),
				).To(MatchError(query.ErrNotFound))
			})
		})
	})

	Describe("Retrieve", func() {
		BeforeEach(func(ctx SpecContext) {
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
			It("Should retrieve status by key", func(ctx SpecContext) {
				var s status.Status[any]
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchKeys[any]("retrieve-a")).
						Entry(&s).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(s.Key).To(Equal("retrieve-a"))
				Expect(s.Name).To(Equal("Status A"))
			})

			It("Should retrieve multiple statuses by keys", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchKeys[any]("retrieve-a", "retrieve-b")).
						Entries(&statuses).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(statuses).To(HaveLen(2))
			})
		})

		Describe("SkipMissingKeys", func() {
			It("Should return not found for a missing key without it", func(
				ctx SpecContext,
			) {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve[any]().
					Where(status.MatchKeys[any]("retrieve-a", "missing")).
					Entries(&statuses).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			})

			It("Should return the found statuses with it", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve[any]().
					Where(status.MatchKeys[any]("retrieve-a", "missing")).
					SkipMissingKeys().
					Entries(&statuses).
					Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveExactElements(HaveField("Key", "retrieve-a")))
			})
		})

		Describe("Limit and Offset", func() {
			It("Should limit results", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(
					svc.NewRetrieve[any]().Limit(2).Entries(&statuses).Exec(ctx, tx),
				).To(Succeed())
				Expect(statuses).To(HaveLen(2))
			})

			It("Should offset results", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(
					svc.NewRetrieve[any]().
						Offset(1).
						Limit(2).
						Entries(&statuses).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(statuses).To(HaveLen(2))
			})
		})

		Describe("MatchSearch", func() {
			It(
				"Should return the matching statuses, closest match first",
				func(ctx SpecContext) {
					var statuses []status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchSearch("Status A")).
							Entries(&statuses).
							Exec(ctx, tx),
					).To(Succeed())
					Expect(len(statuses)).To(BeNumerically(">", 1))
					Expect(statuses[0].Key).To(Equal("retrieve-a"))
				},
			)

			It("Should page through the matches by relevance", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchSearch("Status A")).
						Limit(1).
						Entries(&statuses).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(statuses).To(HaveExactElements(HaveField("Key", "retrieve-a")))
			})

			It("Should match no status for an empty term", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchSearch("")).
						Entries(&statuses).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(statuses).To(BeEmpty())
			})

			It(
				"Should skip a match that the transaction no longer holds",
				func(ctx SpecContext) {
					// The delete skips the table, so the search index still holds the
					// status.
					Expect(gorp.NewDelete[status.Key, status.Status[any]]().
						Where(gorp.MatchKeys[status.Key, status.Status[any]](
							"retrieve-b",
						)).
						Exec(ctx, tx)).To(Succeed())
					var statuses []status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchSearch("Status")).
							Entries(&statuses).
							Exec(ctx, tx),
					).To(Succeed())
					Expect(statuses).To(ContainElement(HaveField("Key", "retrieve-a")))
					Expect(
						statuses,
					).ToNot(ContainElement(HaveField("Key", "retrieve-b")))
				},
			)
		})

		Describe("MatchVariants", func() {
			It("Should retrieve statuses with a single variant", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchVariants[any](status.VariantInfo)).
						Entries(&statuses).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(statuses).To(HaveLen(3))
				for _, s := range statuses {
					Expect(s.Variant).To(Equal(status.VariantInfo))
				}
			})

			It(
				"Should retrieve statuses with multiple variants",
				func(ctx SpecContext) {
					var statuses []status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchVariants[any](status.VariantInfo, status.VariantWarning)).
							Entries(&statuses).
							Exec(ctx, tx),
					).To(Succeed())
					Expect(statuses).To(HaveLen(5))
					for _, s := range statuses {
						Expect(
							s.Variant,
						).To(SatisfyAny(Equal(status.VariantInfo), Equal(status.VariantWarning)))
					}
				},
			)

			It(
				"Should return empty when no statuses match variant",
				func(ctx SpecContext) {
					var statuses []status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchVariants[any](status.VariantSuccess)).
							Entries(&statuses).
							Exec(ctx, tx),
					).To(Succeed())
					Expect(statuses).To(BeEmpty())
				},
			)

			It("Should retrieve only error variant statuses", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchVariants[any](status.VariantError)).
						Entries(&statuses).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(statuses).To(HaveLen(1))
				Expect(statuses[0].Key).To(Equal("retrieve-c"))
			})
		})

		Describe("MatchNames", func() {
			It("Should retrieve a status with a single name", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchNames[any]("Status A")).
						Entries(&statuses).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(statuses).To(HaveLen(1))
				Expect(statuses[0].Key).To(Equal("retrieve-a"))
			})

			It("Should retrieve statuses with multiple names", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchNames[any]("Status A", "Status B")).
						Entries(&statuses).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(statuses).To(HaveLen(2))
				for _, s := range statuses {
					Expect(s.Name).To(SatisfyAny(Equal("Status A"), Equal("Status B")))
				}
			})

			It(
				"Should return empty when no statuses match name",
				func(ctx SpecContext) {
					var statuses []status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchNames[any]("Nonexistent")).
							Entries(&statuses).
							Exec(ctx, tx),
					).To(Succeed())
					Expect(statuses).To(BeEmpty())
				},
			)
		})

		Describe("MatchLabels", func() {
			It(
				"Should retrieve statuses that have any of the provided labels",
				func(ctx SpecContext) {
					l := &label.Label{Name: "status-label-a"}
					Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
					s := &status.Status[any]{
						Key:     "labeled-status",
						Name:    "Labeled",
						Variant: status.VariantInfo,
						Time:    telem.Now(),
					}
					Expect(svc.NewWriter(tx).Set(ctx, s)).To(Succeed())
					Expect(
						labelSvc.NewWriter(tx).
							Label(ctx, s.OntologyID(), []label.Key{l.Key}),
					).To(Succeed())
					var statuses []status.Status[any]
					Expect(svc.NewRetrieve[any]().
						Where(status.MatchLabels[any](l.Key)).
						Entries(&statuses).
						Exec(ctx, tx)).To(Succeed())
					keys := make([]status.Key, len(statuses))
					for i, st := range statuses {
						keys[i] = st.Key
					}
					Expect(keys).To(ContainElement("labeled-status"))
				},
			)
			It(
				"Should return empty when no statuses have the specified label",
				func(ctx SpecContext) {
					l := &label.Label{Name: "status-unused-label"}
					Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
					var statuses []status.Status[any]
					Expect(svc.NewRetrieve[any]().
						Where(status.MatchLabels[any](l.Key)).
						Entries(&statuses).
						Exec(ctx, tx)).To(Succeed())
					Expect(statuses).To(BeEmpty())
				},
			)
		})

		Describe("Combinators", func() {
			It("Should wrap a closure via Match", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve[any]().Where(
					status.Match(func(_ gorp.Context, _ status.Retrieve[any], s *status.Status[any]) (bool, error) {
						return s.Key == "retrieve-a", nil
					}),
				).
					Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(1))
				Expect(statuses[0].Key).To(Equal("retrieve-a"))
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

		It(
			"Should store and retrieve every field of a status with typed details",
			func(ctx SpecContext) {
				type IntDetails struct {
					Count int
				}
				intWriter := svc.NewWriter(tx)
				s := &status.Status[IntDetails]{
					Key:         "typed-int-status",
					Name:        "Typed Int Status",
					Variant:     "info",
					Message:     "Counted",
					Description: "The counter reached its target.",
					Details:     IntDetails{Count: 42},
					Time:        telem.Now(),
				}
				Expect(intWriter.Set(ctx, s)).To(Succeed())

				var retrieved status.Status[IntDetails]
				intRetrieve := svc.NewRetrieve[IntDetails]()
				Expect(
					intRetrieve.Where(status.MatchKeys[IntDetails]("typed-int-status")).
						Entry(&retrieved).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(retrieved).To(Equal(*s))
			},
		)

		It(
			"Should retrieve statuses stored with different detail types using any",
			func(ctx SpecContext) {
				// Store a status with specific typed details
				type StringDetails struct {
					Message string
				}
				typedWriter := svc.NewWriter(tx)
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
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchKeys[any]("typed-string-status")).
						Entry(&retrieved).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(retrieved.Key).To(Equal("typed-string-status"))
				// Details will be decoded as map[string]interface{} when using any
				details, ok := retrieved.Details.(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(details["Message"]).To(Equal("hello"))
			},
		)

		It(
			"Should allow retrieval of all statuses regardless of detail type",
			func(ctx SpecContext) {
				// Store statuses with different detail types
				type TypeA struct{ ValueA int }
				type TypeB struct{ ValueB string }

				writerA := svc.NewWriter(tx)
				writerB := svc.NewWriter(tx)

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
				Expect(
					svc.NewRetrieve[any]().
						Where(status.MatchKeys[any]("generic-type-a", "generic-type-b")).
						Entries(&statuses).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(statuses).To(HaveLen(2))
			},
		)

		It(
			"Should decode mismatched types with zero values for missing fields",
			func(ctx SpecContext) {
				// Store a status with TypeA details
				type TypeA struct {
					FieldB string
					FieldA int
				}
				writerA := svc.NewWriter(tx)
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
				retrieveC := svc.NewRetrieve[TypeC]()
				Expect(
					retrieveC.Where(status.MatchKeys[TypeC]("mismatch-test")).
						Entry(&retrieved).
						Exec(ctx, tx),
				).To(Succeed())

				// The status is retrieved successfully, but Details has zero values
				// because TypeC's fields don't match TypeA's fields
				Expect(retrieved.Key).To(Equal("mismatch-test"))
				Expect(retrieved.Name).To(Equal("Mismatch Test"))
				Expect(retrieved.Details.FieldC).To(Equal(float64(0)))
				Expect(retrieved.Details.FieldD).To(BeFalse())
			},
		)
		It("should not work when details are not a struct", func(ctx SpecContext) {
			type DetailsA int
			type DetailsB string
			writerB := svc.NewWriter(tx)
			Expect(writerB.Set(ctx, &status.Status[DetailsB]{
				Key: "details-b", Name: "Details B", Variant: "info",
				Details: DetailsB("hello"), Time: telem.Now(),
			})).To(Succeed())
			var retrieved status.Status[DetailsA]
			retrieveA := svc.NewRetrieve[DetailsA]()
			Expect(
				retrieveA.Entry(&retrieved).Exec(ctx, tx),
			).To(MatchError(ContainSubstring("all codecs failed to decode")))
		})
	})

	Describe("Observe", func() {
		It("Should notify when a status is created", func(ctx SpecContext) {
			tx := db.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()
			w := svc.NewWriter(tx)
			s := &status.Status[any]{
				Key: "observe-test", Name: "Observe Test",
				Variant: status.VariantSuccess, Time: telem.Now(),
			}
			Expect(w.Set(ctx, s)).To(Succeed())
			called := false
			svc.Observe().
				OnChange(func(context.Context, gorp.TxReader[status.Key, status.Status[any]]) {
					called = true
				})
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(called).To(BeTrue())
		})
	})

	Describe("Search", func() {
		create := func(ctx SpecContext, key, name string) ontology.ID {
			GinkgoHelper()
			s := status.Status[any]{
				Key:     key,
				Name:    name,
				Variant: "success",
				Time:    telem.Now(),
			}
			Expect(svc.NewWriter(nil).Set(ctx, &s)).To(Succeed())
			return status.OntologyID(s.Key)
		}

		It(
			"Should return the matching statuses, closest match first",
			func(ctx SpecContext) {
				a := create(ctx, "a", "turbopump test")
				b := create(ctx, "b", "turbopump")
				Expect(MustSucceed(svc.Search(ctx, db, "turbopump", 0)).Keys).
					To(Equal([]ontology.ID{b, a}))
				Expect(MustSucceed(svc.Search(ctx, db, "turbopump", 1)).Keys).
					To(Equal([]ontology.ID{b}))
			},
		)

		It("Should match a renamed status by its new name", func(ctx SpecContext) {
			id := create(ctx, "c", "cryogenic")
			Expect(svc.NewWriter(nil).Rename(ctx, "c", "hydraulic")).To(Succeed())
			Expect(MustSucceed(svc.Search(ctx, db, "hydraulic", 0)).Keys).
				To(Equal([]ontology.ID{id}))
			Expect(MustSucceed(svc.Search(ctx, db, "cryogenic", 0)).Keys).
				To(BeEmpty())
		})
	})
})
