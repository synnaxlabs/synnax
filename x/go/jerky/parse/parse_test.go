// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parse_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/jerky/parse"
)

var _ = Describe("Parser", func() {
	var (
		parser  *parse.Parser
		tempDir string
	)

	BeforeEach(func() {
		parser = parse.NewParser()
		var err error
		tempDir, err = os.MkdirTemp("", "jerky-parse-test-*")
		Expect(err).ToNot(HaveOccurred())
		// Create a go.mod file so packages.Load works
		goMod := `module testmod

go 1.21

require github.com/google/uuid v1.6.0
`
		err = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("ParseFile", func() {
		Context("with a simple struct", func() {
			It("should parse a struct with primitive fields", func() {
				source := `package test

//go:generate jerky
type User struct {
	Name string
	Age  int
}
`
				file := writeTestFile(tempDir, "user.go", source)
				structs, err := parser.ParseFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(structs).To(HaveLen(1))
				Expect(structs[0].Name).To(Equal("User"))
				Expect(structs[0].Fields).To(HaveLen(2))
				Expect(structs[0].Fields[0].Name).To(Equal("Name"))
				Expect(structs[0].Fields[0].GoType.Kind).To(Equal(parse.KindPrimitive))
				Expect(structs[0].Fields[0].GoType.Name).To(Equal("string"))
				Expect(structs[0].Fields[1].Name).To(Equal("Age"))
				Expect(structs[0].Fields[1].GoType.Name).To(Equal("int"))
			})
		})

		Context("with named types", func() {
			It("should parse time.Time types", func() {
				source := `package test

import "time"

//go:generate jerky
type Entity struct {
	CreatedAt time.Time
}
`
				file := writeTestFile(tempDir, "entity.go", source)
				structs, err := parser.ParseFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(structs).To(HaveLen(1))
				Expect(structs[0].Fields[0].GoType.Kind).To(Equal(parse.KindNamed))
				Expect(structs[0].Fields[0].GoType.Name).To(Equal("time.Time"))
				Expect(structs[0].Fields[0].GoType.PackagePath).To(Equal("time"))
			})
		})

		Context("with type aliases", func() {
			It("should detect underlying types for type wrappers", func() {
				source := `package test

type UserID uint32

//go:generate jerky
type User struct {
	ID UserID
}
`
				file := writeTestFile(tempDir, "user.go", source)
				structs, err := parser.ParseFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(structs).To(HaveLen(1))
				field := structs[0].Fields[0]
				Expect(field.Name).To(Equal("ID"))
				Expect(field.GoType.Kind).To(Equal(parse.KindNamed))
				Expect(field.GoType.Underlying).ToNot(BeNil())
				Expect(field.GoType.Underlying.Name).To(Equal("uint32"))
			})
		})

		Context("with slice types", func() {
			It("should parse []string", func() {
				source := `package test

//go:generate jerky
type Tags struct {
	Values []string
}
`
				file := writeTestFile(tempDir, "tags.go", source)
				structs, err := parser.ParseFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(structs).To(HaveLen(1))
				field := structs[0].Fields[0]
				Expect(field.GoType.Kind).To(Equal(parse.KindSlice))
				Expect(field.GoType.Elem).ToNot(BeNil())
				Expect(field.GoType.Elem.Name).To(Equal("string"))
			})

			It("should parse []byte", func() {
				source := `package test

//go:generate jerky
type Binary struct {
	Data []byte
}
`
				file := writeTestFile(tempDir, "binary.go", source)
				structs, err := parser.ParseFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(structs).To(HaveLen(1))
				field := structs[0].Fields[0]
				Expect(field.GoType.Kind).To(Equal(parse.KindSlice))
				Expect(field.GoType.Name).To(Equal("[]byte"))
			})
		})

		Context("with map types", func() {
			It("should parse map[string]int", func() {
				source := `package test

//go:generate jerky
type Counts struct {
	Values map[string]int
}
`
				file := writeTestFile(tempDir, "counts.go", source)
				structs, err := parser.ParseFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(structs).To(HaveLen(1))
				field := structs[0].Fields[0]
				Expect(field.GoType.Kind).To(Equal(parse.KindMap))
				Expect(field.GoType.Key).ToNot(BeNil())
				Expect(field.GoType.Key.Name).To(Equal("string"))
				Expect(field.GoType.Elem).ToNot(BeNil())
				Expect(field.GoType.Elem.Name).To(Equal("int"))
			})
		})

		Context("with struct tags", func() {
			It("should parse json tags", func() {
				source := `package test

//go:generate jerky
type User struct {
	Name string ` + "`json:\"name\"`" + `
}
`
				file := writeTestFile(tempDir, "user.go", source)
				structs, err := parser.ParseFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(structs).To(HaveLen(1))
				Expect(structs[0].Fields[0].Tags.JSON).To(Equal("name"))
			})
		})

		Context("with unexported fields", func() {
			It("should skip unexported fields", func() {
				source := `package test

//go:generate jerky
type User struct {
	Name     string
	internal int
	Email    string
}
`
				file := writeTestFile(tempDir, "user.go", source)
				structs, err := parser.ParseFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(structs).To(HaveLen(1))
				Expect(structs[0].Fields).To(HaveLen(2))
				Expect(structs[0].FieldNames()).To(Equal([]string{"Name", "Email"}))
			})
		})

		Context("with no jerky directive", func() {
			It("should return empty when no directive present", func() {
				source := `package test

type User struct {
	Name string
}
`
				file := writeTestFile(tempDir, "user.go", source)
				structs, err := parser.ParseFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(structs).To(BeEmpty())
			})
		})

		Context("with multiple structs", func() {
			It("should parse multiple annotated structs", func() {
				source := `package test

//go:generate jerky
type User struct {
	Name string
}

//go:generate jerky
type Group struct {
	Title string
}
`
				file := writeTestFile(tempDir, "types.go", source)
				structs, err := parser.ParseFile(file)
				Expect(err).ToNot(HaveOccurred())
				Expect(structs).To(HaveLen(2))
				Expect(structs[0].Name).To(Equal("User"))
				Expect(structs[1].Name).To(Equal("Group"))
			})
		})
	})
})

func writeTestFile(dir, name, content string) string {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	Expect(err).ToNot(HaveOccurred())
	return path
}
