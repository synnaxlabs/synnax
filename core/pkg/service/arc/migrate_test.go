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
	textv0 "github.com/synnaxlabs/arc/text/types/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v2"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateArc", func() {
	It("Should seed the document from the previously persisted raw text", func(ctx SpecContext) {
		old := v1.Arc{
			Name: "legacy",
			Mode: v1.ModeText,
			Text: textv0.Text{Raw: "x := 1"},
		}
		migrated := MustSucceed(v2.MigrateArc(ctx, old))
		Expect(migrated.Text.Materialize().Raw).To(Equal("x := 1"))
	})

	It("Should produce an empty document when there is no prior text", func(ctx SpecContext) {
		old := v1.Arc{Name: "empty", Mode: v1.ModeGraph}
		migrated := MustSucceed(v2.MigrateArc(ctx, old))
		Expect(migrated.Text.Materialize().Raw).To(Equal(""))
	})
})
