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
		w        status.Writer
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
				s := &status.Status{
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
				s := &status.Status{
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

				var retrieved status.Status
				Expect(svc.NewRetrieve().WhereKeys("update-key").Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
				Expect(retrieved.Message).To(Equal("Updated message"))
				Expect(retrieved.Variant).To(Equal(xstatus.Variant("warning")))
			})
			Context("Parent Management", func() {
				It("Should set a custom parent for the status", func() {
					parent := status.Status{
						Name:    "Parent Status",
						Key:     "parent-key",
						Variant: "info",
						Message: "Parent status",
						Time:    telem.Now(),
					}
					Expect(w.Set(ctx, &parent)).To(Succeed())

					child := status.Status{
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
				statuses := []status.Status{
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

				var retrieved []status.Status
				Expect(svc.NewRetrieve().WhereKeys("key1", "key2").Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
				Expect(retrieved).To(HaveLen(2))
			})
		})

		Describe("Delete", func() {
			It("Should delete a status", func() {
				s := &status.Status{
					Name:    "To Delete",
					Key:     "delete-key",
					Variant: "info",
					Message: "Will be deleted",
					Time:    telem.Now(),
				}
				Expect(w.Set(ctx, s)).To(Succeed())
				Expect(w.Delete(ctx, "delete-key")).To(Succeed())

				err := svc.NewRetrieve().WhereKeys("delete-key").Entry(&status.Status{}).Exec(ctx, tx)
				Expect(err).To(MatchError(query.NotFound))
			})

			It("Should be idempotent", func() {
				Expect(w.Delete(ctx, "non-existent-key")).To(Succeed())
			})
		})

		Describe("DeleteMany", func() {
			It("Should delete multiple statuses", func() {
				statuses := []status.Status{
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
			statuses := []status.Status{
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
			}
			Expect(w.SetMany(ctx, &statuses)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			tx = db.OpenTx()
			w = svc.NewWriter(tx)
		})

		Describe("WhereKeys", func() {
			It("Should retrieve status by key", func() {
				var s status.Status
				Expect(svc.NewRetrieve().WhereKeys("retrieve-a").Entry(&s).Exec(ctx, tx)).To(Succeed())
				Expect(s.Key).To(Equal("retrieve-a"))
				Expect(s.Name).To(Equal("Status A"))
			})

			It("Should retrieve multiple statuses by keys", func() {
				var statuses []status.Status
				Expect(svc.NewRetrieve().WhereKeys("retrieve-a", "retrieve-b").Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(2))
			})
		})

		Describe("Limit and Offset", func() {
			It("Should limit results", func() {
				var statuses []status.Status
				Expect(svc.NewRetrieve().Limit(2).Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(2))
			})

			It("Should offset results", func() {
				var statuses []status.Status
				Expect(svc.NewRetrieve().Offset(1).Limit(2).Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(statuses).To(HaveLen(2))
			})
		})

		Describe("Search", func() {
			It("Should search for statuses", func() {
				var statuses []status.Status
				Expect(svc.NewRetrieve().Search("Status A").Entries(&statuses).Exec(ctx, tx)).To(Succeed())
				Expect(len(statuses)).To(BeNumerically(">", 1))
				Expect(statuses[0].Key).To(Equal("retrieve-a"))
			})
		})
	})
})
