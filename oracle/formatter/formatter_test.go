// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package formatter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/formatter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Format", func() {
	format := func(source string) string {
		return MustSucceed(formatter.Format(source))
	}

	Describe("Empty and Minimal Schemas", func() {
		It("should format an empty schema to empty string", func() {
			Expect(format("")).To(Equal(""))
		})

		It("should return source unchanged on syntax error", func() {
			source := "this is not valid {{{ oracle"
			Expect(format(source)).To(Equal(source))
		})
	})

	Describe("Imports", func() {
		It("should format a single import", func() {
			Expect(format("import \"common.oracle\"\n")).To(Equal("import \"common.oracle\"\n"))
		})

		It("should format multiple imports", func() {
			result := format("import \"a.oracle\"\nimport \"b.oracle\"\n")
			Expect(result).To(ContainSubstring("import \"a.oracle\""))
			Expect(result).To(ContainSubstring("import \"b.oracle\""))
		})
	})

	Describe("Struct Definitions", func() {
		It("should format an empty struct", func() {
			Expect(format("User struct {}\n")).To(Equal("User struct {}\n"))
		})

		It("should format a struct with fields", func() {
			result := format("User struct {\n  name string\n  age int32\n}\n")
			Expect(result).To(ContainSubstring("name"))
			Expect(result).To(ContainSubstring("string"))
			Expect(result).To(ContainSubstring("age"))
			Expect(result).To(ContainSubstring("int32"))
		})

		It("should format a struct with extends", func() {
			result := format("Child struct extends Parent {\n  name string\n}\n")
			Expect(result).To(ContainSubstring("extends Parent"))
		})

		It("should align field names and types", func() {
			result := format("User struct {\n  x int32\n  longName string\n}\n")
			Expect(result).To(ContainSubstring("x        int32"))
			Expect(result).To(ContainSubstring("longName string"))
		})

		It("should format a struct with multiple inheritance", func() {
			source := "A struct {\n  a string\n}\n\nB struct {\n  b string\n}\n\nC struct extends A, B {\n  c string\n}\n"
			result := format(source)
			Expect(result).To(ContainSubstring("extends A, B"))
		})
	})

	Describe("Field Omissions", func() {
		It("should format field omissions in struct aliases", func() {
			source := "Parent struct {\n  name string\n  age int32\n}\n\nChild struct extends Parent {\n  -age\n  email string\n}\n"
			result := format(source)
			Expect(result).To(ContainSubstring("-age"))
			Expect(result).To(ContainSubstring("email string"))
		})
	})

	Describe("Enum Definitions", func() {
		It("should format an empty enum", func() {
			Expect(format("Status enum {}\n")).To(Equal("Status enum {}\n"))
		})

		It("should format an enum with string values", func() {
			result := format("Status enum {\n  ACTIVE = \"active\"\n  INACTIVE = \"inactive\"\n}\n")
			Expect(result).To(ContainSubstring("ACTIVE"))
			Expect(result).To(ContainSubstring("\"active\""))
		})

		It("should align enum values", func() {
			result := format("Status enum {\n  A = 1\n  LONG_NAME = 2\n}\n")
			Expect(result).To(ContainSubstring("A         = 1"))
			Expect(result).To(ContainSubstring("LONG_NAME = 2"))
		})

		It("should format an enum with domains", func() {
			source := "Status enum {\n  ACTIVE = 1\n  INACTIVE = 2\n\n  @go output \"core/pkg/status\"\n}\n"
			result := format(source)
			Expect(result).To(ContainSubstring("ACTIVE"))
			Expect(result).To(ContainSubstring("@go output"))
		})
	})

	Describe("Domains", func() {
		It("should format file-level domains", func() {
			result := format("@go output \"core/pkg/user\"\n\nUser struct {}\n")
			Expect(result).To(ContainSubstring("@go output \"core/pkg/user\""))
		})

		It("should format multiple file-level domains with alignment", func() {
			source := "@ts output \"client/ts/src/range\"\n@go output \"core/pkg/service/range\"\n\nRange struct {}\n"
			result := format(source)
			Expect(result).To(ContainSubstring("@ts output"))
			Expect(result).To(ContainSubstring("@go output"))
		})

		It("should format struct-level domains", func() {
			result := format("User struct {\n  name string\n\n  @go output \"core/pkg/user\"\n}\n")
			Expect(result).To(ContainSubstring("@go output"))
		})

		It("should format inline field domains", func() {
			result := format("User struct {\n  name string @validate required\n}\n")
			Expect(result).To(ContainSubstring("@validate required"))
		})

		It("should format struct-level domain blocks with multiple expressions", func() {
			source := "User struct {\n  name string\n\n  @validate {\n    required\n    max_length 255\n  }\n}\n"
			result := format(source)
			Expect(result).To(ContainSubstring("required"))
			Expect(result).To(ContainSubstring("max_length 255"))
		})

		It("should format field body with brace-form domains", func() {
			source := "User struct {\n  name string {\n    @validate required\n    @doc description \"The name\"\n  }\n}\n"
			result := format(source)
			Expect(result).To(ContainSubstring("@validate"))
			Expect(result).To(ContainSubstring("@doc"))
		})

		It("should format domain without content", func() {
			source := "User struct {\n  name string @validate\n}\n"
			result := format(source)
			Expect(result).To(ContainSubstring("@validate"))
		})
	})

	Describe("Type Definitions", func() {
		It("should format a simple typedef", func() {
			Expect(format("UserID uuid\n")).To(Equal("UserID uuid\n"))
		})

		It("should format a typedef with domains", func() {
			source := "ChannelKey uint32 {\n  @go output \"core/pkg/channel\"\n  @ts output \"client/ts/src/channel\"\n}\n"
			result := format(source)
			Expect(result).To(ContainSubstring("ChannelKey uint32"))
			Expect(result).To(ContainSubstring("@go output"))
			Expect(result).To(ContainSubstring("@ts output"))
		})
	})

	Describe("Struct Aliases", func() {
		It("should format a struct alias", func() {
			result := format("UserAlias = User\n")
			Expect(result).To(ContainSubstring("UserAlias = User"))
		})

		It("should format a struct alias with domains", func() {
			source := "MyStatus = Status {\n  @go output \"core/pkg/status\"\n}\n"
			result := format(source)
			Expect(result).To(ContainSubstring("MyStatus = Status"))
			Expect(result).To(ContainSubstring("@go output"))
		})
	})

	Describe("Comments", func() {
		It("should preserve line comments", func() {
			result := format("// This is a comment\nUser struct {}\n")
			Expect(result).To(ContainSubstring("// This is a comment"))
		})

		It("should preserve leading comments before definitions", func() {
			source := "// Copyright header\n\nUser struct {}\n"
			result := format(source)
			Expect(result).To(ContainSubstring("// Copyright header"))
			Expect(result).To(ContainSubstring("User struct {}"))
		})

		It("should preserve leading file comments followed by imports", func() {
			source := "// Copyright 2026\n\nimport \"common.oracle\"\n"
			result := format(source)
			Expect(result).To(ContainSubstring("// Copyright 2026"))
			Expect(result).To(ContainSubstring("import \"common.oracle\""))
		})
	})

	Describe("Blank Lines", func() {
		It("should insert blank line between imports and definitions", func() {
			result := format("import \"common.oracle\"\nUser struct {}\n")
			Expect(result).To(ContainSubstring("import \"common.oracle\"\n\nUser struct {}"))
		})

		It("should insert blank line between definitions", func() {
			result := format("User struct {}\nStatus enum {}\n")
			Expect(result).To(ContainSubstring("User struct {}\n\nStatus enum {}"))
		})

		It("should insert blank line between file domains and definitions", func() {
			result := format("@go output \"core/pkg/user\"\nUser struct {}\n")
			Expect(result).To(ContainSubstring("@go output \"core/pkg/user\"\n\nUser struct {}"))
		})

		It("should insert blank line between imports and file domains", func() {
			result := format("import \"common.oracle\"\n@go output \"core/pkg/user\"\n\nUser struct {}\n")
			Expect(result).To(ContainSubstring("import \"common.oracle\"\n\n@go output"))
		})
	})

	Describe("Trailing Newline", func() {
		It("should ensure trailing newline", func() {
			Expect(format("User struct {}")).To(HaveSuffix("\n"))
		})
	})

	Describe("Type Parameters", func() {
		It("should format struct with type parameters", func() {
			result := format("Response<T> struct {\n  data T\n}\n")
			Expect(result).To(ContainSubstring("Response<T>"))
		})

		It("should format type params with extends constraint", func() {
			result := format("Status<D extends json> struct {\n  data D\n}\n")
			Expect(result).To(ContainSubstring("<D extends json>"))
		})

		It("should format type params with default value", func() {
			result := format("Config<T = string> struct {\n  value T\n}\n")
			Expect(result).To(ContainSubstring("<T = string>"))
		})

		It("should format type params with extends and default", func() {
			result := format("Task<C extends json = json> struct {\n  config C\n}\n")
			Expect(result).To(ContainSubstring("extends json"))
			Expect(result).To(ContainSubstring("= json"))
		})

		It("should format struct alias with type parameters", func() {
			result := format("MyResponse<T> = Response<T>\n")
			Expect(result).To(ContainSubstring("MyResponse<T>"))
			Expect(result).To(ContainSubstring("Response<T>"))
		})

		It("should format typedef with type parameters", func() {
			result := format("List<T> T[]\n")
			Expect(result).To(ContainSubstring("List<T>"))
		})

		It("should format optional type parameter", func() {
			result := format("Wrapper<T?> struct {\n  value T\n}\n")
			Expect(result).To(ContainSubstring("<T?>"))
		})
	})

	Describe("Complex Types", func() {
		It("should format map types", func() {
			result := format("Config struct {\n  metadata map<string, json>\n}\n")
			Expect(result).To(ContainSubstring("map<string, json>"))
		})

		It("should format optional types", func() {
			result := format("User struct {\n  nickname string?\n}\n")
			Expect(result).To(ContainSubstring("string?"))
		})

		It("should format array types", func() {
			result := format("User struct {\n  tags string[]\n}\n")
			Expect(result).To(ContainSubstring("string[]"))
		})

		It("should format fixed-size array types", func() {
			result := format("Data struct {\n  values float64[10]\n}\n")
			Expect(result).To(ContainSubstring("float64[10]"))
		})

		It("should format optional map types", func() {
			result := format("Config struct {\n  metadata map<string, json>?\n}\n")
			Expect(result).To(ContainSubstring("map<string, json>?"))
		})

		It("should format type args on references", func() {
			result := format("Response struct extends Base<string> {\n  data string\n}\n")
			Expect(result).To(ContainSubstring("Base<string>"))
		})

		It("should format qualified type references", func() {
			result := format("User struct {\n  role access.Role\n}\n")
			Expect(result).To(ContainSubstring("access.Role"))
		})
	})

	Describe("Expression Values", func() {
		It("should format integer expression values", func() {
			result := format("User struct {\n  name string @validate max_length 255\n}\n")
			Expect(result).To(ContainSubstring("max_length 255"))
		})

		It("should format string expression values", func() {
			result := format("@go output \"core/pkg/user\"\n\nUser struct {}\n")
			Expect(result).To(ContainSubstring("\"core/pkg/user\""))
		})

		It("should format boolean expression values", func() {
			result := format("User struct {\n  active bool @validate default true\n}\n")
			Expect(result).To(ContainSubstring("default true"))
		})

		It("should format qualified ident expression values", func() {
			result := format("User struct {\n  role string @relation target access.Role\n}\n")
			Expect(result).To(ContainSubstring("target access.Role"))
		})
	})

	Describe("Idempotency", func() {
		It("should produce the same output when formatted twice", func() {
			source := "import \"common.oracle\"\n\n@go output \"core/pkg/user\"\n\nUser struct {\n  name string @validate required\n  age  int32\n}\n"
			first := format(source)
			second := format(first)
			Expect(second).To(Equal(first))
		})

		It("should be idempotent for complex schemas", func() {
			source := "import \"label.oracle\"\n\n@ts output \"client/ts\"\n@go output \"core/pkg\"\n\nStatus enum {\n  ACTIVE = 1\n  INACTIVE = 2\n}\n\nUser struct {\n  name string\n  age int32\n}\n"
			first := format(source)
			second := format(first)
			Expect(second).To(Equal(first))
		})
	})
})
