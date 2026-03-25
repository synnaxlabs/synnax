// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin/go/migrate"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

func TestLayout(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Layout Suite")
}

var _ = Describe("BuildLayout", Ordered, func() {
	var (
		table *resolution.Table
	)

	BeforeAll(func() {
		repoRoot := MustSucceed(paths.RepoRoot())
		loader := analyzer.NewStandardFileLoader(repoRoot)
		var diag interface{ Ok() bool }
		table, diag = analyzer.Analyze(
			context.Background(),
			[]string{"schemas/arc.oracle"},
			loader,
		)
		Expect(diag.Ok()).To(BeTrue(), "analyzing arc schema should produce no errors")
		Expect(table).NotTo(BeNil())
	})

	Describe("Arc type layout", Ordered, func() {
		var (
			arcType resolution.Type
			layouts []gorp.FieldLayout
		)

		BeforeAll(func() {
			var ok bool
			arcType, ok = table.Get("arc.Arc")
			Expect(ok).To(BeTrue(), "arc.Arc type must exist in the resolution table")

			var err error
			layouts, err = migrate.BuildLayout(arcType, table)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should have 7 top-level fields", func() {
			Expect(layouts).To(HaveLen(7))
		})

		It("Should have correct field names in order", func() {
			names := make([]string, len(layouts))
			for i, l := range layouts {
				names[i] = l.Name
			}
			Expect(names).To(Equal([]string{
				"key", "name", "mode", "graph", "text", "program", "status",
			}))
		})

		It("Should encode key as UUID", func() {
			Expect(layouts[0].Encoding).To(Equal(gorp.EncodingUUID))
		})

		It("Should encode name as String", func() {
			Expect(layouts[1].Encoding).To(Equal(gorp.EncodingString))
		})

		It("Should encode mode enum as String", func() {
			Expect(layouts[2].Encoding).To(Equal(gorp.EncodingString))
		})

		Describe("graph field", func() {
			It("Should be a struct", func() {
				Expect(layouts[3].Encoding).To(Equal(gorp.EncodingStruct))
				Expect(layouts[3].Fields).NotTo(BeEmpty())
			})

			It("Should have viewport, functions, edges, and nodes sub-fields", func() {
				graphFieldNames := fieldNames(layouts[3].Fields)
				Expect(graphFieldNames).To(ContainElements(
					"viewport", "functions", "edges", "nodes",
				))
			})

			It("Should encode functions as an array of structs", func() {
				functionsField := findField(layouts[3].Fields, "functions")
				Expect(functionsField).NotTo(BeNil())
				Expect(functionsField.Encoding).To(Equal(gorp.EncodingArray))
				Expect(functionsField.Element).NotTo(BeNil())
				Expect(functionsField.Element.Encoding).To(Equal(gorp.EncodingStruct))
				Expect(functionsField.Element.Fields).NotTo(BeEmpty())
			})

			It("Should encode edges as an array of structs", func() {
				edgesField := findField(layouts[3].Fields, "edges")
				Expect(edgesField).NotTo(BeNil())
				Expect(edgesField.Encoding).To(Equal(gorp.EncodingArray))
				Expect(edgesField.Element).NotTo(BeNil())
				Expect(edgesField.Element.Encoding).To(Equal(gorp.EncodingStruct))
			})

			It("Should encode nodes as an array of structs", func() {
				nodesField := findField(layouts[3].Fields, "nodes")
				Expect(nodesField).NotTo(BeNil())
				Expect(nodesField.Encoding).To(Equal(gorp.EncodingArray))
				Expect(nodesField.Element).NotTo(BeNil())
				Expect(nodesField.Element.Encoding).To(Equal(gorp.EncodingStruct))
			})

			It("Should encode viewport as a struct with position and zoom", func() {
				viewportField := findField(layouts[3].Fields, "viewport")
				Expect(viewportField).NotTo(BeNil())
				Expect(viewportField.Encoding).To(Equal(gorp.EncodingStruct))
				vpFieldNames := fieldNames(viewportField.Fields)
				Expect(vpFieldNames).To(ContainElements("position", "zoom"))
			})
		})

		Describe("text field", func() {
			It("Should be a struct with a raw string field", func() {
				Expect(layouts[4].Encoding).To(Equal(gorp.EncodingStruct))
				rawField := findField(layouts[4].Fields, "raw")
				Expect(rawField).NotTo(BeNil())
				Expect(rawField.Encoding).To(Equal(gorp.EncodingString))
			})
		})

		Describe("program field (hard optional struct extends IR + Output)", func() {
			It("Should be marked HardOptional", func() {
				Expect(layouts[5].HardOptional).To(BeTrue())
			})

			It("Should be a struct encoding", func() {
				Expect(layouts[5].Encoding).To(Equal(gorp.EncodingStruct))
			})

			It("Should contain fields from both IR and compiler.Output", func() {
				programFieldNames := fieldNames(layouts[5].Fields)
				Expect(programFieldNames).To(ContainElements(
					"functions", "nodes", "edges", "strata",
					"sequences", "authorities",
					"WASM", "OutputMemoryBases",
				))
			})
		})

		Describe("status field (hard optional alias to Status<StatusDetails>)", func() {
			It("Should be marked HardOptional", func() {
				Expect(layouts[6].HardOptional).To(BeTrue())
			})

			It("Should be a struct encoding", func() {
				Expect(layouts[6].Encoding).To(Equal(gorp.EncodingStruct))
			})

			It("Should contain key, variant, message, and details fields", func() {
				statusFieldNames := fieldNames(layouts[6].Fields)
				Expect(statusFieldNames).To(ContainElements(
					"key", "variant", "message", "details",
				))
			})
		})
	})

	Describe("Recursive Type/Param handling", func() {
		It("Should build layout for types.Type without infinite loop", func() {
			typ, ok := table.Get("types.Type")
			Expect(ok).To(BeTrue(), "types.Type must exist")

			layouts, err := migrate.BuildLayout(typ, table)
			Expect(err).NotTo(HaveOccurred())
			Expect(layouts).NotTo(BeEmpty())
		})

		It("Should have elem field as a hard optional struct", func() {
			typ := table.MustGet("types.Type")
			layouts := MustSucceed(migrate.BuildLayout(typ, table))

			elemField := findField(layouts, "elem")
			Expect(elemField).NotTo(BeNil(), "types.Type should have an elem field")
			Expect(elemField.HardOptional).To(BeTrue())
			Expect(elemField.Encoding).To(Equal(gorp.EncodingStruct))
		})

		It("Should terminate recursion in the elem field's nested Type", func() {
			typ := table.MustGet("types.Type")
			layouts := MustSucceed(migrate.BuildLayout(typ, table))

			elemField := findField(layouts, "elem")
			Expect(elemField).NotTo(BeNil())

			nestedElem := findField(elemField.Fields, "elem")
			Expect(nestedElem).NotTo(BeNil(), "nested Type should also have elem")
			Expect(nestedElem.Encoding).To(Equal(gorp.EncodingStruct))
			Expect(nestedElem.Fields).To(BeEmpty(),
				"recursive reference should terminate with empty fields")
		})

		It("Should have constraint field as a hard optional recursive struct", func() {
			typ := table.MustGet("types.Type")
			layouts := MustSucceed(migrate.BuildLayout(typ, table))

			constraintField := findField(layouts, "constraint")
			Expect(constraintField).NotTo(BeNil())
			Expect(constraintField.HardOptional).To(BeTrue())
			Expect(constraintField.Encoding).To(Equal(gorp.EncodingStruct))
		})

		It("Should build layout for types.Param containing recursive Type", func() {
			typ, ok := table.Get("types.Param")
			Expect(ok).To(BeTrue(), "types.Param must exist")

			layouts, err := migrate.BuildLayout(typ, table)
			Expect(err).NotTo(HaveOccurred())
			Expect(layouts).NotTo(BeEmpty())

			typeField := findField(layouts, "type")
			Expect(typeField).NotTo(BeNil())
			Expect(typeField.Encoding).To(Equal(gorp.EncodingStruct))
			Expect(typeField.Fields).NotTo(BeEmpty())
		})

		It("Should build layout for ir.Function with nested Params arrays", func() {
			typ, ok := table.Get("ir.Function")
			Expect(ok).To(BeTrue(), "ir.Function must exist")

			layouts, err := migrate.BuildLayout(typ, table)
			Expect(err).NotTo(HaveOccurred())

			configField := findField(layouts, "config")
			Expect(configField).NotTo(BeNil())
			Expect(configField.Encoding).To(Equal(gorp.EncodingArray))
			Expect(configField.Element).NotTo(BeNil())
			Expect(configField.Element.Encoding).To(Equal(gorp.EncodingStruct))
		})
	})

	Describe("types.Channels layout", func() {
		It("Should have map fields for read and write", func() {
			typ, ok := table.Get("types.Channels")
			Expect(ok).To(BeTrue())

			layouts, err := migrate.BuildLayout(typ, table)
			Expect(err).NotTo(HaveOccurred())
			Expect(layouts).To(HaveLen(2))

			readField := findField(layouts, "read")
			Expect(readField).NotTo(BeNil())
			Expect(readField.Encoding).To(Equal(gorp.EncodingMap))
			Expect(readField.Key).NotTo(BeNil())
			Expect(readField.Key.Encoding).To(Equal(gorp.EncodingUint32))
			Expect(readField.Value).NotTo(BeNil())
			Expect(readField.Value.Encoding).To(Equal(gorp.EncodingString))

			writeField := findField(layouts, "write")
			Expect(writeField).NotTo(BeNil())
			Expect(writeField.Encoding).To(Equal(gorp.EncodingMap))
		})
	})

	Describe("ir.Authorities layout", func() {
		It("Should have default as hard optional and channels as a soft optional map", func() {
			typ, ok := table.Get("ir.Authorities")
			Expect(ok).To(BeTrue())

			layouts, err := migrate.BuildLayout(typ, table)
			Expect(err).NotTo(HaveOccurred())
			Expect(layouts).To(HaveLen(2))

			defaultField := findField(layouts, "default")
			Expect(defaultField).NotTo(BeNil())
			Expect(defaultField.HardOptional).To(BeTrue())
			Expect(defaultField.Encoding).To(Equal(gorp.EncodingUint8))

			channelsField := findField(layouts, "channels")
			Expect(channelsField).NotTo(BeNil())
			Expect(channelsField.Optional).To(BeTrue())
			Expect(channelsField.Encoding).To(Equal(gorp.EncodingMap))
		})
	})

	Describe("compiler.Output layout", func() {
		It("Should encode WASM as bytes and OutputMemoryBases as a map", func() {
			typ, ok := table.Get("compiler.Output")
			Expect(ok).To(BeTrue())

			layouts, err := migrate.BuildLayout(typ, table)
			Expect(err).NotTo(HaveOccurred())
			Expect(layouts).To(HaveLen(2))

			wasmField := findField(layouts, "WASM")
			Expect(wasmField).NotTo(BeNil())
			Expect(wasmField.Encoding).To(Equal(gorp.EncodingBytes))

			memField := findField(layouts, "OutputMemoryBases")
			Expect(memField).NotTo(BeNil())
			Expect(memField.Encoding).To(Equal(gorp.EncodingMap))
			Expect(memField.Key.Encoding).To(Equal(gorp.EncodingString))
			Expect(memField.Value.Encoding).To(Equal(gorp.EncodingUint32))
		})
	})
})

func fieldNames(fields []gorp.FieldLayout) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}

func findField(fields []gorp.FieldLayout, name string) *gorp.FieldLayout {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}
