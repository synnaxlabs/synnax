// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package primitives_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/primitives"
	. "github.com/synnaxlabs/x/testutil"
)

func TestPrimitives(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Primitives Suite")
}

var _ = Describe("Primitives", func() {
	Describe("IsPrimitive", func() {
		It("Should return true for registered primitives", func() {
			Expect(primitives.IsPrimitive("string")).To(BeTrue())
			Expect(primitives.IsPrimitive("uuid")).To(BeTrue())
			Expect(primitives.IsPrimitive("int32")).To(BeTrue())
			Expect(primitives.IsPrimitive("float64")).To(BeTrue())
			Expect(primitives.IsPrimitive("bytes")).To(BeTrue())
		})

		It("Should return false for non-primitives", func() {
			Expect(primitives.IsPrimitive("MyStruct")).To(BeFalse())
			Expect(primitives.IsPrimitive("CustomType")).To(BeFalse())
			Expect(primitives.IsPrimitive("")).To(BeFalse())
		})
	})
})

// NOTE: Language-specific mapping tests have been moved to per-language test files:
// - oracle/plugin/go/primitives/mapping_test.go
// - oracle/plugin/py/primitives/mapping_test.go
// - oracle/plugin/ts/primitives/mapping_test.go
// - oracle/plugin/cpp/primitives/mapping_test.go
// - oracle/plugin/pb/primitives/mapping_test.go

var _ = ShouldNotLeakGoroutinesPerSpec()
