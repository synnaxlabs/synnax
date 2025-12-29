// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Group", Ordered, func() {
	var (
		db  *gorp.DB
		svc *group.Service
		otg *ontology.Ontology
		w   group.Writer
	)

	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		svc = MustSucceed(group.OpenService(ctx, group.ServiceConfig{DB: db, Ontology: otg}))
		w = svc.NewWriter(nil)
	})

	AfterAll(func() {
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})

	Describe("Create", func() {
		It("Should create a new group", func() {
			g, err := w.Create(ctx, "test1", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())
			Expect(g.Key).ToNot(Equal(uuid.Nil))
			Expect(g.Name).To(Equal("test1"))
		})

		It("Should create a group with a specific key", func() {
			key := uuid.New()
			g, err := w.CreateWithKey(ctx, key, "test2", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())
			Expect(g.Key).To(Equal(key))
			Expect(g.Name).To(Equal("test2"))
		})

		It("Should create a nested group", func() {
			parent, err := w.Create(ctx, "parent", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			child, err := w.Create(ctx, "child", group.OntologyID(parent.Key))
			Expect(err).ToNot(HaveOccurred())
			Expect(child.Name).To(Equal("child"))
		})

	})

	Describe("Retrieve", func() {
		It("Should retrieve a group by its key", func() {
			created, err := w.Create(ctx, "retrieve-test", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			var g group.Group
			err = svc.NewRetrieve().WhereKeys(created.Key).Entry(&g).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(g).To(Equal(created))
		})

		It("Should retrieve multiple groups by keys", func() {
			g1, err := w.Create(ctx, "multi1", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			g2, err := w.Create(ctx, "multi2", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			var ret []group.Group
			err = svc.NewRetrieve().WhereKeys(g1.Key, g2.Key).Entries(&ret).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(ret).To(ConsistOf(g1, g2))
		})

		It("Should retrieve a group by its name", func() {
			created, err := w.Create(ctx, "name-test", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			var g group.Group
			err = svc.NewRetrieve().WhereNames(created.Name).Entry(&g).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(g).To(Equal(created))
		})
	})

	Describe("Rename", func() {
		It("Should rename a group", func() {
			created, err := w.Create(ctx, "original-name", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			newName := "renamed"
			Expect(w.Rename(ctx, created.Key, newName)).To(Succeed())

			var g group.Group
			err = svc.NewRetrieve().WhereKeys(created.Key).Entry(&g).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(g.Name).To(Equal(newName))
		})
	})

	Describe("Delete", func() {
		It("Should not delete a group with children", func() {
			parent, err := w.Create(ctx, "parent-to-keep", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			_, err = w.Create(ctx, "child-blocking-delete", group.OntologyID(parent.Key))
			Expect(err).ToNot(HaveOccurred())

			err = w.Delete(ctx, parent.Key)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, validate.Error)).To(BeTrue())
		})

		It("Should delete a group without children", func() {
			created, err := w.Create(ctx, "to-delete", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			Expect(w.Delete(ctx, created.Key)).To(Succeed())

			var g group.Group
			err = svc.NewRetrieve().WhereKeys(created.Key).Entry(&g).Exec(ctx, nil)
			Expect(err).To(HaveOccurred())
		})

		It("Should delete multiple groups", func() {
			parent, err := w.Create(ctx, "parent-for-deletion", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			child, err := w.Create(ctx, "child-for-deletion", group.OntologyID(parent.Key))
			Expect(err).ToNot(HaveOccurred())

			Expect(w.Delete(ctx, child.Key)).To(Succeed())
			Expect(w.Delete(ctx, parent.Key)).To(Succeed())

			var g group.Group
			err = svc.NewRetrieve().WhereKeys(parent.Key, child.Key).Entry(&g).Exec(ctx, nil)
			Expect(err).To(HaveOccurred())
		})

		It("Should allow batch deletion when parent is being deleted along with all of its children", func() {
			parent, err := w.Create(ctx, "parent-batch-delete", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			child1, err := w.Create(ctx, "child1-batch-delete", group.OntologyID(parent.Key))
			Expect(err).ToNot(HaveOccurred())

			child2, err := w.Create(ctx, "child2-batch-delete", group.OntologyID(parent.Key))
			Expect(err).ToNot(HaveOccurred())

			Expect(w.Delete(ctx, child2.Key, parent.Key, child1.Key)).To(Succeed())

			var groups []group.Group
			Expect(svc.NewRetrieve().WhereKeys(child1.Key, child2.Key, parent.Key).
				Entries(&groups).Exec(ctx, nil)).
				To(HaveOccurredAs(query.NotFound))
			Expect(groups).To(BeEmpty())
		})

		It("Should allow deleting nested hierarchy when ordered leaf to root", func() {
			root, err := w.Create(ctx, "root-nested", ontology.RootID)
			Expect(err).ToNot(HaveOccurred())

			level1, err := w.Create(ctx, "level1-nested", group.OntologyID(root.Key))
			Expect(err).ToNot(HaveOccurred())

			level2, err := w.Create(ctx, "level2-nested", group.OntologyID(level1.Key))
			Expect(err).ToNot(HaveOccurred())

			level3, err := w.Create(ctx, "level3-nested", group.OntologyID(level2.Key))
			Expect(err).ToNot(HaveOccurred())

			Expect(w.Delete(ctx, level3.Key, level2.Key, level1.Key, root.Key)).To(Succeed())

			for _, key := range []uuid.UUID{root.Key, level1.Key, level2.Key, level3.Key} {
				var g group.Group
				err = svc.NewRetrieve().WhereKeys(key).Entry(&g).Exec(ctx, nil)
				Expect(err).To(HaveOccurred())
			}
		})
	})
})
