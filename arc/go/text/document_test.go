// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
)

var _ = Describe("Create", func() {
	It("Should build a document that materializes back to the raw text", func() {
		doc := text.Create("x * 2")
		Expect(doc.Inserts).ToNot(BeEmpty())
		t := text.Text{Doc: doc}
		Expect(t.Materialize().Raw).To(Equal("x * 2"))
	})
	It("Should build an empty document from empty text", func() {
		doc := text.Create("")
		Expect(doc.Inserts).To(BeEmpty())
		Expect(doc.Deletes).To(BeEmpty())
	})
})
