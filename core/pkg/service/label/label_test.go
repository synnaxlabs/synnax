// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Label", Ordered, func() {
	var (
		db  *gorp.DB
		svc *label.Service
		w   label.Writer
		otg *ontology.Ontology
		tx  gorp.Tx
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.Open())
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		svc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Search:   searchIdx,
		}))
		Expect(searchIdx.Initialize(ctx)).To(Succeed())
	})
	BeforeEach(func(ctx SpecContext) {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should create a new label", func(ctx SpecContext) {
			l := &label.Label{
				Name:  "Label",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			Expect(l.Key).ToNot(Equal(label.Key(uuid.Nil)))
		})
		It("Should return a validation error when the name is empty", func(ctx SpecContext) {
			Expect(w.Create(ctx, &label.Label{})).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should create many labels", func(ctx SpecContext) {
			ls := []label.Label{
				{
					Name:  "Label1",
					Color: color.MustFromHex("#000000"),
				},
				{
					Name:  "label",
					Color: color.MustFromHex("#000000"),
				},
			}
			Expect(w.CreateMany(ctx, &ls)).To(Succeed())
			for _, l := range ls {
				Expect(l.Key).ToNot(Equal(label.Key(uuid.Nil)))
			}
		})
	})
	Describe("Delete", func() {
		It("Should delete a label", func(ctx SpecContext) {
			l := &label.Label{
				Name:  "Label",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			Expect(w.Delete(ctx, l.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(label.MatchKeys(l.Key)).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
		It("Should delete many labels", func(ctx SpecContext) {
			ls := []label.Label{
				{
					Name:  "Label1",
					Color: color.MustFromHex("#000000"),
				},
				{
					Name:  "label",
					Color: color.MustFromHex("#000000"),
				},
			}
			Expect(w.CreateMany(ctx, &ls)).To(Succeed())
			Expect(w.Delete(ctx, ls[0].Key, ls[1].Key)).To(Succeed())
			for _, l := range ls {
				Expect(svc.NewRetrieve().Where(label.MatchKeys(l.Key)).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			}
		})
	})
	Describe("Retrieve", func() {
		It("Should get the labels for an ontology resource", func(ctx SpecContext) {
			l := &label.Label{
				Name:  "Label",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			labeled := &label.Label{
				Name:  "Labeled",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, labeled)).To(Succeed())
			Expect(w.Label(ctx, label.OntologyID(labeled.Key), []label.Key{l.Key})).To(Succeed())
			labels := MustSucceed(svc.RetrieveFor(ctx, label.OntologyID(labeled.Key), tx))
			Expect(labels).To(HaveLen(1))
			Expect(labels[0].Key).To(Equal(l.Key))
		})
	})
	Describe("MatchNames", func() {
		It("Should retrieve a label by its name", func(ctx SpecContext) {
			l := &label.Label{Name: "match-name-target", Color: color.MustFromHex("#000000")}
			Expect(w.Create(ctx, l)).To(Succeed())
			var got label.Label
			Expect(svc.NewRetrieve().
				Where(label.MatchNames("match-name-target")).
				Entry(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got.Key).To(Equal(l.Key))
		})
		It("Should retrieve labels matching any of the provided names", func(ctx SpecContext) {
			ls := []label.Label{
				{Name: "mn-a", Color: color.MustFromHex("#000000")},
				{Name: "mn-b", Color: color.MustFromHex("#000000")},
				{Name: "mn-c", Color: color.MustFromHex("#000000")},
			}
			Expect(w.CreateMany(ctx, &ls)).To(Succeed())
			var got []label.Label
			Expect(svc.NewRetrieve().
				Where(label.MatchNames("mn-a", "mn-b")).
				Entries(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(2))
		})
	})
	Describe("Limit and Offset", func() {
		BeforeEach(func(ctx SpecContext) {
			ls := []label.Label{
				{Name: "page-a", Color: color.MustFromHex("#000000")},
				{Name: "page-b", Color: color.MustFromHex("#000000")},
				{Name: "page-c", Color: color.MustFromHex("#000000")},
			}
			Expect(w.CreateMany(ctx, &ls)).To(Succeed())
		})
		It("Should limit the number of returned labels", func(ctx SpecContext) {
			var got []label.Label
			Expect(svc.NewRetrieve().Limit(2).Entries(&got).Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(2))
		})
		It("Should offset the returned labels", func(ctx SpecContext) {
			var all []label.Label
			Expect(svc.NewRetrieve().Entries(&all).Exec(ctx, tx)).To(Succeed())
			Expect(len(all)).To(BeNumerically(">=", 3))
			var got []label.Label
			Expect(svc.NewRetrieve().Offset(1).Entries(&got).Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(len(all) - 1))
		})
	})
	Describe("Search", func() {
		It("Should fuzzy search labels by name", func(ctx SpecContext) {
			l := &label.Label{Name: "Searchable Critical Label", Color: color.MustFromHex("#000000")}
			Expect(w.Create(ctx, l)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			tx = db.OpenTx()
			w = svc.NewWriter(tx)
			var got []label.Label
			Expect(svc.NewRetrieve().
				Search("Searchable Critical").
				Entries(&got).
				Exec(ctx, nil)).To(Succeed())
			Expect(got).ToNot(BeEmpty())
			Expect(got[0].Name).To(ContainSubstring("Searchable"))
		})
	})
	Describe("RemoveLabel", func() {
		It("Should remove a label", func(ctx SpecContext) {
			l := &label.Label{
				Name:  "Label",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			labeled := &label.Label{
				Name:  "Labeled",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, labeled)).To(Succeed())
			Expect(w.Label(ctx, label.OntologyID(labeled.Key), []label.Key{l.Key})).To(Succeed())
			labels := MustSucceed(svc.RetrieveFor(ctx, label.OntologyID(labeled.Key), tx))
			Expect(labels).To(HaveLen(1))
			Expect(labels[0].Key).To(Equal(l.Key))
			Expect(w.RemoveLabel(ctx, label.OntologyID(labeled.Key), []label.Key{l.Key})).To(Succeed())
			labels = MustSucceed(svc.RetrieveFor(ctx, label.OntologyID(labeled.Key), tx))
			Expect(labels).To(BeEmpty())
		})
	})
	Describe("Clear", func() {
		It("Should remove all labels on an object", func(ctx SpecContext) {
			l := &label.Label{
				Name:  "Label",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			labeled := &label.Label{
				Name:  "Labeled",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, labeled)).To(Succeed())
			Expect(w.Label(ctx, label.OntologyID(labeled.Key), []label.Key{l.Key})).To(Succeed())
			labels := MustSucceed(svc.RetrieveFor(ctx, label.OntologyID(labeled.Key), tx))
			Expect(labels).To(HaveLen(1))
			Expect(labels[0].Key).To(Equal(l.Key))
			Expect(w.Clear(ctx, label.OntologyID(labeled.Key))).To(Succeed())
			labels = MustSucceed(svc.RetrieveFor(ctx, label.OntologyID(labeled.Key), tx))
			Expect(labels).To(BeEmpty())
		})
	})
})
