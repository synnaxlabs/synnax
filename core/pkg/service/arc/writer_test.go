// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create an Arc with generated key", func() {
			a := arc.Arc{
				Name:  "test-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should create an Arc with explicit key", func() {
			key := uuid.New()
			a := arc.Arc{
				Key:   key,
				Name:  "test-arc-with-key",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Key).To(Equal(key))
		})

		It("Should handle multiple arc creations", func() {
			a1 := arc.Arc{
				Name:  "arc-1",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			a2 := arc.Arc{
				Name:  "arc-2",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}

			Expect(svc.NewWriter(tx).Create(ctx, &a1)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &a2)).To(Succeed())

			Expect(a1.Key).ToNot(Equal(uuid.Nil))
			Expect(a2.Key).ToNot(Equal(uuid.Nil))
			Expect(a1.Key).ToNot(Equal(a2.Key))
		})
	})

	Describe("Update", func() {
		It("Should update an existing Arc", func() {
			key := uuid.New()
			a := arc.Arc{
				Key:   key,
				Name:  "existing-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}

			// Create the arc initially
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			// Update with same key
			a.Name = "updated-arc"
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			// Verify the update
			var retrieved arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(key).Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Name).To(Equal("updated-arc"))
		})
	})

	Describe("Delete", func() {
		It("Should delete an Arc", func() {
			a := arc.Arc{
				Name:  "arc-to-delete",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())

			var retrieved arc.Arc
			err := svc.NewRetrieve().WhereKeys(a.Key).Entry(&retrieved).Exec(ctx, tx)
			Expect(err).To(HaveOccurred())
		})

		It("Should delete multiple Arcs", func() {
			a1 := arc.Arc{
				Name:  "arc-to-delete-1",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			a2 := arc.Arc{
				Name:  "arc-to-delete-2",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &a1)).To(Succeed())
			Expect(w.Create(ctx, &a2)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, a1.Key, a2.Key)).To(Succeed())

			var retrieved arc.Arc
			err := svc.NewRetrieve().WhereKeys(a1.Key).Entry(&retrieved).Exec(ctx, tx)
			Expect(err).To(HaveOccurred())
			err = svc.NewRetrieve().WhereKeys(a2.Key).Entry(&retrieved).Exec(ctx, tx)
			Expect(err).To(HaveOccurred())
		})

		It("Should handle delete of non-existent arc gracefully", func() {
			nonExistentKey := uuid.New()
			Expect(svc.NewWriter(tx).Delete(ctx, nonExistentKey)).To(Succeed())
		})
	})
})
