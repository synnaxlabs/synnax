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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	Describe("Arc Retrieve", func() {
		It("Should retrieve an Arc", func(ctx SpecContext) {
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{
				Name:  "test-retrieve",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}))

			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Entry(&retrievedArc).Exec(ctx, tx)).To(Succeed())
			Expect(retrievedArc.Key).To(Equal(a.Key))
			Expect(retrievedArc.Name).To(Equal(a.Name))
		})

		It("Should retrieve multiple Arcs", func(ctx SpecContext) {
			news := []arc.New{
				{Name: "arc-multi-1", Graph: graph.Graph{}, Text: text.Text{}},
				{Name: "arc-multi-2", Graph: graph.Graph{}, Text: text.Text{}},
				{Name: "arc-multi-3", Graph: graph.Graph{}, Text: text.Text{}},
			}

			keys := make([]uuid.UUID, 0, len(news))
			for _, n := range news {
				a := MustSucceed(svc.NewWriter(tx).Create(ctx, n))
				keys = append(keys, a.Key)
			}

			var retrievedArcs []arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(keys...)).Entries(&retrievedArcs).Exec(ctx, tx)).To(Succeed())
			Expect(retrievedArcs).To(HaveLen(3))
		})

		It("Should retrieve Arc after transaction commit", func(ctx SpecContext) {
			localTx := db.OpenTx()
			a := MustSucceed(svc.NewWriter(localTx).Create(ctx, arc.New{
				Name:  "tx-test-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}))
			Expect(localTx.Commit(ctx)).To(Succeed())

			newTx := db.OpenTx()

			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Entry(&retrievedArc).Exec(ctx, newTx)).To(Succeed())
			Expect(retrievedArc.Name).To(Equal("tx-test-arc"))
			Expect(newTx.Close()).To(Succeed())
		})

		It("Should retrieve Arc without transaction", func(ctx SpecContext) {
			localTx := db.OpenTx()
			a := MustSucceed(svc.NewWriter(localTx).Create(ctx, arc.New{
				Name:  "no-tx-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}))
			Expect(localTx.Commit(ctx)).To(Succeed())

			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Entry(&retrievedArc).Exec(ctx, nil)).To(Succeed())
			Expect(retrievedArc.Name).To(Equal("no-tx-arc"))
		})
	})
})
