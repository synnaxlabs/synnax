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
	"fmt"
	"sync"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	statusv2 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Status", Ordered, func() {
	var (
		store    kv.DB
		db       *gorp.DB
		svc      *status.Service
		w        status.Writer
		labelSvc *label.Service
		racks    *gorp.Table[rack.Key, rack.Rack]
		tx       gorp.Tx
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		store = memkv.New()
		db = DeferClose(gorp.Wrap(store))
		labelSvc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db,
		}))
		racks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[rack.Key, rack.Rack]{
			DB:   db,
			Name: string(resource.TypeRack),
		}))
		svc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
	})
	openTx := func() {
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx)
	}
	createRack := func(ctx SpecContext, key rack.Key) resource.ID {
		GinkgoHelper()
		Expect(racks.NewCreate().
			Entry(&rack.Rack{Key: key, Name: "Rack"}).
			Exec(ctx, tx)).To(Succeed())
		return key.ResourceID()
	}
	createLabel := func(ctx SpecContext, name string) label.Key {
		GinkgoHelper()
		l := &label.Label{Name: name}
		Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
		return l.Key
	}
	retrieve := func(ctx SpecContext, key status.Key) status.Status[any] {
		GinkgoHelper()
		var s status.Status[any]
		Expect(svc.NewRetrieve[any]().
			Where(status.MatchKeys[any](key)).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	newStatus := func(key status.Key) status.Status[any] {
		return status.Status[any]{
			Key:     key,
			Name:    "Status",
			Variant: status.VariantInfo,
			Time:    telem.Now(),
		}
	}

	Describe("Msgpack storage", func() {
		It(
			"Should rename a status stored as msgpack, keeping every other field",
			func(ctx SpecContext) {
				type details struct {
					Count int64 `json:"count" msgpack:"count"`
				}
				s := status.Status[details]{
					Name:    "Before",
					Key:     "rename-msgpack-key",
					Variant: status.VariantSuccess,
					Time:    telem.Now(),
					Details: details{Count: 9007199254740993},
				}
				Expect(gorp.NewCreate[status.Key, status.Status[details]]().
					Entry(&s).
					Exec(ctx, gorp.Wrap(store, gorp.WithCodec(msgpack.Codec)))).
					To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, s.Key)
					})).To(Succeed())
				})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Rename(ctx, s.Key, "After")
				})).
					To(Succeed())
				var res status.Status[details]
				Expect(svc.NewRetrieve[details]().
					Where(status.MatchKeys[details](s.Key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				expected := s
				expected.Name = "After"
				Expect(res).To(Equal(expected))
			},
		)
	})

	Describe("Writer", func() {
		BeforeEach(openTx)
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
			It(
				"Should assign a UUID key to a status with no key",
				func(ctx SpecContext) {
					s := newStatus("")
					Expect(w.Set(ctx, &s)).To(Succeed())
					MustSucceed(uuid.Parse(s.Key))
				},
			)
			Context("Owner", func() {
				It(
					"Should write to the status the owner holds, whatever the key",
					func(ctx SpecContext) {
						owner := createRack(ctx, 1)
						first := newStatus("")
						first.Owner = &owner
						Expect(w.Set(ctx, &first)).To(Succeed())
						second := newStatus("other-key")
						second.Owner = &owner
						second.Message = "Updated"
						Expect(w.Set(ctx, &second)).To(Succeed())
						Expect(second.Key).To(Equal(first.Key))
						Expect(retrieve(ctx, first.Key).Message).To(Equal("Updated"))
						Expect(svc.NewRetrieve[any]().
							Where(status.MatchKeys[any]("other-key")).
							Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
					},
				)
				It(
					"Should keep the stored owner when a set has none",
					func(ctx SpecContext) {
						owner := createRack(ctx, 2)
						s := newStatus("")
						s.Owner = &owner
						Expect(w.Set(ctx, &s)).To(Succeed())
						next := newStatus(s.Key)
						Expect(w.Set(ctx, &next)).To(Succeed())
						Expect(next.Owner).To(HaveValue(Equal(owner)))
						Expect(retrieve(ctx, s.Key).Owner).To(HaveValue(Equal(owner)))
					},
				)
				It(
					"Should keep the stored owner when an update carries another",
					func(ctx SpecContext) {
						held, other := createRack(ctx, 14), createRack(ctx, 15)
						s := newStatus("")
						s.Owner = &held
						Expect(w.Set(ctx, &s)).To(Succeed())
						next := newStatus(s.Key)
						next.Owner = &other
						Expect(w.Set(ctx, &next)).To(Succeed())
						Expect(next.Owner).To(HaveValue(Equal(held)))
						Expect(retrieve(ctx, s.Key).Owner).To(HaveValue(Equal(held)))
					},
				)
				It(
					"Should give no owner to a stored status that has none",
					func(ctx SpecContext) {
						s := newStatus("")
						Expect(w.Set(ctx, &s)).To(Succeed())
						next := newStatus(s.Key)
						next.Owner = new(createRack(ctx, 16))
						Expect(w.Set(ctx, &next)).To(Succeed())
						Expect(next.Owner).To(BeNil())
						Expect(retrieve(ctx, s.Key).Owner).To(BeNil())
					},
				)
				It("Should reject an owner that does not exist", func(ctx SpecContext) {
					s := newStatus("")
					s.Owner = new(rack.Key(99).ResourceID())
					err := w.Set(ctx, &s)
					Expect(err).To(BeAssignableToTypeOf(validate.PathError{}))
					Expect(err).To(MatchError(
						"owner: rack 99 does not exist: validation error",
					))
				})
				It(
					"Should reject an owner whose type cannot hold a status",
					func(ctx SpecContext) {
						s := newStatus("")
						l := createLabel(ctx, "Owner")
						s.Owner = new(label.ResourceID(l))
						err := w.Set(ctx, &s)
						Expect(err).To(BeAssignableToTypeOf(validate.PathError{}))
						Expect(err).To(MatchError(
							"owner: status cannot reference label " + l.String() + ": validation error",
						))
					},
				)
				It(
					"Should delete the status when its owner is deleted",
					func(ctx SpecContext) {
						owner := createRack(ctx, 3)
						s := newStatus("")
						s.Owner = &owner
						Expect(w.Set(ctx, &s)).To(Succeed())
						Expect(racks.NewDelete().
							Where(gorp.MatchKeys[rack.Key, rack.Rack](3)).
							Exec(ctx, tx)).To(Succeed())
						Expect(svc.NewRetrieve[any]().
							Where(status.MatchKeys[any](s.Key)).
							Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
						Expect(svc.Resources().Retrieve(
							ctx, tx, []resource.ID{status.ResourceID(s.Key)},
						)).To(BeEmpty())
					},
				)
			})
			Context("Labels", func() {
				It("Should store the labels a create carries", func(ctx SpecContext) {
					a := createLabel(ctx, "A")
					s := newStatus("create-labels-key")
					s.Labels = []label.Key{a}
					Expect(w.Set(ctx, &s)).To(Succeed())
					Expect(retrieve(ctx, "create-labels-key").Labels).
						To(Equal([]label.Key{a}))
				})
				It(
					"Should keep the stored labels on an update and ignore the carried ones",
					func(ctx SpecContext) {
						a, b := createLabel(ctx, "A"), createLabel(ctx, "B")
						s := newStatus("update-labels-key")
						s.Labels = []label.Key{a}
						Expect(w.Set(ctx, &s)).To(Succeed())
						next := newStatus("update-labels-key")
						next.Labels = []label.Key{b}
						Expect(w.Set(ctx, &next)).To(Succeed())
						Expect(next.Labels).To(Equal([]label.Key{a}))
						Expect(retrieve(ctx, "update-labels-key").Labels).
							To(Equal([]label.Key{a}))
					},
				)
				It("Should reject a label that does not exist", func(ctx SpecContext) {
					s := newStatus("missing-label-key")
					missing := uuid.New()
					s.Labels = []label.Key{missing}
					err := w.Set(ctx, &s)
					Expect(err).To(BeAssignableToTypeOf(validate.PathError{}))
					Expect(err).To(MatchError(
						"labels: label " + missing.String() + " does not exist: validation error",
					))
				})
				It(
					"Should remove a deleted label from the statuses that hold it",
					func(ctx SpecContext) {
						a, b := createLabel(ctx, "A"), createLabel(ctx, "B")
						s := newStatus("detach-key")
						s.Labels = []label.Key{a, b}
						Expect(w.Set(ctx, &s)).To(Succeed())
						Expect(labelSvc.NewWriter(tx).Delete(ctx, a)).To(Succeed())
						Expect(
							retrieve(ctx, "detach-key").Labels,
						).To(Equal([]label.Key{b}))
					},
				)
			})
		})

		Describe("AddLabels", func() {
			It("Should add labels to a status", func(ctx SpecContext) {
				a, b := createLabel(ctx, "A"), createLabel(ctx, "B")
				s := newStatus("set-labels-key")
				s.Labels = []label.Key{a}
				Expect(w.Set(ctx, &s)).To(Succeed())
				Expect(w.AddLabels(ctx, "set-labels-key", []label.Key{b})).
					To(Succeed())
				Expect(retrieve(ctx, "set-labels-key").Labels).To(ConsistOf(a, b))
			})
			It(
				"Should return a not found error for a missing status",
				func(ctx SpecContext) {
					Expect(w.AddLabels(ctx, "missing-key", nil)).
						To(MatchError(query.ErrNotFound))
				},
			)
		})

		Describe("ReplaceLabels", func() {
			It("Should replace the labels of a status", func(ctx SpecContext) {
				a, b := createLabel(ctx, "A"), createLabel(ctx, "B")
				s := newStatus("replace-labels-key")
				s.Labels = []label.Key{a}
				Expect(w.Set(ctx, &s)).To(Succeed())
				Expect(w.ReplaceLabels(ctx, "replace-labels-key", []label.Key{b})).
					To(Succeed())
				Expect(
					retrieve(ctx, "replace-labels-key").Labels,
				).To(Equal([]label.Key{b}))
			})
			It(
				"Should return a not found error for a missing status",
				func(ctx SpecContext) {
					Expect(w.ReplaceLabels(ctx, "missing-key", nil)).
						To(MatchError(query.ErrNotFound))
				},
			)
		})

		Describe("RemoveLabels", func() {
			It("Should remove labels from a status", func(ctx SpecContext) {
				a, b := createLabel(ctx, "A"), createLabel(ctx, "B")
				s := newStatus("remove-labels-key")
				s.Labels = []label.Key{a, b}
				Expect(w.Set(ctx, &s)).To(Succeed())
				Expect(w.RemoveLabels(ctx, "remove-labels-key", []label.Key{a})).
					To(Succeed())
				Expect(
					retrieve(ctx, "remove-labels-key").Labels,
				).To(Equal([]label.Key{b}))
			})
		})

		Describe("RenameOwned", func() {
			It("Should rename the status an owner holds", func(ctx SpecContext) {
				owner := createRack(ctx, 4)
				s := newStatus("")
				s.Owner = &owner
				Expect(w.Set(ctx, &s)).To(Succeed())
				Expect(w.RenameOwned(ctx, owner, "Renamed")).To(Succeed())
				Expect(retrieve(ctx, s.Key).Name).To(Equal("Renamed"))
			})
			It(
				"Should return a not found error for an owner with no status",
				func(ctx SpecContext) {
					Expect(w.RenameOwned(ctx, createRack(ctx, 5), "Renamed")).
						To(MatchError(query.ErrNotFound))
				},
			)
		})

		Describe("DeleteOwned", func() {
			It("Should delete the status an owner holds", func(ctx SpecContext) {
				owner := createRack(ctx, 11)
				s := newStatus("")
				s.Owner = &owner
				Expect(w.Set(ctx, &s)).To(Succeed())
				Expect(w.DeleteOwned(ctx, owner)).To(Succeed())
				Expect(svc.NewRetrieve[any]().
					Where(status.MatchKeys[any](s.Key)).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			})
			It("Should succeed for an owner with no status", func(ctx SpecContext) {
				Expect(w.DeleteOwned(ctx, createRack(ctx, 12))).To(Succeed())
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

				Expect(svc.NewRetrieve[any]().
					Where(status.MatchKeys[any]("delete-key")).
					Entry(&status.Status[any]{}).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
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
		BeforeEach(openTx)
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
			openTx()
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

		Describe("After", func() {
			It("Should resume strictly past the key", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve[any]().
					Where(status.MatchKeys[any]("retrieve-c", "retrieve-a", "retrieve-b")).
					After("retrieve-a").
					Entries(&statuses).
					Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveExactElements(
					HaveField("Key", "retrieve-b"),
					HaveField("Key", "retrieve-c"),
				))
			})
		})

		Describe("Limit", func() {
			It("Should limit results", func(ctx SpecContext) {
				var statuses []status.Status[any]
				Expect(
					svc.NewRetrieve[any]().Limit(2).Entries(&statuses).Exec(ctx, tx),
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
					Expect(statuses).To(HaveLen(1))
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

		Describe("WherePrefix", func() {
			It("Should retrieve the statuses whose key has the prefix",
				func(ctx SpecContext) {
					var statuses []status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							WherePrefix("retrieve-").
							Where(status.MatchNames[any]("Status A", "Status B")).
							Entries(&statuses).
							Exec(ctx, tx),
					).To(Succeed())
					Expect(statuses).To(ConsistOf(
						HaveField("Key", "retrieve-a"),
						HaveField("Key", "retrieve-b"),
					))
				})

			It(
				"Should retrieve nothing for a prefix no key has",
				func(ctx SpecContext) {
					var statuses []status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							WherePrefix("missing-").
							Entries(&statuses).
							Exec(ctx, tx),
					).To(Succeed())
					Expect(statuses).To(BeEmpty())
				},
			)
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
				"Should not match a name that differs only in case",
				func(ctx SpecContext) {
					lower := newStatus("retrieve-a-lower")
					lower.Name = "status a"
					Expect(w.Set(ctx, &lower)).To(Succeed())
					var statuses []status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchNames[any]("Status A")).
							Entries(&statuses).
							Exec(ctx, tx),
					).To(Succeed())
					Expect(
						statuses,
					).To(HaveExactElements(HaveField("Key", "retrieve-a")))
				},
			)

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
					l := createLabel(ctx, "status-label-a")
					s := newStatus("labeled-status")
					s.Labels = []label.Key{l}
					Expect(svc.NewWriter(tx).Set(ctx, &s)).To(Succeed())
					var statuses []status.Status[any]
					Expect(svc.NewRetrieve[any]().
						Where(status.MatchLabels[any](l)).
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

		Describe("MatchOwners", func() {
			It("Should retrieve the statuses the owners hold", func(ctx SpecContext) {
				a, b := createRack(ctx, 6), createRack(ctx, 7)
				for _, owner := range []resource.ID{a, b} {
					s := newStatus("")
					s.Owner = &owner
					Expect(w.Set(ctx, &s)).To(Succeed())
				}
				var statuses []status.Status[any]
				Expect(svc.NewRetrieve[any]().
					Where(status.MatchOwners[any](a)).
					Entries(&statuses).
					Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(1))
				Expect(statuses[0].Owner).To(HaveValue(Equal(a)))
			})
			It(
				"Should match nothing for an owner with no status",
				func(ctx SpecContext) {
					var statuses []status.Status[any]
					Expect(svc.NewRetrieve[any]().
						Where(status.MatchOwners[any](createRack(ctx, 8))).
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
		BeforeEach(openTx)
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

	Describe("SetMissing", func() {
		BeforeEach(openTx)
		It(
			"Should write a status for each owner that holds none",
			func(ctx SpecContext) {
				held, missing := createRack(ctx, 9), createRack(ctx, 10)
				existing := newStatus("")
				existing.Owner = &held
				existing.Message = "Running"
				Expect(w.Set(ctx, &existing)).To(Succeed())
				backfill := []status.Status[any]{newStatus(""), newStatus("")}
				backfill[0].Owner, backfill[1].Owner = &held, &missing
				Expect(svc.SetMissing(ctx, tx, backfill)).To(Succeed())
				Expect(retrieve(ctx, existing.Key).Message).To(Equal("Running"))
				key, _ := statusv2.MigrateKey(missing.String())
				Expect(retrieve(ctx, key).Owner).To(Equal(&missing))
			},
		)
	})

	Describe("Writer in concurrent transactions", func() {
		It(
			"Should write one status for an owner that concurrent sets write",
			func(ctx SpecContext) {
				Expect(racks.NewCreate().
					Entry(&rack.Rack{Key: 13, Name: "Rack"}).
					Exec(ctx, db)).To(Succeed())
				owner := rack.Key(13).ResourceID()
				var wg sync.WaitGroup
				for range 10 {
					wg.Go(func() {
						defer GinkgoRecover()
						s := newStatus("")
						s.Owner = &owner
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Set(ctx, &s)
						})).To(Succeed())
					})
				}
				wg.Wait()
				var held []status.Status[any]
				Expect(svc.NewRetrieve[any]().
					Where(status.MatchOwners[any](owner)).
					Entries(&held).
					Exec(ctx, db)).To(Succeed())
				Expect(held).To(HaveLen(1))
			},
		)
		It(
			"Should keep every label that concurrent adds write",
			func(ctx SpecContext) {
				s := newStatus("concurrent-labels-key")
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Set(ctx, &s)
				})).To(Succeed())
				labels := make([]label.Key, 10)
				for i := range labels {
					l := &label.Label{Name: fmt.Sprintf("Concurrent %d", i)}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return labelSvc.NewWriter(tx).Create(ctx, l)
					})).To(Succeed())
					labels[i] = l.Key
				}
				var wg sync.WaitGroup
				for _, l := range labels {
					wg.Go(func() {
						defer GinkgoRecover()
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).AddLabels(
								ctx,
								s.Key,
								[]label.Key{l},
							)
						})).To(Succeed())
					})
				}
				wg.Wait()
				var stored status.Status[any]
				Expect(svc.NewRetrieve[any]().
					Where(status.MatchKeys[any](s.Key)).
					Entry(&stored).
					Exec(ctx, db)).To(Succeed())
				Expect(stored.Labels).To(ConsistOf(labels))
			},
		)
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

	Describe("Changes", func() {
		It("Should return the status that a commit sets", func(ctx SpecContext) {
			var keys []status.Key
			DeferCleanup(
				db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
					for ch := range svc.Changes(ctx, c) {
						keys = append(keys, ch.Key)
					}
				}),
			)
			s := newStatus("changes-test")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Set(ctx, &s)
			})).To(Succeed())
			Expect(keys).To(Equal([]status.Key{"changes-test"}))
		})
	})

	Describe("Search", func() {
		create := func(ctx SpecContext, key, name string) resource.ID {
			GinkgoHelper()
			s := status.Status[any]{
				Key:     key,
				Name:    name,
				Variant: "success",
				Time:    telem.Now(),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Set(ctx, &s)
			})).To(Succeed())
			return status.ResourceID(s.Key)
		}

		It(
			"Should return the matching statuses, closest match first",
			func(ctx SpecContext) {
				a := create(ctx, "a", "turbopump test")
				b := create(ctx, "b", "turbopump")
				q := search.Query[resource.ID]{Term: "turbopump"}
				Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
					To(Equal([]resource.ID{b, a}))
				q.Limit = 1
				Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
					To(Equal([]resource.ID{b}))
			},
		)

		It("Should match a renamed status by its new name", func(ctx SpecContext) {
			id := create(ctx, "c", "cryogenic")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Rename(ctx, "c", "hydraulic")
			})).To(Succeed())
			q := search.Query[resource.ID]{Term: "hydraulic"}
			Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
				To(Equal([]resource.ID{id}))
			q = search.Query[resource.ID]{Term: "cryogenic"}
			Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
				To(BeEmpty())
		})
	})
})
