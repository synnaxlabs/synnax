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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Collab", func() {
	// dispatch generates the local edits as collaborative actions and dispatches them
	// through the service, the way a client does.
	toActions := func(inserts []crdt.Insert) []arc.Action {
		out := make([]arc.Action, len(inserts))
		for i, op := range inserts {
			out[i] = arc.NewInsertCharAction(arc.InsertCharPayload{Op: op})
		}
		return out
	}

	It("Should integrate dispatched operations into the authoritative document", func(ctx SpecContext) {
		a := &arc.Arc{Name: "collab-empty", Mode: arc.ModeText}
		Expect(svc.NewWriter(nil).Create(ctx, a)).To(Succeed())

		client := crdt.New(2)
		Expect(svc.NewWriter(nil).Dispatch(ctx, a.Key, "dk", toActions(client.Insert(0, "hello")))).
			To(Succeed())

		inserts, deletes := MustSucceed2(svc.Snapshot(ctx, a.Key))
		fresh := crdt.New(3)
		fresh.Load(inserts, deletes)
		Expect(fresh.String()).To(Equal("hello"))
	})

	It("Should bootstrap a joiner into the document and integrate its edits", func(ctx SpecContext) {
		a := &arc.Arc{Name: "collab-seeded", Mode: arc.ModeText}
		a.Text.Raw = "base"
		Expect(svc.NewWriter(nil).Create(ctx, a)).To(Succeed())

		clientA := crdt.New(2)
		clientA.Load(MustSucceed2(svc.Snapshot(ctx, a.Key)))
		Expect(clientA.String()).To(Equal("base"))

		Expect(svc.NewWriter(nil).Dispatch(ctx, a.Key, "dk", toActions(clientA.Insert(0, "X")))).
			To(Succeed())

		clientB := crdt.New(3)
		clientB.Load(MustSucceed2(svc.Snapshot(ctx, a.Key)))
		Expect(clientB.String()).To(Equal("Xbase"))
	})
})
