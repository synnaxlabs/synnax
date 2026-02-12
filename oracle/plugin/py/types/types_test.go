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
	"context"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/py/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

func TestTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Python Types Plugin Suite")
}

var _ = Describe("PyFormatter", func() {
	f := types.PyFormatter

	Describe("FormatQualified", func() {
		It("Should format qualified names with dot separator", func() {
			Expect(f.FormatQualified("pkg", "Type")).To(Equal("pkg.Type"))
		})

		It("Should return type name when qualifier is empty", func() {
			Expect(f.FormatQualified("", "Type")).To(Equal("Type"))
		})
	})

	Describe("FormatGeneric", func() {
		It("Should format generic types with square brackets", func() {
			Expect(f.FormatGeneric("Container", []string{"T", "U"})).To(Equal("Container[T, U]"))
		})

		It("Should return base name when no type args", func() {
			Expect(f.FormatGeneric("Container", nil)).To(Equal("Container"))
		})
	})

	Describe("FormatArray", func() {
		It("Should format as list type", func() {
			Expect(f.FormatArray("str")).To(Equal("list[str]"))
		})
	})

	Describe("FormatMap", func() {
		It("Should format as dict type", func() {
			Expect(f.FormatMap("str", "int")).To(Equal("dict[str, int]"))
		})
	})

	Describe("FallbackType", func() {
		It("Should return Any", func() {
			Expect(f.FallbackType()).To(Equal("Any"))
		})
	})
})

var _ = Describe("Python Types Plugin", func() {
	var (
		ctx         context.Context
		loader      *MockFileLoader
		typesPlugin *types.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = NewMockFileLoader()
		typesPlugin = types.New(types.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(typesPlugin.Name()).To(Equal("py/types"))
		})

		It("Should have no domain filter", func() {
			Expect(typesPlugin.Domains()).To(BeEmpty())
		})

		It("Should have no dependencies", func() {
			Expect(typesPlugin.Requires()).To(BeNil())
		})

		It("Should pass check", func() {
			Expect(typesPlugin.Check(&plugin.Request{})).To(BeNil())
		})
	})

	Describe("Generate", func() {
		Context("basic struct generation", func() {
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
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "types_gen.py").
					ToContain(
						`from __future__ import annotations`,
						`from uuid import UUID`,
						`from pydantic import BaseModel`,
						`class User(BaseModel):`,
						`key: UUID`,
						`name: str`,
						`age: int`,
						`active: bool`,
					)
			})
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`from typing import Literal`))
			Expect(content).To(ContainSubstring(`DataType = Literal["float32", "float64", "int32"]`))
			Expect(content).To(ContainSubstring(`data_type: DataType`))
		})

		Context("primitive type mappings", func() {
			DescribeTable("should generate correct Python type",
				func(oracleType, expectedPyType string) {
					source := `
						@py output "out"

						Test struct {
							field ` + oracleType + `
						}
					`
					resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
					ExpectContent(resp, "types_gen.py").ToContain("field: " + expectedPyType)
				},
				Entry("uuid", "uuid", "UUID"),
				Entry("string", "string", "str"),
				Entry("bool", "bool", "bool"),
				Entry("int8", "int8", "int"),
				Entry("int16", "int16", "int"),
				Entry("int32", "int32", "int"),
				Entry("int64", "int64", "int"),
				Entry("uint8", "uint8", "int"),
				Entry("uint16", "uint16", "int"),
				Entry("uint32", "uint32", "int"),
				Entry("uint64", "uint64", "int"),
				Entry("float32", "float32", "float"),
				Entry("float64", "float64", "float"),
				Entry("json", "json", "dict[str, Any]"),
				Entry("bytes", "bytes", "bytes"),
			)

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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Child should override name to be optional
			Expect(content).To(ContainSubstring(`class Child(Parent):`))
			Expect(content).To(ContainSubstring(`name: str | None = None`))
		})
		It("Should generate multiple inheritance for multiple extends", func() {
			source := `
				@py output "out"

				A struct {
					a string
				}

				B struct {
					b int32
				}

				C struct extends A, B {
					c bool
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// C should inherit from both A and B using Python multiple inheritance
			Expect(content).To(ContainSubstring(`class C(A, B):`))
			Expect(content).To(ContainSubstring(`c: bool`))
		})

		It("Should handle field omission with multiple extends", func() {
			source := `
				@py output "out"

				A struct {
					a string
					shared string
				}

				B struct {
					b int32
				}

				C struct extends A, B {
					-shared
					c bool
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// C should inherit from both A and B
			Expect(content).To(ContainSubstring(`class C(A, B):`))
			Expect(content).To(ContainSubstring(`c: bool`))
			// shared field should not be present in C (omitted)
			// Note: Python inheritance doesn't explicitly omit, but the field shouldn't be redefined
		})

		It("Should handle three extends with multiple inheritance", func() {
			source := `
				@py output "out"

				A struct {
					a string
				}

				B struct {
					b int32
				}

				D struct {
					d bool
				}

				C struct extends A, B, D {
					c float32
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// C should inherit from all three parents
			Expect(content).To(ContainSubstring(`class C(A, B, D):`))
			Expect(content).To(ContainSubstring(`c: float`))
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
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

		Context("@omit directive", func() {
			It("Should skip types with @py omit directive", func() {
				source := `
					@py output "out"

					User struct {
						key uuid
						name string
					}

					InternalState struct {
						cache json
						@py omit
					}
				`
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`class User(BaseModel):`))
				Expect(content).NotTo(ContainSubstring(`InternalState`))
			})

			It("Should skip enums with @py omit directive", func() {
				source := `
					@py output "out"

					Status enum {
						active = 1
						inactive = 2
					}

					DebugLevel enum {
						verbose = 0
						trace = 1
						@py omit
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`class Status(IntEnum):`))
				Expect(content).NotTo(ContainSubstring(`DebugLevel`))
			})
		})

		Context("type aliases", func() {
			It("Should generate TypeAlias for simple type alias", func() {
				source := `
					@py output "out"

					UserID = uuid
				`
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`from typing import TypeAlias`))
				Expect(content).To(ContainSubstring(`UserID: TypeAlias = UUID`))
			})

			It("Should generate NewType for distinct type", func() {
				source := `
					@py output "out"

					UserKey uuid
				`
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`from typing import NewType`))
				Expect(content).To(ContainSubstring(`UserKey = NewType("UserKey", UUID)`))
			})
		})

		Context("field omission in extensions", func() {
			It("Should handle field omission with minus prefix", func() {
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
				resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				// Parent should have all fields
				Expect(content).To(ContainSubstring(`class Parent(BaseModel):`))
				// Child inherits from Parent but should set age to None
				Expect(content).To(ContainSubstring(`class Child(Parent):`))
				Expect(content).To(ContainSubstring(`email: str`))
			})

			It("Should handle multiple field omissions", func() {
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
				resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`class Child(Parent):`))
				Expect(content).To(ContainSubstring(`e: str`))
			})
		})

		Context("documentation", func() {
			It("Should generate docstrings and comments from doc domain", func() {
				source := `
					@py output "out"

					User struct {
						@doc value "is a representation of a user in the system."

						key uuid @key {
							@doc value "is the unique identifier for the user."
						}

						name string {
							@doc value "is the user's display name."
						}

						age int32
					}
				`
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`"""Is a representation of a user in the system.`))
				Expect(content).To(ContainSubstring(`Attributes:`))
				Expect(content).To(ContainSubstring(`key: Is the unique identifier for the user.`))
				Expect(content).To(ContainSubstring(`name: Is the user's display name.`))
			})
		})

		Context("generic structs", func() {
			It("Should generate Generic[T] for struct with type parameter", func() {
				source := `
					@py output "out"

					Response struct<T> {
						data T
						status int32
					}
				`
				resp := MustGenerate(ctx, source, "api", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(
						`from typing import Generic`,
						`TypeVar`,
						`T = TypeVar("T")`,
						`class Response(BaseModel, Generic[T]):`,
						`data: T`,
						`status: int`,
					)
			})

			It("Should generate constrained TypeVar for bounded type param", func() {
				source := `
					@py output "out"

					Container struct<V extends json> {
						value V
					}
				`
				resp := MustGenerate(ctx, source, "api", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(
						`V = TypeVar("V", bound=dict[str, Any])`,
						`class Container(BaseModel, Generic[V]):`,
						`value: V`,
					)
			})

			It("Should skip type params with defaults in Generic[]", func() {
				source := `
					@py output "out"

					Task struct<C extends json = json> {
						config C
					}
				`
				resp := MustGenerate(ctx, source, "api", loader, typesPlugin)
				content := MustContentOf(resp, "types_gen.py")
				Expect(content).To(ContainSubstring(`class Task(BaseModel):`))
				Expect(content).To(ContainSubstring(`config: dict[str, Any]`))
			})

			It("Should generate extends with generic parent and type args", func() {
				source := `
					@py output "out"

					Status struct<D> {
						data D
						variant string
					}

					TaskStatus struct extends Status<json> {
						task_key uuid
					}
				`
				resp := MustGenerate(ctx, source, "api", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(
						`class Status(BaseModel, Generic[D]):`,
						`class TaskStatus(Status[dict[str, Any]]):`,
						`task_key: UUID`,
					)
			})
		})

		Context("cross-namespace references", func() {
			BeforeEach(func() {
				loader.Add("schemas/status", `
					@py output "client/py/synnax/status"

					StatusCode enum {
						ok = 0
						error = 1
					}

					StatusInfo struct {
						code StatusCode
						message string
					}
				`)
			})

			It("Should import cross-namespace struct reference", func() {
				source := `
					import "schemas/status"

					@py output "client/py/synnax/task"

					Task struct {
						key uuid
						info status.StatusInfo
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(
						`from synnax import status`,
						`info: status.StatusInfo`,
					)
			})

			It("Should import cross-namespace enum reference", func() {
				source := `
					import "schemas/status"

					@py output "client/py/synnax/task"

					Task struct {
						key uuid
						code status.StatusCode
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(
						`from synnax import status`,
						`code: status.StatusCode`,
					)
			})

			It("Should alias module import when it conflicts with a field name", func() {
				source := `
					import "schemas/status"

					@py output "client/py/synnax/task"

					Task struct {
						key uuid
						status status.StatusInfo
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(
						`from synnax import status as status_`,
						`status: status_.StatusInfo`,
					)
			})
		})

		Context("typedef with non-primitive base", func() {
			It("Should generate NewType for distinct typedef referencing another typedef", func() {
				source := `
					@py output "out"

					BaseID uuid

					UserID = BaseID
				`
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(
						`BaseID = NewType("BaseID", UUID)`,
						`UserID: TypeAlias = BaseID`,
					)
			})

			It("Should generate NewType referencing another distinct type in same namespace", func() {
				source := `
					@py output "out"

					Key uuid

					UserKey = Key
				`
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(
						`Key = NewType("Key", UUID)`,
						`UserKey: TypeAlias = Key`,
					)
			})
		})

		Context("fixed-size arrays", func() {
			It("Should generate Tuple type for fixed-size arrays", func() {
				source := `
					@py output "out"

					Point struct {
						coords float64[3]
					}
				`
				resp := MustGenerate(ctx, source, "geo", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(`Tuple[float, float, float]`)
			})
		})

		Context("@py name directive", func() {
			It("Should override struct name in Python output", func() {
				source := `
					@py output "out"

					GoTask struct {
						key uuid
						name string
						@py name "Task"
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(`class Task(BaseModel):`).
					ToNotContain(`class GoTask`)
			})
		})

		Context("string default values", func() {
			It("Should generate Field with string default", func() {
				source := `
					@py output "out"

					Config struct {
						mode string @validate default "normal"
					}
				`
				resp := MustGenerate(ctx, source, "config", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(`mode: str = Field(default="normal")`)
			})
		})

		Context("constrained type param with enum", func() {
			It("Should use enum name as TypeVar bound", func() {
				source := `
					@py output "out"

					Priority enum {
						low = 0
						medium = 1
						high = 2
					}

					Ranked struct<P extends Priority> {
						priority P
						name string
					}
				`
				resp := MustGenerate(ctx, source, "api", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(
						`P = TypeVar("P", bound=Priority)`,
						`class Ranked(BaseModel, Generic[P]):`,
					)
			})
		})

		Context("constrained type param with struct", func() {
			It("Should use struct name as TypeVar bound", func() {
				source := `
					@py output "out"

					Base struct {
						id uuid
					}

					Collection struct<T extends Base> {
						items T[]
					}
				`
				resp := MustGenerate(ctx, source, "api", loader, typesPlugin)
				ExpectContent(resp, "types_gen.py").
					ToContain(
						`T = TypeVar("T", bound=Base)`,
						`class Collection(BaseModel, Generic[T]):`,
					)
			})
		})
	})
})
