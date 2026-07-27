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
	v0 "github.com/synnaxlabs/arc/types/versions/v0"
	v1 "github.com/synnaxlabs/arc/types/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
)

// migrateType lifts a v0 type through MigrateParam, the package's exported
// migration surface.
func migrateType(ctx SpecContext, old v0.Type) v1.Type {
	migrated := MustSucceed(v1.MigrateParam(ctx, v0.Param{Name: "p", Type: old}))
	return migrated.Type
}

var _ = Describe("MigrateParam", func() {
	It("Should carry the name, type, and value", func(ctx SpecContext) {
		migrated := MustSucceed(v1.MigrateParam(ctx, v0.Param{
			Name:  "rate",
			Type:  v0.Type{Kind: v0.KindF64},
			Value: "100",
		}))
		Expect(migrated.Name).To(Equal("rate"))
		Expect(migrated.Type.Kind).To(Equal(v1.KindF64))
		Expect(migrated.Value).To(Equal("100"))
	})

	It("Should carry the type's kind, name, and unit", func(ctx SpecContext) {
		migrated := migrateType(ctx, v0.Type{
			Kind: v0.KindF64,
			Name: "f64",
			Unit: new(v0.Unit{Scale: 1, Name: "psi"}),
		})
		Expect(migrated.Kind).To(Equal(v1.KindF64))
		Expect(migrated.Name).To(Equal("f64"))
		Expect(migrated.Unit.Name).To(Equal("psi"))
	})

	It("Should migrate the element type of compound types", func(ctx SpecContext) {
		migrated := migrateType(ctx, v0.Type{
			Kind: v0.KindChan,
			Elem: new(v0.Type{Kind: v0.KindF32}),
		})
		Expect(migrated.Elem.Kind).To(Equal(v1.KindF32))
	})

	It("Should carry function inputs and outputs while dropping the removed config", func(ctx SpecContext) {
		migrated := migrateType(ctx, v0.Type{
			FunctionProperties: v0.FunctionProperties{
				Inputs:  v0.Params{{Name: "in"}},
				Outputs: v0.Params{{Name: "out"}},
				Config:  v0.Params{{Name: "cfg"}},
			},
			Kind: v0.KindFunction,
		})
		Expect(migrated.Inputs).To(HaveLen(1))
		Expect(migrated.Inputs[0].Name).To(Equal("in"))
		Expect(migrated.Outputs).To(HaveLen(1))
		Expect(migrated.Outputs[0].Name).To(Equal("out"))
	})
})
