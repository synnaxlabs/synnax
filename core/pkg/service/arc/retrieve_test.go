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
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Retrieve", func() {
	Describe("Status Retrieve", func() {
		It("Should retrieve an Arc and its associated status", func() {
			a := arc.Arc{
				Name:  "test-retrieve",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Entry(&retrievedArc).Exec(ctx, tx)).To(Succeed())
			Expect(retrievedArc).To(Equal(a))

			var s status.Status[arc.StatusDetails]
			statusKey := a.Key.String()
			Expect(gorp.NewRetrieve[string, status.Status[arc.StatusDetails]]().
				WhereKeys(statusKey).
				Entry(&s).
				Exec(ctx, tx)).To(Succeed())

			Expect(s.Key).To(Equal(statusKey))
			Expect(s.Name).To(Equal("test-retrieve Status"))
		})

		It("Should retrieve multiple Arcs with their statuses", func() {
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

			for _, a := range retrievedArcs {
				var s status.Status[arc.StatusDetails]
				Expect(gorp.NewRetrieve[string, status.Status[arc.StatusDetails]]().
					WhereKeys(a.Key.String()).
					Entry(&s).
					Exec(ctx, tx)).To(Succeed())

				Expect(s.Key).To(Equal(a.Key.String()))
				Expect(s.Name).To(ContainSubstring(a.Name))
				Expect(s.Name).To(ContainSubstring("Status"))
			}
		})

		It("Should retrieve Arc with status after transaction commit", func() {
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

			var s status.Status[arc.StatusDetails]
			Expect(gorp.NewRetrieve[string, status.Status[arc.StatusDetails]]().
				WhereKeys(a.Key.String()).
				Entry(&s).
				Exec(ctx, newTx)).To(Succeed())

			Expect(s.Key).To(Equal(a.Key.String()))
			Expect(s.Name).To(Equal("tx-test-arc Status"))
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

			var s status.Status[arc.StatusDetails]
			Expect(gorp.NewRetrieve[string, status.Status[arc.StatusDetails]]().
				WhereKeys(a.Key.String()).
				Entry(&s).
				Exec(ctx, db)).To(Succeed())

			Expect(s.Key).To(Equal(a.Key.String()))
		})

		It("Should retrieve Arc and verify status metadata integrity", func() {
			a := arc.Arc{
				Name:  "metadata-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Entry(&retrievedArc).Exec(ctx, tx)).To(Succeed())

			var s status.Status[arc.StatusDetails]
			Expect(gorp.NewRetrieve[string, status.Status[arc.StatusDetails]]().
				WhereKeys(a.Key.String()).
				Entry(&s).
				Exec(ctx, tx)).To(Succeed())

			Expect(s.Key).To(Equal(a.Key.String()))
			Expect(s.Name).To(Equal("metadata-arc Status"))
			Expect(s.Message).To(Equal("Deploying"))
			Expect(s.Time).ToNot(BeZero())
		})
	})
})
