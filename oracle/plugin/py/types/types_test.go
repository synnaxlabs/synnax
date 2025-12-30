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
				struct User {
					field key uuid
					field name string
					field age int32
					field active bool
					domain py { output "out" }
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
				struct Range {
					field key uuid
					field labels uuid[]
					field parent uuid?
					field tags string[]?
					domain py { output "out" }
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
				struct User {
					field name string {
						domain validate {
							min_length 1
							max_length 255
						}
					}
					field age int32 {
						domain validate {
							min 0
							max 150
						}
					}
					domain py { output "out" }
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
				enum TaskState {
					pending = 0
					running = 1
					completed = 2
				}

				struct Task {
					field state TaskState
					domain py { output "out" }
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
				enum DataType {
					float32 = "float32"
					float64 = "float64"
					int32 = "int32"
				}

				struct Telem {
					field data_type DataType
					domain py { output "out" }
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
				struct AllTypes {
					field a uuid
					field b string
					field c bool
					field d int8
					field e int16
					field f int32
					field g int64
					field h uint8
					field i uint16
					field j uint32
					field k uint64
					field l float32
					field m float64
					field n timestamp
					field o timespan
					field p json
					field q bytes
					domain py { output "out" }
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
				struct Range {
					field created_at timestamp
					field time_range string
					field my_long_field_name string
					domain py { output "out" }
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

		It("Should generate type alias for ID fields", func() {
			source := `
				struct User {
					field key uuid {
						domain id
					}
					field username string
					domain py { output "out" }
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
				struct New {
					field key uuid?
					field username string
					field password string {
						domain validate {
							min_length 1
						}
					}
					field first_name string?
					field last_name string?
					domain py { output "out" }
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

		It("Should handle nullable scalar types", func() {
			source := `
				struct Device {
					field key uuid
					field name string
					field status string!
					domain py { output "out" }
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
			Expect(content).To(ContainSubstring(`status: str | None`))
			// Should NOT have " = None" since it's nullable but not optional
			Expect(content).NotTo(ContainSubstring(`status: str | None = None`))
		})

		It("Should handle nullish types (optional + nullable)", func() {
			source := `
				struct Task {
					field key uuid
					field name string
					field status string?!
					field description string!?
					domain py { output "out" }
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
			Expect(content).To(ContainSubstring(`status: str | None = None`))
			Expect(content).To(ContainSubstring(`description: str | None = None`))
		})

		It("Should handle nullable arrays with default factory", func() {
			source := `
				struct Policy {
					field key uuid
					field objects uuid[]!
					field actions string[]!
					domain py { output "out" }
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
			Expect(content).To(ContainSubstring(`from pydantic import BaseModel, Field`))
			Expect(content).To(ContainSubstring(`objects: list[UUID] | None = Field(default_factory=list)`))
			Expect(content).To(ContainSubstring(`actions: list[str] | None = Field(default_factory=list)`))
		})

		It("Should handle default values", func() {
			source := `
				struct Config {
					field enabled bool {
						domain validate {
							default false
						}
					}
					field retries int32 {
						domain validate {
							default 3
						}
					}
					domain py { output "out" }
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
	})
})
