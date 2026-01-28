// Copyright 2026 Synnax Labs, Inc.
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

var _ = Describe("Retrieve", func() {
	Describe("Arc Retrieve", func() {
		It("Should retrieve an Arc", func() {
			a := arc.Arc{
				Name:  "test-retrieve",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Entry(&retrievedArc).Exec(ctx, tx)).To(Succeed())
			Expect(retrievedArc).To(Equal(a))
		})

		It("Should retrieve multiple Arcs", func() {
			arcs := []arc.Arc{
				{Name: "arc-multi-1", Graph: graph.Graph{}, Text: text.Text{}},
				{Name: "arc-multi-2", Graph: graph.Graph{}, Text: text.Text{}},
				{Name: "arc-multi-3", Graph: graph.Graph{}, Text: text.Text{}},
			}

			keys := make([]uuid.UUID, 0, len(arcs))
			for i := range arcs {
				Expect(svc.NewWriter(tx).Create(ctx, &arcs[i])).To(Succeed())
				keys = append(keys, arcs[i].Key)
			}

			var retrievedArcs []arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(keys...).Entries(&retrievedArcs).Exec(ctx, tx)).To(Succeed())
			Expect(retrievedArcs).To(HaveLen(3))
		})

		It("Should retrieve Arc after transaction commit", func() {
			localTx := db.OpenTx()
			a := arc.Arc{
				Name:  "tx-test-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(localTx).Create(ctx, &a)).To(Succeed())
			Expect(localTx.Commit(ctx)).To(Succeed())

			newTx := db.OpenTx()

			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Entry(&retrievedArc).Exec(ctx, newTx)).To(Succeed())
			Expect(retrievedArc.Name).To(Equal("tx-test-arc"))
			Expect(newTx.Close()).To(Succeed())
		})

		It("Should retrieve Arc without transaction", func() {
			localTx := db.OpenTx()
			a := arc.Arc{
				Name:  "no-tx-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(localTx).Create(ctx, &a)).To(Succeed())
			Expect(localTx.Commit(ctx)).To(Succeed())

			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Entry(&retrievedArc).Exec(ctx, nil)).To(Succeed())
			Expect(retrievedArc.Name).To(Equal("no-tx-arc"))
		})
	})
})
