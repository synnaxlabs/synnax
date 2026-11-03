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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
)

var _ = Describe("Retrieve", func() {
	Describe("Arc Retrieve", func() {
		It("Should retrieve an Arc", func(ctx SpecContext) {
			a := newAutomation(arc.Base{
				Name:  "test-retrieve",
				Mode:  arc.ModeText,
				Graph: graph.Graph{},
				Text:  text.Text{},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			var retrievedArc arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Entry(&retrievedArc).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(retrievedArc.Base().Key).To(Equal(a.Base().Key))
			Expect(retrievedArc.Base().Name).To(Equal(a.Base().Name))
		})

		It("Should retrieve multiple Arcs", func(ctx SpecContext) {
			arcs := []arc.Arc{
				newAutomation(arc.Base{
					Name:  "arc-multi-1",
					Mode:  arc.ModeText,
					Graph: graph.Graph{},
					Text:  text.Text{},
				}),
				newAutomation(arc.Base{
					Name:  "arc-multi-2",
					Mode:  arc.ModeText,
					Graph: graph.Graph{},
					Text:  text.Text{},
				}),
				newAutomation(arc.Base{
					Name:  "arc-multi-3",
					Mode:  arc.ModeText,
					Graph: graph.Graph{},
					Text:  text.Text{},
				}),
			}

			keys := make([]uuid.UUID, 0, len(arcs))
			for i := range arcs {
				Expect(svc.NewWriter(tx).Create(ctx, &arcs[i])).To(Succeed())
				keys = append(keys, arcs[i].Base().Key)
			}

			var retrievedArcs []arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(keys...)).
					Entries(&retrievedArcs).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(retrievedArcs).To(HaveLen(3))
		})

		It("Should retrieve Arc after transaction commit", func(ctx SpecContext) {
			localTx := db.OpenTx()
			a := newAutomation(arc.Base{
				Name:  "tx-test-arc",
				Mode:  arc.ModeText,
				Graph: graph.Graph{},
				Text:  text.Text{},
			})
			Expect(svc.NewWriter(localTx).Create(ctx, &a)).To(Succeed())
			Expect(localTx.Commit(ctx)).To(Succeed())

			newTx := db.OpenTx()

			var retrievedArc arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Entry(&retrievedArc).
					Exec(ctx, newTx),
			).To(Succeed())
			Expect(retrievedArc.Base().Name).To(Equal("tx-test-arc"))
			Expect(newTx.Close()).To(Succeed())
		})

		It("Should retrieve Arc without transaction", func(ctx SpecContext) {
			localTx := db.OpenTx()
			a := newAutomation(arc.Base{
				Name:  "no-tx-arc",
				Mode:  arc.ModeText,
				Graph: graph.Graph{},
				Text:  text.Text{},
			})
			Expect(svc.NewWriter(localTx).Create(ctx, &a)).To(Succeed())
			Expect(localTx.Commit(ctx)).To(Succeed())

			var retrievedArc arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Entry(&retrievedArc).
					Exec(ctx, nil),
			).To(Succeed())
			Expect(retrievedArc.Base().Name).To(Equal("no-tx-arc"))
		})
	})
})
