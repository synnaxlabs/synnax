// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/includes"
	"github.com/synnaxlabs/oracle/plugin/cpp/types"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("FieldProcessor", func() {
	var (
		loader *MockFileLoader
		mgr    *includes.Manager
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		mgr = includes.NewManager()
	})

	process := func(ctx SpecContext, source, field string) types.FieldData {
		GinkgoHelper()
		req := MustGenerateRequest(ctx, source, "test", loader)
		item := req.Resolutions.MustGet("test.Item")
		f := MustBeOk(item.Form.(resolution.StructForm).Field(field))
		fp := &types.FieldProcessor{
			Includes:  mgr,
			Namespace: "test",
			Table:     req.Resolutions,
		}
		return fp.ProcessField(f, item)
	}

	It("Should declare a required field with its zero value", func(ctx SpecContext) {
		fd := process(ctx, `
			@cpp output "client/cpp/test"

			Item struct {
				count int32 { @doc value "is the count." }
			}
		`, "count")
		Expect(fd.Name).To(Equal("count"))
		Expect(fd.CppType).To(Equal("std::int32_t"))
		Expect(fd.DefaultValue).To(Equal("0"))
		Expect(fd.Doc).To(Equal("is the count."))
		Expect(mgr.SystemIncludes()).To(ContainElement("cstdint"))
	})

	It("Should wrap an optional field in std::optional", func(ctx SpecContext) {
		fd := process(ctx, `
			@cpp output "client/cpp/test"

			Item struct {
				after string?
			}
		`, "after")
		Expect(fd.CppType).To(Equal("std::optional<std::string>"))
		Expect(mgr.SystemIncludes()).To(ContainElement("optional"))
	})

	It("Should name a type of the namespace by its bare name", func(ctx SpecContext) {
		fd := process(ctx, `
			@cpp output "client/cpp/test"

			Tile struct {
				key string
			}

			Item struct {
				tiles Tile[]
			}
		`, "tiles")
		Expect(fd.CppType).To(Equal("std::vector<Tile>"))
		Expect(mgr.InternalIncludes()).To(BeEmpty())
	})

	It("Should qualify a type of another namespace and include its header", func(
		ctx SpecContext,
	) {
		loader.Add("schemas/patch", `
			@cpp output "x/cpp/patch"

			Guard struct {
				value string
			}
		`)
		fd := process(ctx, `
			import "schemas/patch"

			@cpp output "client/cpp/test"

			Item struct {
				guards patch.Guard[] = []
			}
		`, "guards")
		Expect(fd.CppType).To(Equal("std::vector<::x::patch::Guard>"))
		Expect(fd.DefaultValue).To(Equal("{}"))
		Expect(mgr.InternalIncludes()).To(ContainElement("x/cpp/patch/types.gen.h"))
	})
})
