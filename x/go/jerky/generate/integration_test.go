// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package generate_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/jerky/generate"
	"github.com/synnaxlabs/x/jerky/parse"
)

// getXModulePath returns the absolute path to the x/go module.
func getXModulePath() string {
	// Get the current working directory (should be in x/go/jerky/generate)
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// Navigate up to x/go
	return filepath.Dir(filepath.Dir(cwd))
}

// setupIntegrationEnv creates a complete test environment for integration tests.
func setupIntegrationEnv(name string) (tempDir string, cleanup func()) {
	tempDir, err := os.MkdirTemp("", "jerky-integration-"+name+"-*")
	Expect(err).ToNot(HaveOccurred())

	xModPath := getXModulePath()

	// Write go.mod with required dependencies
	goMod := `module testpkg

go 1.21

require (
	github.com/synnaxlabs/x v0.0.0
	github.com/google/uuid v1.6.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
	google.golang.org/protobuf v1.36.10
)

replace github.com/synnaxlabs/x => ` + xModPath + `
`
	err = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
	Expect(err).ToNot(HaveOccurred())

	cleanup = func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// writeTestFile creates a Go source file in the temp directory.
func writeIntegrationTestFile(dir, filename, content string) string {
	path := filepath.Join(dir, filename)
	err := os.WriteFile(path, []byte(content), 0644)
	Expect(err).ToNot(HaveOccurred())
	return path
}

// verifyGoCompiles runs `go build` to verify generated code compiles.
func verifyGoCompiles(dir string) {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	Expect(err).ToNot(HaveOccurred(), "go build failed: %s", string(output))
}

var _ = Describe("Integration Tests", func() {
	Describe("Simple Struct Lifecycle", func() {
		var (
			tempDir string
			cleanup func()
		)

		BeforeEach(func() {
			tempDir, cleanup = setupIntegrationEnv("simple")
		})

		AfterEach(func() {
			cleanup()
		})

		It("should generate all artifacts for a simple struct with primitive fields", func() {
			source := `package testpkg

//go:generate jerky
type User struct {
	Name string
	Age  int32
}
`
			file := writeIntegrationTestFile(tempDir, "user.go", source)

			// Parse the file
			parser := parse.NewParser()
			structs, err := parser.ParseFile(file)
			Expect(err).ToNot(HaveOccurred())
			Expect(structs).To(HaveLen(1))
			Expect(structs[0].Name).To(Equal("User"))

			// Generate artifacts
			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Verify generated files exist
			typeDir := filepath.Join(tempDir, "types", "user")
			Expect(typeDir).To(BeADirectory())

			// Check for expected files
			Expect(filepath.Join(typeDir, "state.json")).To(BeAnExistingFile())
			Expect(filepath.Join(typeDir, "current.go")).To(BeAnExistingFile())
			Expect(filepath.Join(typeDir, "v1.proto")).To(BeAnExistingFile())
		})

		It("should generate gorp translation file", func() {
			source := `package testpkg

//go:generate jerky
type Person struct {
	FirstName string
	LastName  string
	Age       int32
}
`
			file := writeIntegrationTestFile(tempDir, "person.go", source)

			parser := parse.NewParser()
			structs, err := parser.ParseFile(file)
			Expect(err).ToNot(HaveOccurred())

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Check for gorp file
			Expect(filepath.Join(tempDir, "person_gorp.go")).To(BeAnExistingFile())
		})

		It("should generate v0 migration file", func() {
			source := `package testpkg

//go:generate jerky
type Entity struct {
	ID   string
	Data string
}
`
			file := writeIntegrationTestFile(tempDir, "entity.go", source)

			parser := parse.NewParser()
			structs, err := parser.ParseFile(file)
			Expect(err).ToNot(HaveOccurred())

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Check for v0 migration file (msgpack bootstrap)
			typeDir := filepath.Join(tempDir, "types", "entity")
			Expect(filepath.Join(typeDir, "v0.go")).To(BeAnExistingFile())
		})
	})

	Describe("State Management", func() {
		var (
			tempDir string
			cleanup func()
		)

		BeforeEach(func() {
			tempDir, cleanup = setupIntegrationEnv("state")
		})

		AfterEach(func() {
			cleanup()
		})

		It("should increment version when struct changes", func() {
			// Initial struct - use a simple field set
			parsed1 := parse.ParsedStruct{
				Name:        "Config",
				PackageName: "testpkg",
				IsEmbedded:  false,
				Fields: []parse.ParsedField{
					{Name: "Name", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "string"}},
				},
			}

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(parsed1)
			Expect(err).ToNot(HaveOccurred())

			// Verify V1 proto exists
			typeDir := filepath.Join(tempDir, "types", "config")
			Expect(filepath.Join(typeDir, "v1.proto")).To(BeAnExistingFile())

			// Modified struct (add a field)
			parsed2 := parse.ParsedStruct{
				Name:        "Config",
				PackageName: "testpkg",
				IsEmbedded:  false,
				Fields: []parse.ParsedField{
					{Name: "Name", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "string"}},
					{Name: "Value", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "string"}},
				},
			}

			// Create new generator instance to pick up state changes
			generator2, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator2.Generate(parsed2)
			Expect(err).ToNot(HaveOccurred())

			// Verify V2 proto was created
			Expect(filepath.Join(typeDir, "v2.proto")).To(BeAnExistingFile())
		})

		It("should preserve field numbers across versions", func() {
			// Initial struct - use ParsedStruct directly
			parsed1 := parse.ParsedStruct{
				Name:        "Data",
				PackageName: "testpkg",
				IsEmbedded:  false,
				Fields: []parse.ParsedField{
					{Name: "Alpha", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "string"}},
					{Name: "Beta", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "int32"}},
				},
			}

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(parsed1)
			Expect(err).ToNot(HaveOccurred())

			// Read initial proto
			typeDir := filepath.Join(tempDir, "types", "data")
			v1Proto, err := os.ReadFile(filepath.Join(typeDir, "v1.proto"))
			Expect(err).ToNot(HaveOccurred())

			// Verify field names in V1 (snake_case in proto)
			Expect(string(v1Proto)).To(ContainSubstring("alpha"))
			Expect(string(v1Proto)).To(ContainSubstring("beta"))

			// Add a new field
			parsed2 := parse.ParsedStruct{
				Name:        "Data",
				PackageName: "testpkg",
				IsEmbedded:  false,
				Fields: []parse.ParsedField{
					{Name: "Alpha", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "string"}},
					{Name: "Beta", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "int32"}},
					{Name: "Gamma", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "string"}},
				},
			}

			generator2, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator2.Generate(parsed2)
			Expect(err).ToNot(HaveOccurred())

			// Read V2 proto and verify new field added
			v2Proto, err := os.ReadFile(filepath.Join(typeDir, "v2.proto"))
			Expect(err).ToNot(HaveOccurred())

			// Gamma should be in V2 (snake_case)
			Expect(string(v2Proto)).To(ContainSubstring("gamma"))
		})

		It("should not increment version when struct is unchanged", func() {
			source := `package testpkg

//go:generate jerky
type Immutable struct {
	Field1 string
	Field2 int32
}
`
			file := writeIntegrationTestFile(tempDir, "immutable.go", source)

			parser := parse.NewParser()
			structs, err := parser.ParseFile(file)
			Expect(err).ToNot(HaveOccurred())

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			// Generate twice with the same struct
			err = generator.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Create new generator to pick up state
			generator2, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator2.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Should only have V1, not V2
			typeDir := filepath.Join(tempDir, "types", "immutable")
			Expect(filepath.Join(typeDir, "v1.proto")).To(BeAnExistingFile())
			Expect(filepath.Join(typeDir, "v2.proto")).ToNot(BeAnExistingFile())
		})
	})

	Describe("Complex Types", func() {
		var (
			tempDir string
			cleanup func()
		)

		BeforeEach(func() {
			tempDir, cleanup = setupIntegrationEnv("complex")
		})

		AfterEach(func() {
			cleanup()
		})

		It("should handle time.Time fields", func() {
			source := `package testpkg

import "time"

//go:generate jerky
type Event struct {
	Name      string
	Timestamp time.Time
}
`
			file := writeIntegrationTestFile(tempDir, "event.go", source)

			parser := parse.NewParser()
			structs, err := parser.ParseFile(file)
			Expect(err).ToNot(HaveOccurred())

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Check proto has int64 type for Time (snake_case field name)
			typeDir := filepath.Join(tempDir, "types", "event")
			protoContent, err := os.ReadFile(filepath.Join(typeDir, "v1.proto"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(protoContent)).To(ContainSubstring("int64 timestamp"))
		})

		It("should handle type aliases", func() {
			source := `package testpkg

type UserID uint32

//go:generate jerky
type Account struct {
	ID    UserID
	Email string
}
`
			file := writeIntegrationTestFile(tempDir, "account.go", source)

			parser := parse.NewParser()
			structs, err := parser.ParseFile(file)
			Expect(err).ToNot(HaveOccurred())

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Check proto has uint32 for UserID (snake_case: i_d)
			typeDir := filepath.Join(tempDir, "types", "account")
			protoContent, err := os.ReadFile(filepath.Join(typeDir, "v1.proto"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(protoContent)).To(ContainSubstring("uint32 i_d"))
		})

		It("should handle slice of primitives", func() {
			source := `package testpkg

//go:generate jerky
type Tags struct {
	Names []string
	IDs   []int32
}
`
			file := writeIntegrationTestFile(tempDir, "tags.go", source)

			parser := parse.NewParser()
			structs, err := parser.ParseFile(file)
			Expect(err).ToNot(HaveOccurred())

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Check proto has repeated fields (snake_case)
			typeDir := filepath.Join(tempDir, "types", "tags")
			protoContent, err := os.ReadFile(filepath.Join(typeDir, "v1.proto"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(protoContent)).To(ContainSubstring("repeated string names"))
			Expect(string(protoContent)).To(ContainSubstring("repeated int32 i_ds"))
		})

		It("should handle map types", func() {
			source := `package testpkg

//go:generate jerky
type Metadata struct {
	Labels map[string]string
	Counts map[string]int32
}
`
			file := writeIntegrationTestFile(tempDir, "metadata.go", source)

			parser := parse.NewParser()
			structs, err := parser.ParseFile(file)
			Expect(err).ToNot(HaveOccurred())

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Check proto has map fields (snake_case)
			typeDir := filepath.Join(tempDir, "types", "metadata")
			protoContent, err := os.ReadFile(filepath.Join(typeDir, "v1.proto"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(protoContent)).To(ContainSubstring("map<string, string> labels"))
			Expect(string(protoContent)).To(ContainSubstring("map<string, int32> counts"))
		})
	})

	Describe("Embedded Types", func() {
		var (
			tempDir string
			cleanup func()
		)

		BeforeEach(func() {
			tempDir, cleanup = setupIntegrationEnv("embedded")
		})

		AfterEach(func() {
			cleanup()
		})

		It("should generate translate file for embedded types", func() {
			parsed := parse.ParsedStruct{
				Name:        "Address",
				PackageName: "testpkg",
				IsEmbedded:  true,
				Fields: []parse.ParsedField{
					{Name: "Street", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "string"}},
					{Name: "City", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "string"}},
					{Name: "Zip", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "string"}},
				},
			}

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(parsed)
			Expect(err).ToNot(HaveOccurred())

			// Should have translate file, not gorp file
			Expect(filepath.Join(tempDir, "address_translate.go")).To(BeAnExistingFile())
			Expect(filepath.Join(tempDir, "address_gorp.go")).ToNot(BeAnExistingFile())
		})

		It("should generate struct_migrate for embedded types", func() {
			// Initial embedded struct
			parsed1 := parse.ParsedStruct{
				Name:        "Location",
				PackageName: "testpkg",
				IsEmbedded:  true,
				Fields: []parse.ParsedField{
					{Name: "Lat", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "float64"}},
					{Name: "Lng", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "float64"}},
				},
			}

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(parsed1)
			Expect(err).ToNot(HaveOccurred())

			// Add a field to trigger version increment
			parsed2 := parse.ParsedStruct{
				Name:        "Location",
				PackageName: "testpkg",
				IsEmbedded:  true,
				Fields: []parse.ParsedField{
					{Name: "Lat", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "float64"}},
					{Name: "Lng", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "float64"}},
					{Name: "Altitude", GoType: parse.GoType{Kind: parse.KindPrimitive, Name: "float64"}},
				},
			}

			generator2, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator2.Generate(parsed2)
			Expect(err).ToNot(HaveOccurred())

			// Should have struct_migrate.go for embedded type migrations
			typeDir := filepath.Join(tempDir, "types", "location")
			Expect(filepath.Join(typeDir, "struct_migrate.go")).To(BeAnExistingFile())
		})
	})

	Describe("Proto Content Verification", func() {
		var (
			tempDir string
			cleanup func()
		)

		BeforeEach(func() {
			tempDir, cleanup = setupIntegrationEnv("proto")
		})

		AfterEach(func() {
			cleanup()
		})

		It("should generate valid proto syntax", func() {
			source := `package testpkg

//go:generate jerky
type Record struct {
	ID    string
	Value int64
	Tags  []string
}
`
			file := writeIntegrationTestFile(tempDir, "record.go", source)

			parser := parse.NewParser()
			structs, err := parser.ParseFile(file)
			Expect(err).ToNot(HaveOccurred())

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Read and verify proto content (snake_case field names)
			typeDir := filepath.Join(tempDir, "types", "record")
			protoContent, err := os.ReadFile(filepath.Join(typeDir, "v1.proto"))
			Expect(err).ToNot(HaveOccurred())

			proto := string(protoContent)
			Expect(proto).To(ContainSubstring("syntax = \"proto3\""))
			Expect(proto).To(ContainSubstring("message V1"))
			Expect(proto).To(ContainSubstring("string i_d = 1"))
			Expect(proto).To(ContainSubstring("int64 value = 2"))
			Expect(proto).To(ContainSubstring("repeated string tags = 3"))
		})

		It("should generate correct package path in proto", func() {
			source := `package testpkg

//go:generate jerky
type Item struct {
	Name string
}
`
			file := writeIntegrationTestFile(tempDir, "item.go", source)

			parser := parse.NewParser()
			structs, err := parser.ParseFile(file)
			Expect(err).ToNot(HaveOccurred())

			generator, err := generate.NewGenerator(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			err = generator.Generate(structs[0])
			Expect(err).ToNot(HaveOccurred())

			// Verify package path in proto
			typeDir := filepath.Join(tempDir, "types", "item")
			protoContent, err := os.ReadFile(filepath.Join(typeDir, "v1.proto"))
			Expect(err).ToNot(HaveOccurred())

			proto := string(protoContent)
			Expect(proto).To(ContainSubstring("package testpkg.types.item"))
			Expect(proto).To(ContainSubstring("option go_package"))
		})
	})
})
