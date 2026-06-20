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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/crdt"
)

var _ = Describe("Collab", func() {
	// toActions turns a client's local insert operations into the collaborative-edit
	// actions a client dispatches through the service.
	toActions := func(inserts []crdt.Insert) []arc.Action {
		out := make([]arc.Action, len(inserts))
		for i, op := range inserts {
			out[i] = arc.NewInsertCharAction(arc.InsertCharPayload{
				ID:     op.ID,
				Origin: op.Origin,
				Side:   op.Side,
				Char:   op.Char,
			})
		}
		return out
	}

	materialized := func(ctx SpecContext, key arc.Key) string {
		var got arc.Arc
		Expect(svc.NewRetrieve().Where(arc.MatchKeys(key)).Entry(&got).Exec(ctx, nil)).
			To(Succeed())
		return got.Text.Materialize().Raw
	}

	It("Should materialize dispatched insertions into the arc's text", func(ctx SpecContext) {
		a := &arc.Arc{Name: "collab-empty", Mode: arc.ModeText}
		Expect(svc.NewWriter(nil).Create(ctx, a)).To(Succeed())

		client := crdt.New(2)
		Expect(svc.NewWriter(nil).Dispatch(ctx, a.Key, "dk", toActions(client.Insert(0, "hello")))).
			To(Succeed())

		Expect(materialized(ctx, a.Key)).To(Equal("hello"))
	})

	It("Should seed the document from raw on create and materialize a bootstrapped edit", func(ctx SpecContext) {
		a := &arc.Arc{Name: "collab-seeded", Mode: arc.ModeText}
		a.Text.Raw = "base"
		Expect(svc.NewWriter(nil).Create(ctx, a)).To(Succeed())

		var seeded arc.Arc
		Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Entry(&seeded).Exec(ctx, nil)).
			To(Succeed())
		client := crdt.New(2)
		client.Load(seeded.Text.Doc.Inserts, seeded.Text.Doc.Deletes)
		Expect(client.String()).To(Equal("base"))

		Expect(svc.NewWriter(nil).Dispatch(ctx, a.Key, "dk", toActions(client.Insert(0, "X")))).
			To(Succeed())

		Expect(materialized(ctx, a.Key)).To(Equal("Xbase"))
	})
})
