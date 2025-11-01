// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view_test

import (
	"context"
	"io"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("View", func() {
	var (
		ctx    context.Context
		otg    *ontology.Ontology
		svc    *view.Service
		w      view.Writer
		tx     gorp.Tx
		closer io.Closer
	)
	BeforeEach(func() {
		ctx = context.Background()
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{
			DB:           db,
			EnableSearch: config.True(),
		}))
		g := MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))
		svc = MustSucceed(view.OpenService(ctx, view.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		Expect(otg.InitializeSearchIndex(ctx)).To(Succeed())
		closer = xio.MultiCloser{otg, g, svc}
	})
	AfterEach(func() {
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
		Describe("Create", func() {
			It("Should create a new view with an auto-generated key", func() {
				s := &view.View{
					Name: "Test View",
					Type: "test",
				}
				Expect(w.Create(ctx, s)).To(Succeed())
				Expect(s.Key).ToNot(Equal(uuid.Nil))
			})
			It("Should update an existing view", func() {
				s := &view.View{
					Name: "Test View",
					Key:  uuid.New(),
					Type: "test",
				}
				Expect(w.Create(ctx, s)).To(Succeed())
				s.Name = "Updated Name"
				Expect(w.Create(ctx, s)).To(Succeed())
				var retrieved view.View
				Expect(svc.NewRetrieve().WhereKeys(s.Key).Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
				Expect(retrieved.Name).To(Equal("Updated Name"))
			})
		})

		Describe("CreateMany", func() {
			It("Should create multiple views", func() {
				views := []view.View{
					{
						Name: "View 1",
						Key:  uuid.New(),
						Type: "test",
					},
					{
						Name: "View 2",
						Key:  uuid.New(),
						Type: "test",
					},
				}
				Expect(w.CreateMany(ctx, &views)).To(Succeed())

				var retrieved []view.View
				Expect(svc.NewRetrieve().WhereKeys(views[0].Key, views[1].Key).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
				Expect(retrieved).To(HaveLen(2))
			})
		})

		Describe("Delete", func() {
			It("Should delete a view", func() {
				s := &view.View{
					Name: "To Delete",
					Key:  uuid.New(),
					Type: "test",
				}
				Expect(w.Create(ctx, s)).To(Succeed())
				Expect(w.Delete(ctx, s.Key)).To(Succeed())

				Expect(svc.NewRetrieve().WhereKeys(s.Key).Entry(&view.View{}).Exec(ctx, tx)).To(HaveOccurredAs(query.NotFound))
			})

			It("Should be idempotent", func() {
				Expect(w.Delete(ctx, uuid.New())).To(Succeed())
			})
		})

		Describe("DeleteMany", func() {
			It("Should delete multiple views", func() {
				views := []view.View{
					{
						Name: "Del 1",
						Key:  uuid.New(),
						Type: "test",
					},
					{
						Name: "Del 2",
						Key:  uuid.New(),
						Type: "test",
					},
				}
				Expect(w.CreateMany(ctx, &views)).To(Succeed())
				Expect(w.DeleteMany(ctx, views[0].Key, views[1].Key)).To(Succeed())

				Expect(svc.NewRetrieve().WhereKeys(views[0].Key, views[1].Key).Exec(ctx, tx)).To(HaveOccurredAs(query.NotFound))
			})
		})
	})

	Describe("Retrieve", func() {
		var views []view.View
		BeforeEach(func() {
			views = []view.View{
				{
					Name: "View A",
					Key:  uuid.New(),
					Type: "test",
				},
				{
					Name: "View B",
					Key:  uuid.New(),
					Type: "test",
				},
				{
					Name: "View C",
					Key:  uuid.New(),
					Type: "test",
				},
			}
			Expect(w.CreateMany(ctx, &views)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			tx = db.OpenTx()
			w = svc.NewWriter(tx)
		})

		Describe("WhereKeys", func() {
			It("Should retrieve view by key", func() {
				var s view.View
				Expect(svc.NewRetrieve().WhereKeys(views[0].Key).Entry(&s).Exec(ctx, tx)).To(Succeed())
				Expect(s.Key).To(Equal(views[0].Key))
				Expect(s.Name).To(Equal("View A"))
			})

			It("Should retrieve multiple views by keys", func() {
				var resViews []view.View
				Expect(svc.NewRetrieve().WhereKeys(views[0].Key, views[1].Key).Entries(&resViews).Exec(ctx, tx)).To(Succeed())
				Expect(resViews).To(HaveLen(2))
			})
		})

		Describe("Limit and Offset", func() {
			It("Should limit results", func() {
				var views []view.View
				Expect(svc.NewRetrieve().Limit(2).Entries(&views).Exec(ctx, tx)).To(Succeed())
				Expect(views).To(HaveLen(2))
			})

			It("Should offset results", func() {
				var views []view.View
				Expect(svc.NewRetrieve().Offset(1).Limit(2).Entries(&views).Exec(ctx, tx)).To(Succeed())
				Expect(views).To(HaveLen(2))
			})
		})

		Describe("Search", func() {
			It("Should search for views", func() {
				var views []view.View
				Expect(svc.NewRetrieve().Search("View A").Entries(&views).Exec(ctx, tx)).To(Succeed())
				Expect(len(views)).To(BeNumerically(">", 1))
				Expect(views[0].Key).To(Equal(views[0].Key))
			})
		})
	})
})
