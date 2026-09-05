// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var _ = Describe("Batch wrapper", func() {
	var r wazero.Runtime
	BeforeEach(func(ctx SpecContext) {
		r = wazero.NewRuntime(ctx)
		DeferCleanup(func(ctx SpecContext) {
			Expect(r.Close(ctx)).To(Succeed())
		})
	})

	instantiate := func(
		ctx SpecContext,
		source, key string,
	) (api.Module, api.Function) {
		GinkgoHelper()
		output := MustSucceed(compile(ctx, source, nil))
		mod := MustSucceed(r.Instantiate(ctx, output.WASM))
		return mod, mod.ExportedFunction(key + compiler.BatchSuffix)
	}

	It("Should compute a whole series in one call", func(ctx SpecContext) {
		mod, batch := instantiate(ctx, `
		func scale(x f64, k f64) f64 {
			return x * k
		}
		`, "scale")
		Expect(batch).ToNot(BeNil())
		var (
			mem    = mod.Memory()
			out    = uint32(0x8000)
			xs     = uint32(0x8100)
			k      = uint32(0x8200)
			values = []float64{1, 2, 3, 4}
		)
		for i, v := range values {
			Expect(mem.WriteFloat64Le(xs+uint32(i)*8, v)).To(BeTrue())
		}
		Expect(mem.WriteFloat64Le(k, 2.5)).To(BeTrue())
		// A stride of zero holds k fixed across every sample.
		MustSucceed(batch.Call(ctx, uint64(len(values)), uint64(out),
			uint64(xs), 8, uint64(k), 0))
		for i, v := range values {
			Expect(MustBeOk(mem.ReadFloat64Le(out + uint32(i)*8))).To(Equal(v * 2.5))
		}
	})

	It("Should write nothing when the count is zero", func(ctx SpecContext) {
		mod, batch := instantiate(ctx, `
		func double(x i64) i64 {
			return x * 2
		}
		`, "double")
		Expect(batch).ToNot(BeNil())
		mem := mod.Memory()
		Expect(mem.WriteUint64Le(0x8000, math.MaxUint64)).To(BeTrue())
		MustSucceed(batch.Call(ctx, 0, 0x8000, 0x8100, 8))
		Expect(MustBeOk(mem.ReadUint64Le(0x8000))).To(Equal(uint64(math.MaxUint64)))
	})

	It("Should truncate narrow samples to their own width", func(ctx SpecContext) {
		mod, batch := instantiate(ctx, `
		func offset(x u8) u8 {
			return x + 1
		}
		`, "offset")
		Expect(batch).ToNot(BeNil())
		mem := mod.Memory()
		Expect(mem.Write(0x8100, []byte{7, 254, 255})).To(BeTrue())
		MustSucceed(batch.Call(ctx, 3, 0x8000, 0x8100, 1))
		Expect(MustBeOk(mem.Read(0x8000, 3))).To(Equal([]byte{8, 255, 0}))
	})

	DescribeTable("Should emit no wrapper for a non-vectorizable signature",
		func(ctx SpecContext, source, key string) {
			output := MustSucceed(compile(ctx, source, nil))
			compiled := MustSucceed(r.CompileModule(ctx, output.WASM))
			DeferCleanup(compiled.Close)
			Expect(compiled.ExportedFunctions()).
				ToNot(HaveKey(key + compiler.BatchSuffix))
		},
		Entry("string return", `
		func label(x i64) str {
			return "hello"
		}
		`, "label"),
		Entry("string parameter", `
		func size(s str) i64 {
			return 1
		}
		`, "size"),
		Entry("series parameter", `
		func total(s series i64) i64 {
			return len(s)
		}
		`, "total"),
	)
})
