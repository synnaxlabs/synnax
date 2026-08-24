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
	v0 "github.com/synnaxlabs/arc/ir/versions/v0"
	v1 "github.com/synnaxlabs/arc/ir/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateFunction", func() {
	It("Should carry a Function's fields to the next version", func(ctx SpecContext) {
		migrated := MustSucceed(v1.MigrateFunction(ctx, v0.Function{
			Key:  "scale",
			Body: v0.Body{Raw: "x * 2"},
		}))
		Expect(migrated.Key).To(Equal("scale"))
		Expect(migrated.Body.Raw).To(Equal("x * 2"))
	})
})
