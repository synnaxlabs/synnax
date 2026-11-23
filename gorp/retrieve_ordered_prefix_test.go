// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Retrieve with a prefix and an order", func() {
	It("Should keep only the keys under the prefix", func(ctx SpecContext) {
		db := DeferClose(wrapMsgpack(memkv.New()))
		StartRun(db.Run)
		labelIdx := gorp.NewSortedIndex(
			"label", func(e *relationEntry) string { return e.Label },
		)
		table := DeferClose(openIndexedTable(ctx, db, labelIdx))
		entries := []relationEntry{
			{From: "a", To: "1", Label: "z"},
			{From: "a", To: "2", Label: "y"},
			{From: "b", To: "1", Label: "x"},
		}
		Expect(table.NewCreate().Entries(&entries).Exec(ctx, db)).To(Succeed())
		var res []relationEntry
		Expect(table.NewRetrieve().
			WherePrefix([]byte("a->")).
			OrderBy(labelIdx.Ordered(gorp.DirectionAsc)).
			Entries(&res).Exec(ctx, db)).To(Succeed())
		Expect(fromsOf(res)).To(Equal([]string{"a", "a"}))
	})
})
