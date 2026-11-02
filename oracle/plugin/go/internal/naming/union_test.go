// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package naming_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("UnionTagMethod", func() {
	It("Should name the tag method after the discriminator", func() {
		Expect(MustBeOk(naming.UnionTagMethod(
			resolution.UnionForm{Discriminator: "type"},
		))).To(Equal("Type"))
	})

	It("Should report a discriminator that collides with the Variant field", func() {
		_, ok := naming.UnionTagMethod(resolution.UnionForm{Discriminator: "variant"})
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("UnionBases", func() {
	It("Should name a method for each shared base except one named Variant", func(
		ctx SpecContext,
	) {
		req := MustGenerateRequest(ctx, `
			@go output "out"

			Base struct {
				name string
			}
			Variant struct {
				size int32
			}
			Shape union on type extends Base, Variant {
				circle { radius float64 }
			}
		`, "test", NewMockFileLoader())
		shape := req.Resolutions.MustGet("test.Shape")
		bases := naming.UnionBases(shape.Form.(resolution.UnionForm), req.Resolutions)
		Expect(lo.Map(bases, func(b naming.UnionBase, _ int) string {
			return b.Method
		})).To(Equal([]string{"Base"}))
	})
})
