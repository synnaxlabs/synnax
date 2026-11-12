// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/includes"
	"github.com/synnaxlabs/oracle/plugin/cpp/json"
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

	process := func(ctx SpecContext, source, field string) json.FieldData {
		GinkgoHelper()
		req := MustGenerateRequest(ctx, source, "test", loader)
		item := req.Resolutions.MustGet("test.Item")
		f := MustBeOk(item.Form.(resolution.StructForm).Field(field))
		fp := &json.FieldProcessor{
			Includes:   mgr,
			Namespace:  "test",
			OutputPath: "client/cpp/test",
			Table:      req.Resolutions,
			JSONPaths:  MustSucceed(json.ContentPaths(req)),
		}
		return fp.ProcessField(f, item)
	}

	It("Should read and write a required field", func(ctx SpecContext) {
		fd := process(ctx, `
			@cpp output "client/cpp/test"

			Item struct {
				key string
			}
		`, "key")
		Expect(fd.Name).To(Equal("key"))
		Expect(fd.ParseExpr).To(Equal(`parser.field<std::string>("key")`))
		Expect(fd.ToJSONExpr).To(Equal(`j["key"] = this->key;`))
	})

	It("Should write an optional field only when it holds a value", func(
		ctx SpecContext,
	) {
		fd := process(ctx, `
			@cpp output "client/cpp/test"

			Item struct {
				after string?
			}
		`, "after")
		Expect(fd.ParseExpr).To(
			Equal(`parser.field<std::optional<std::string>>("after")`),
		)
		Expect(fd.ToJSONExpr).To(
			Equal(`if (this->after.has_value()) j["after"] = *this->after;`),
		)
	})

	It("Should qualify the codec of a union in the namespace", func(ctx SpecContext) {
		fd := process(ctx, `
			@cpp output "client/cpp/test"

			Shape union on type {
				circle {
					radius float64
				}
				square {}
			}

			Item struct {
				shape Shape
			}
		`, "shape")
		Expect(fd.ToJSONExpr).To(
			Equal(`j["shape"] = ::synnax::test::to_json(this->shape);`),
		)
	})

	It("Should include the JSON header of a type in another namespace", func(
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
				guard patch.Guard
			}
		`, "guard")
		Expect(fd.ParseExpr).To(Equal(`parser.field<::x::patch::Guard>("guard")`))
		Expect(fd.ToJSONExpr).To(Equal(`j["guard"] = this->guard.to_json();`))
		Expect(mgr.InternalIncludes()).To(ContainElement("x/cpp/patch/json.gen.h"))
	})
})
