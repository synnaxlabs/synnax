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
	"github.com/synnaxlabs/oracle/plugin/ts/types"
)

func TestTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TS Types Plugin Suite")
}

// MockFileLoader is a file loader that serves files from memory.
type MockFileLoader struct {
	Files map[string]string
}

func (m *MockFileLoader) Load(importPath string) (string, string, error) {
	if content, ok := m.Files[importPath]; ok {
		return content, importPath + ".oracle", nil
	}
	if content, ok := m.Files[importPath+".oracle"]; ok {
		return content, importPath + ".oracle", nil
	}
	return "", "", &fileNotFoundError{path: importPath}
}

func (m *MockFileLoader) RepoRoot() string {
	return "/mock/repo"
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

var _ = Describe("TS Types Plugin", func() {
	var (
		ctx         context.Context
		loader      *MockFileLoader
		typesPlugin *types.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = &MockFileLoader{Files: make(map[string]string)}
		typesPlugin = types.New(types.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(typesPlugin.Name()).To(Equal("ts/types"))
		})

		It("Should have no domain filter", func() {
			Expect(typesPlugin.Domains()).To(BeEmpty())
		})
	})

	Describe("Generate", func() {
		It("Should generate schema for simple struct", func() {
			source := `
				struct User {
					field key uuid
					field name string
					field age int32
					field active bool
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`import { z } from "zod"`))
			Expect(content).To(ContainSubstring(`import { zod } from "@synnaxlabs/x"`))
			Expect(content).To(ContainSubstring(`export const userZ = z.object(`))
			Expect(content).To(ContainSubstring(`key: z.uuid()`))
			Expect(content).To(ContainSubstring(`name: z.string()`))
			Expect(content).To(ContainSubstring(`age: zod.int32Z`))
			Expect(content).To(ContainSubstring(`active: z.boolean()`))
			Expect(content).To(ContainSubstring(`export type User = z.infer<typeof userZ>`))
		})

		It("Should handle optional and array types", func() {
			source := `
				struct Range {
					field key uuid
					field labels uuid[]
					field parent uuid?
					field tags string[]?
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`labels: z.array(z.uuid())`))
			Expect(content).To(ContainSubstring(`parent: z.uuid().optional()`))
			Expect(content).To(ContainSubstring(`tags: z.array(z.string()).optional()`))
		})

		It("Should apply validation rules", func() {
			source := `
				struct User {
					field name string {
						domain validate {
							min_length 1
							max_length 255
						}
					}
					field email string {
						domain validate {
							email
						}
					}
					field age int32 {
						domain validate {
							min 0
							max 150
						}
					}
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`name: z.string().min(1).max(255)`))
			Expect(content).To(ContainSubstring(`email: z.string().email()`))
			Expect(content).To(ContainSubstring(`age: zod.int32Z.min(0).max(150)`))
		})

		It("Should generate enums", func() {
			source := `
				enum TaskState {
					pending = 0
					running = 1
					completed = 2
				}

				struct Task {
					field state TaskState
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`export const taskStateZ = z.enum([`))
			Expect(content).To(ContainSubstring(`"pending"`))
			Expect(content).To(ContainSubstring(`"running"`))
			Expect(content).To(ContainSubstring(`"completed"`))
			Expect(content).To(ContainSubstring(`state: taskStateZ`))
		})

		It("Should generate string enums", func() {
			source := `
				enum DataType {
					float32 = "float32"
					float64 = "float64"
					int32 = "int32"
				}

				struct Telem {
					field data_type DataType
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`export const dataTypeZ = z.enum(["float32", "float64", "int32"])`))
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
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`a: z.uuid()`))
			Expect(content).To(ContainSubstring(`b: z.string()`))
			Expect(content).To(ContainSubstring(`c: z.boolean()`))
			Expect(content).To(ContainSubstring(`d: zod.int8Z`))
			Expect(content).To(ContainSubstring(`e: zod.int16Z`))
			Expect(content).To(ContainSubstring(`f: zod.int32Z`))
			Expect(content).To(ContainSubstring(`g: zod.int64Z`))
			Expect(content).To(ContainSubstring(`h: zod.uint8Z`))
			Expect(content).To(ContainSubstring(`i: zod.uint16Z`))
			Expect(content).To(ContainSubstring(`j: zod.uint32Z`))
			Expect(content).To(ContainSubstring(`k: zod.uint64Z`))
			Expect(content).To(ContainSubstring(`l: zod.float32Z`))
			Expect(content).To(ContainSubstring(`m: zod.float64Z`))
			Expect(content).To(ContainSubstring(`n: TimeStamp.z`))
			Expect(content).To(ContainSubstring(`o: TimeSpan.z`))
			Expect(content).To(ContainSubstring(`p: z.record(z.string(), z.unknown())`))
			Expect(content).To(ContainSubstring(`q: z.instanceof(Uint8Array)`))
			Expect(content).To(ContainSubstring(`import { TimeSpan, TimeStamp, zod } from "@synnaxlabs/x"`))
		})

		It("Should convert snake_case to camelCase for field names", func() {
			source := `
				struct Range {
					field created_at timestamp
					field time_range string
					field my_long_field_name string
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`createdAt:`))
			Expect(content).To(ContainSubstring(`timeRange:`))
			Expect(content).To(ContainSubstring(`myLongFieldName:`))
		})

		It("Should generate create request struct with optional key and password", func() {
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
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`export const newZ = z.object({`))
			Expect(content).To(ContainSubstring(`key: z.uuid().optional()`))
			Expect(content).To(ContainSubstring(`username: z.string()`))
			Expect(content).To(ContainSubstring(`password: z.string().min(1)`))
			Expect(content).To(ContainSubstring(`firstName: z.string().optional()`))
			Expect(content).To(ContainSubstring(`lastName: z.string().optional()`))
			Expect(content).To(ContainSubstring(`export type New = z.infer<typeof newZ>`))
		})

		It("Should handle nullable scalar types", func() {
			source := `
				struct Device {
					field key uuid
					field name string
					field status string!
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`status: z.string().nullable()`))
		})

		It("Should handle nullish types (optional + nullable)", func() {
			source := `
				struct Task {
					field key uuid
					field name string
					field status string?!
					field description string!?
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`status: z.string().nullish()`))
			Expect(content).To(ContainSubstring(`description: z.string().nullish()`))
		})

		It("Should handle nullable arrays with array.nullableZ", func() {
			source := `
				struct Policy {
					field key uuid
					field objects uuid[]!
					field actions string[]!
					domain ts { output "out" }
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
			Expect(content).To(ContainSubstring(`import { array } from "@synnaxlabs/x"`))
			Expect(content).To(ContainSubstring(`objects: array.nullableZ(z.uuid())`))
			Expect(content).To(ContainSubstring(`actions: array.nullableZ(z.string())`))
		})

		It("Should handle nullable optional arrays", func() {
			source := `
				struct Channel {
					field key uuid
					field operations string[]?!
					domain ts { output "out" }
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "channel", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`operations: array.nullableZ(z.string()).optional()`))
		})
	})
})
