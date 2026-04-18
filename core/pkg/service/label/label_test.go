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
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	xlabel "github.com/synnaxlabs/x/label"
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
			l := &xlabel.Label{
				Name:  "Label",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			Expect(l.Key).ToNot(Equal(label.Key(uuid.Nil)))
		})
		It("Should create many labels", func(ctx SpecContext) {
			ls := []xlabel.Label{
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
			l := &xlabel.Label{
				Name:  "Label",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			Expect(w.Delete(ctx, l.Key)).To(Succeed())
			Expect(svc.NewRetrieve().WhereKeys(l.Key).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
		It("Should delete many labels", func(ctx SpecContext) {
			ls := []xlabel.Label{
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
			Expect(w.DeleteMany(ctx, []label.Key{ls[0].Key, ls[1].Key})).To(Succeed())
			for _, l := range ls {
				Expect(svc.NewRetrieve().WhereKeys(l.Key).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			}
		})
	})
	Describe("Retrieve", func() {
		It("Should get the labels for an ontology resource", func(ctx SpecContext) {
			l := &xlabel.Label{
				Name:  "Label",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			labeled := &xlabel.Label{
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
	Describe("RemoveLabel", func() {
		It("Should remove a label", func(ctx SpecContext) {
			l := &xlabel.Label{
				Name:  "Label",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			labeled := &xlabel.Label{
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
			Expect(labels).To(HaveLen(0))
		})
	})
	Describe("Clear", func() {
		It("Should remove all labels on an object", func(ctx SpecContext) {
			l := &xlabel.Label{
				Name:  "Label",
				Color: color.MustFromHex("#000000"),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			labeled := &xlabel.Label{
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
			Expect(labels).To(HaveLen(0))
		})
	})
})
