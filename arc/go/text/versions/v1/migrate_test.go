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
	v0 "github.com/synnaxlabs/arc/text/versions/v0"
	v1 "github.com/synnaxlabs/arc/text/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateText", func() {
	It(
		"Should seed the replicated document from the raw source",
		func(ctx SpecContext) {
			migrated := MustSucceed(v1.MigrateText(ctx, v0.Text{Raw: "x := 1"}))
			Expect(migrated.Materialize().Raw).To(Equal("x := 1"))
		},
	)

	It("Should leave the derived raw text empty", func(ctx SpecContext) {
		migrated := MustSucceed(v1.MigrateText(ctx, v0.Text{Raw: "x := 1"}))
		Expect(migrated.Raw).To(BeEmpty())
	})

	It(
		"Should produce an empty document when there is no raw text",
		func(ctx SpecContext) {
			migrated := MustSucceed(v1.MigrateText(ctx, v0.Text{}))
			Expect(migrated.Materialize().Raw).To(BeEmpty())
		},
	)
})
