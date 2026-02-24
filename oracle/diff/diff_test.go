// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package diff_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/diff"
	"github.com/synnaxlabs/oracle/resolution"
)

func makeStruct(name string, fields []resolution.Field) resolution.Type {
	return resolution.Type{
		Name:          name,
		QualifiedName: name,
		Form:          resolution.StructForm{Fields: fields},
	}
}

func makeField(name, typeName string) resolution.Field {
	return resolution.Field{Name: name, Type: resolution.TypeRef{Name: typeName}}
}

func makeFieldRef(name string, ref resolution.TypeRef) resolution.Field {
	return resolution.Field{Name: name, Type: ref}
}

func intPtr(v int64) *int64 { return &v }

var _ = Describe("Diff", func() {
	Describe("FormatTypeRef", func() {
		It("Should format a simple name", func() {
			ref := resolution.TypeRef{Name: "string"}
			Expect(diff.FormatTypeRef(ref)).To(Equal("string"))
		})

		It("Should format with TypeArgs", func() {
			ref := resolution.TypeRef{
				Name:     "Array",
				TypeArgs: []resolution.TypeRef{{Name: "int32"}},
			}
			Expect(diff.FormatTypeRef(ref)).To(Equal("Array<int32>"))
		})

		It("Should format nested TypeArgs", func() {
			ref := resolution.TypeRef{
				Name: "Map",
				TypeArgs: []resolution.TypeRef{
					{Name: "string"},
					{Name: "Array", TypeArgs: []resolution.TypeRef{{Name: "float64"}}},
				},
			}
			Expect(diff.FormatTypeRef(ref)).To(Equal("Map<string, Array<float64>>"))
		})

		It("Should format with ArraySize", func() {
			ref := resolution.TypeRef{
				Name:      "int32",
				ArraySize: intPtr(10),
			}
			Expect(diff.FormatTypeRef(ref)).To(Equal("int32[10]"))
		})
	})

	Describe("DiffStructs", func() {
		It("Should report no changes for identical structs", func() {
			old := makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
				makeField("value", "int32"),
			})
			new := makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
				makeField("value", "int32"),
			})
			oldTable := resolution.NewTable()
			Expect(oldTable.Add(old)).To(Succeed())
			newTable := resolution.NewTable()
			Expect(newTable.Add(new)).To(Succeed())

			td := diff.DiffStructs(old, new, oldTable, newTable)
			Expect(td.Changed).To(BeFalse())
			Expect(td.Fields).To(HaveLen(2))
			for _, f := range td.Fields {
				Expect(f.Kind).To(Equal(diff.FieldUnchanged))
			}
		})

		It("Should detect a field added", func() {
			old := makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
			})
			new := makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
				makeField("description", "string"),
			})
			oldTable := resolution.NewTable()
			Expect(oldTable.Add(old)).To(Succeed())
			newTable := resolution.NewTable()
			Expect(newTable.Add(new)).To(Succeed())

			td := diff.DiffStructs(old, new, oldTable, newTable)
			Expect(td.Changed).To(BeTrue())
			var added []diff.FieldDiff
			for _, f := range td.Fields {
				if f.Kind == diff.FieldAdded {
					added = append(added, f)
				}
			}
			Expect(added).To(HaveLen(1))
			Expect(added[0].Name).To(Equal("description"))
		})

		It("Should detect a field removed", func() {
			old := makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
				makeField("legacy", "string"),
			})
			new := makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
			})
			oldTable := resolution.NewTable()
			Expect(oldTable.Add(old)).To(Succeed())
			newTable := resolution.NewTable()
			Expect(newTable.Add(new)).To(Succeed())

			td := diff.DiffStructs(old, new, oldTable, newTable)
			Expect(td.Changed).To(BeTrue())
			var removed []diff.FieldDiff
			for _, f := range td.Fields {
				if f.Kind == diff.FieldRemoved {
					removed = append(removed, f)
				}
			}
			Expect(removed).To(HaveLen(1))
			Expect(removed[0].Name).To(Equal("legacy"))
		})

		It("Should detect a field type change", func() {
			old := makeStruct("Entry", []resolution.Field{
				makeField("count", "int32"),
			})
			new := makeStruct("Entry", []resolution.Field{
				makeField("count", "int64"),
			})
			oldTable := resolution.NewTable()
			Expect(oldTable.Add(old)).To(Succeed())
			newTable := resolution.NewTable()
			Expect(newTable.Add(new)).To(Succeed())

			td := diff.DiffStructs(old, new, oldTable, newTable)
			Expect(td.Changed).To(BeTrue())
			var changed []diff.FieldDiff
			for _, f := range td.Fields {
				if f.Kind == diff.FieldTypeChanged {
					changed = append(changed, f)
				}
			}
			Expect(changed).To(HaveLen(1))
			Expect(changed[0].OldType).To(Equal("int32"))
			Expect(changed[0].NewType).To(Equal("int64"))
		})

		It("Should detect multiple changes", func() {
			old := makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
				makeField("count", "int32"),
				makeField("legacy", "bool"),
			})
			new := makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
				makeField("count", "int64"),
				makeField("description", "string"),
			})
			oldTable := resolution.NewTable()
			Expect(oldTable.Add(old)).To(Succeed())
			newTable := resolution.NewTable()
			Expect(newTable.Add(new)).To(Succeed())

			td := diff.DiffStructs(old, new, oldTable, newTable)
			Expect(td.Changed).To(BeTrue())

			kinds := map[diff.ChangeKind]int{}
			for _, f := range td.Fields {
				kinds[f.Kind]++
			}
			Expect(kinds[diff.FieldUnchanged]).To(Equal(1))
			Expect(kinds[diff.FieldTypeChanged]).To(Equal(1))
			Expect(kinds[diff.FieldRemoved]).To(Equal(1))
			Expect(kinds[diff.FieldAdded]).To(Equal(1))
		})

		It("Should detect optional field", func() {
			old := makeStruct("Entry", []resolution.Field{
				{Name: "opt", Type: resolution.TypeRef{Name: "string"}, IsOptional: true},
			})
			new := makeStruct("Entry", []resolution.Field{
				{Name: "opt", Type: resolution.TypeRef{Name: "string"}, IsOptional: true},
			})
			oldTable := resolution.NewTable()
			Expect(oldTable.Add(old)).To(Succeed())
			newTable := resolution.NewTable()
			Expect(newTable.Add(new)).To(Succeed())

			td := diff.DiffStructs(old, new, oldTable, newTable)
			Expect(td.Changed).To(BeFalse())
		})

		It("Should detect Array field type change", func() {
			old := makeStruct("Entry", []resolution.Field{
				makeFieldRef("items", resolution.TypeRef{
					Name:     "Array",
					TypeArgs: []resolution.TypeRef{{Name: "int32"}},
				}),
			})
			new := makeStruct("Entry", []resolution.Field{
				makeFieldRef("items", resolution.TypeRef{
					Name:     "Array",
					TypeArgs: []resolution.TypeRef{{Name: "int64"}},
				}),
			})
			oldTable := resolution.NewTable()
			Expect(oldTable.Add(old)).To(Succeed())
			newTable := resolution.NewTable()
			Expect(newTable.Add(new)).To(Succeed())

			td := diff.DiffStructs(old, new, oldTable, newTable)
			Expect(td.Changed).To(BeTrue())
			Expect(td.Fields[0].OldType).To(Equal("Array<int32>"))
			Expect(td.Fields[0].NewType).To(Equal("Array<int64>"))
		})

		It("Should detect Map field change", func() {
			old := makeStruct("Entry", []resolution.Field{
				makeFieldRef("data", resolution.TypeRef{
					Name:     "Map",
					TypeArgs: []resolution.TypeRef{{Name: "string"}, {Name: "int32"}},
				}),
			})
			new := makeStruct("Entry", []resolution.Field{
				makeFieldRef("data", resolution.TypeRef{
					Name:     "Map",
					TypeArgs: []resolution.TypeRef{{Name: "string"}, {Name: "float64"}},
				}),
			})
			oldTable := resolution.NewTable()
			Expect(oldTable.Add(old)).To(Succeed())
			newTable := resolution.NewTable()
			Expect(newTable.Add(new)).To(Succeed())

			td := diff.DiffStructs(old, new, oldTable, newTable)
			Expect(td.Changed).To(BeTrue())
		})

		It("Should detect inherited field change via extends", func() {
			oldParent := resolution.Type{
				Name:          "Base",
				QualifiedName: "Base",
				Form: resolution.StructForm{
					Fields: []resolution.Field{makeField("id", "int32")},
				},
			}
			old := resolution.Type{
				Name:          "Entry",
				QualifiedName: "Entry",
				Form: resolution.StructForm{
					Fields:  []resolution.Field{makeField("name", "string")},
					Extends: []resolution.TypeRef{{Name: "Base"}},
				},
			}
			newParent := resolution.Type{
				Name:          "Base",
				QualifiedName: "Base",
				Form: resolution.StructForm{
					Fields: []resolution.Field{makeField("id", "int64")},
				},
			}
			newEntry := resolution.Type{
				Name:          "Entry",
				QualifiedName: "Entry",
				Form: resolution.StructForm{
					Fields:  []resolution.Field{makeField("name", "string")},
					Extends: []resolution.TypeRef{{Name: "Base"}},
				},
			}
			oldTable := resolution.NewTable()
			Expect(oldTable.Add(oldParent)).To(Succeed())
			Expect(oldTable.Add(old)).To(Succeed())
			newTable := resolution.NewTable()
			Expect(newTable.Add(newParent)).To(Succeed())
			Expect(newTable.Add(newEntry)).To(Succeed())

			td := diff.DiffStructs(old, newEntry, oldTable, newTable)
			Expect(td.Changed).To(BeTrue())
		})
	})

	Describe("DiffTables", func() {
		It("Should return empty for identical tables", func() {
			old := resolution.NewTable()
			Expect(old.Add(makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
			}))).To(Succeed())
			new := resolution.NewTable()
			Expect(new.Add(makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
			}))).To(Succeed())

			diffs := diff.DiffTables(old, new)
			Expect(diffs).To(BeEmpty())
		})

		It("Should detect type removed", func() {
			old := resolution.NewTable()
			Expect(old.Add(makeStruct("Entry", []resolution.Field{
				makeField("name", "string"),
			}))).To(Succeed())
			new := resolution.NewTable()

			diffs := diff.DiffTables(old, new)
			Expect(diffs).To(HaveLen(1))
			Expect(diffs[0].TypeName).To(Equal("Entry"))
			Expect(diffs[0].Changed).To(BeTrue())
		})

		It("Should report mix of changed and unchanged", func() {
			old := resolution.NewTable()
			Expect(old.Add(makeStruct("A", []resolution.Field{
				makeField("x", "int32"),
			}))).To(Succeed())
			Expect(old.Add(makeStruct("B", []resolution.Field{
				makeField("y", "string"),
			}))).To(Succeed())

			new := resolution.NewTable()
			Expect(new.Add(makeStruct("A", []resolution.Field{
				makeField("x", "int64"),
			}))).To(Succeed())
			Expect(new.Add(makeStruct("B", []resolution.Field{
				makeField("y", "string"),
			}))).To(Succeed())

			diffs := diff.DiffTables(old, new)
			Expect(diffs).To(HaveLen(1))
			Expect(diffs[0].TypeName).To(Equal("A"))
		})
	})
})
