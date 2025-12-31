// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"context"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/py/types"
	"github.com/synnaxlabs/oracle/testutil"
)

func TestTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Python Types Plugin Suite")
}

var _ = Describe("Python Types Plugin", func() {
	var (
		ctx         context.Context
		loader      *testutil.MockFileLoader
		typesPlugin *types.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = testutil.NewMockFileLoader()
		typesPlugin = types.New(types.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(typesPlugin.Name()).To(Equal("py/types"))
		})

		It("Should have no domain filter", func() {
			Expect(typesPlugin.Domains()).To(BeEmpty())
		})
	})

	Describe("Generate", func() {
		It("Should generate Pydantic model for simple struct", func() {
			source := `
				@py output "out"

				User struct {
					key uuid
					name string
					age int32
					active bool
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`from __future__ import annotations`))
			Expect(content).To(ContainSubstring(`from uuid import UUID`))
			Expect(content).To(ContainSubstring(`from pydantic import BaseModel`))
			Expect(content).To(ContainSubstring(`class User(BaseModel):`))
			Expect(content).To(ContainSubstring(`key: UUID`))
			Expect(content).To(ContainSubstring(`name: str`))
			Expect(content).To(ContainSubstring(`age: int`))
			Expect(content).To(ContainSubstring(`active: bool`))
		})

		It("Should handle optional and array types", func() {
			source := `
				@py output "out"

				Range struct {
					key uuid
					labels uuid[]
					parent uuid?
					tags string[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`labels: list[UUID]`))
			Expect(content).To(ContainSubstring(`parent: UUID | None = None`))
			Expect(content).To(ContainSubstring(`tags: list[str] | None = None`))
		})

		It("Should apply validation rules with Field constraints", func() {
			source := `
				@py output "out"

				User struct {
					name string @validate {
						min_length 1
						max_length 255
					}
					age int32 @validate {
						min 0
						max 150
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`from pydantic import BaseModel, Field`))
			Expect(content).To(ContainSubstring(`name: str = Field(min_length=1, max_length=255)`))
			Expect(content).To(ContainSubstring(`age: int = Field(ge=0, le=150)`))
		})

		It("Should generate IntEnum for integer enums", func() {
			source := `
				@py output "out"

				TaskState enum {
					pending = 0
					running = 1
					completed = 2
				}

				Task struct {
					state TaskState
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`from enum import IntEnum`))
			Expect(content).To(ContainSubstring(`class TaskState(IntEnum):`))
			Expect(content).To(ContainSubstring(`pending = 0`))
			Expect(content).To(ContainSubstring(`running = 1`))
			Expect(content).To(ContainSubstring(`completed = 2`))
			Expect(content).To(ContainSubstring(`state: TaskState`))
		})

		It("Should generate Literal type for string enums", func() {
			source := `
				@py output "out"

				DataType enum {
					float32 = "float32"
					float64 = "float64"
					int32 = "int32"
				}

				Telem struct {
					data_type DataType
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "telem", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`from typing import Literal`))
			Expect(content).To(ContainSubstring(`DataType = Literal["float32", "float64", "int32"]`))
			Expect(content).To(ContainSubstring(`data_type: DataType`))
		})

		It("Should handle primitive type mappings", func() {
			source := `
				@py output "out"

				AllTypes struct {
					a uuid
					b string
					c bool
					d int8
					e int16
					f int32
					g int64
					h uint8
					i uint16
					j uint32
					k uint64
					l float32
					m float64
					n timestamp
					o timespan
					p json
					q bytes
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`a: UUID`))
			Expect(content).To(ContainSubstring(`b: str`))
			Expect(content).To(ContainSubstring(`c: bool`))
			Expect(content).To(ContainSubstring(`d: int`))
			Expect(content).To(ContainSubstring(`e: int`))
			Expect(content).To(ContainSubstring(`f: int`))
			Expect(content).To(ContainSubstring(`g: int`))
			Expect(content).To(ContainSubstring(`h: int`))
			Expect(content).To(ContainSubstring(`i: int`))
			Expect(content).To(ContainSubstring(`j: int`))
			Expect(content).To(ContainSubstring(`k: int`))
			Expect(content).To(ContainSubstring(`l: float`))
			Expect(content).To(ContainSubstring(`m: float`))
			Expect(content).To(ContainSubstring(`n: TimeStamp`))
			Expect(content).To(ContainSubstring(`o: TimeSpan`))
			Expect(content).To(ContainSubstring(`p: dict[str, Any]`))
			Expect(content).To(ContainSubstring(`q: bytes`))
			Expect(content).To(ContainSubstring(`from uuid import UUID`))
			Expect(content).To(ContainSubstring(`from typing import Any`))
			Expect(content).To(ContainSubstring(`from synnax.telem import TimeSpan, TimeStamp`))
		})

		It("Should keep snake_case for field names (Python convention)", func() {
			source := `
				@py output "out"

				Range struct {
					created_at timestamp
					time_range string
					my_long_field_name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`created_at:`))
			Expect(content).To(ContainSubstring(`time_range:`))
			Expect(content).To(ContainSubstring(`my_long_field_name:`))
		})

		It("Should generate type alias for key fields", func() {
			source := `
				@py output "out"

				User struct {
					key uuid @key
					username string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`Key = UUID`))
		})

		It("Should handle optional key and password fields", func() {
			source := `
				@py output "out"

				New struct {
					key uuid?
					username string
					password string @validate min_length 1
					first_name string?
					last_name string?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`class New(BaseModel):`))
			Expect(content).To(ContainSubstring(`key: UUID | None = None`))
			Expect(content).To(ContainSubstring(`username: str`))
			Expect(content).To(ContainSubstring(`password: str = Field(min_length=1)`))
			Expect(content).To(ContainSubstring(`first_name: str | None = None`))
			Expect(content).To(ContainSubstring(`last_name: str | None = None`))
		})

		It("Should handle soft optional types (?)", func() {
			source := `
				@py output "out"

				Device struct {
					key uuid
					name string
					status string?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "device", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Soft optional (?) becomes T | None = None in Python
			Expect(content).To(ContainSubstring(`status: str | None = None`))
		})

		It("Should handle hard optional types (??)", func() {
			source := `
				@py output "out"

				Task struct {
					key uuid
					name string
					status string??
					description string??
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Hard optional (??) also becomes T | None = None in Python (no pointer distinction)
			Expect(content).To(ContainSubstring(`status: str | None = None`))
			Expect(content).To(ContainSubstring(`description: str | None = None`))
		})

		It("Should handle optional arrays", func() {
			source := `
				@py output "out"

				Policy struct {
					key uuid
					objects uuid[]?
					actions string[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "policy", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Optional arrays in Python use None default (not default_factory)
			// to distinguish "not provided" from "empty list"
			Expect(content).To(ContainSubstring(`objects: list[UUID] | None = None`))
			Expect(content).To(ContainSubstring(`actions: list[str] | None = None`))
		})

		It("Should handle default values", func() {
			source := `
				@py output "out"

				Config struct {
					enabled bool @validate default false
					retries int32 @validate default 3
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "config", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`enabled: bool = Field(default=False)`))
			Expect(content).To(ContainSubstring(`retries: int = Field(default=3)`))
		})

		It("Should generate class inheritance for basic struct extension", func() {
			source := `
				@py output "out"

				Parent struct {
					name string
					age int32
				}

				Child struct extends Parent {
					email string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Parent should be a regular class
			Expect(content).To(ContainSubstring(`class Parent(BaseModel):`))
			Expect(content).To(ContainSubstring(`name: str`))
			Expect(content).To(ContainSubstring(`age: int`))

			// Child should inherit from Parent
			Expect(content).To(ContainSubstring(`class Child(Parent):`))
			Expect(content).To(ContainSubstring(`email: str`))
		})

		It("Should generate ConfigDict for field omissions", func() {
			source := `
				@py output "out"

				Parent struct {
					name string
					age int32
					status string
				}

				Child struct extends Parent {
					-age
					email string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Child should have ConfigDict import
			Expect(content).To(ContainSubstring(`from pydantic import`))
			Expect(content).To(ContainSubstring(`ConfigDict`))

			// Child should inherit and use model_config
			Expect(content).To(ContainSubstring(`class Child(Parent):`))
			Expect(content).To(ContainSubstring(`email: str`))
			Expect(content).To(ContainSubstring(`model_config = ConfigDict(`))
			Expect(content).To(ContainSubstring(`"age": {"exclude": True}`))
		})

		It("Should generate ConfigDict for multiple field omissions", func() {
			source := `
				@py output "out"

				Parent struct {
					a string
					b string
					c string
					d string
				}

				Child struct extends Parent {
					-a
					-c
					e string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`class Child(Parent):`))
			Expect(content).To(ContainSubstring(`model_config = ConfigDict(`))
			Expect(content).To(ContainSubstring(`"a": {"exclude": True}`))
			Expect(content).To(ContainSubstring(`"c": {"exclude": True}`))
		})

		It("Should handle field override to make it optional", func() {
			source := `
				@py output "out"

				Parent struct {
					name string
					age int32
				}

				Child struct extends Parent {
					name string?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Child should override name to be optional
			Expect(content).To(ContainSubstring(`class Child(Parent):`))
			Expect(content).To(ContainSubstring(`name: str | None = None`))
		})

		It("Should handle extension without new fields (only omissions)", func() {
			source := `
				@py output "out"

				Parent struct {
					a string
					b string
					c string
				}

				Child struct extends Parent {
					-b
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`class Child(Parent):`))
			Expect(content).To(ContainSubstring(`model_config = ConfigDict(`))
			Expect(content).To(ContainSubstring(`"b": {"exclude": True}`))
		})

		It("Should handle extension of generic struct with type arguments", func() {
			source := `
				@py output "out"

				Details struct {
					message string
				}

				Status struct<D extends schema> {
					variant int32
					data D
				}

				RackStatus struct extends Status<Details> {
					timestamp timestamp
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// RackStatus should extend Status (generic type args are inherited via extends)
			Expect(content).To(ContainSubstring(`class RackStatus(Status):`))
			Expect(content).To(ContainSubstring(`timestamp: TimeStamp`))
		})

		It("Should preserve struct declaration order", func() {
			source := `
				@py output "out"

				Zebra struct {
					name string
				}

				Apple struct {
					color string
				}

				Mango struct {
					ripe bool
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "animals", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			zebraIdx := strings.Index(content, "class Zebra")
			appleIdx := strings.Index(content, "class Apple")
			mangoIdx := strings.Index(content, "class Mango")
			Expect(zebraIdx).To(BeNumerically("<", appleIdx))
			Expect(appleIdx).To(BeNumerically("<", mangoIdx))
		})

		It("Should preserve field declaration order", func() {
			source := `
				@py output "out"

				Record struct {
					zebra string
					apple int32
					mango bool
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "order", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			zebraIdx := strings.Index(content, "zebra:")
			appleIdx := strings.Index(content, "apple:")
			mangoIdx := strings.Index(content, "mango:")
			Expect(zebraIdx).To(BeNumerically("<", appleIdx))
			Expect(appleIdx).To(BeNumerically("<", mangoIdx))
		})
	})
})
