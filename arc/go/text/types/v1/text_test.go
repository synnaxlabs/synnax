// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/text/types/v1"
	"github.com/synnaxlabs/x/crdt"
)

var _ = Describe("Text", func() {
	Describe("Materialize", func() {
		It("Should derive the raw source from the replicated document", func() {
			doc := crdt.New(v1.SeedReplica)
			doc.Insert(0, "x * 2")
			inserts, deletes := doc.Snapshot()
			t := v1.Text{Doc: v1.Document{Inserts: inserts, Deletes: deletes}}
			Expect(t.Materialize().Raw).To(Equal("x * 2"))
		})
		It("Should materialize an empty document to an empty string", func() {
			t := v1.Text{Raw: "stale"}
			Expect(t.Materialize().Raw).To(BeEmpty())
		})
	})
})
