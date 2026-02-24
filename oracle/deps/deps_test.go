// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deps_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/deps"
	"github.com/synnaxlabs/oracle/resolution"
)

func gorpStruct(name string, fields []resolution.Field) resolution.Type {
	return resolution.Type{
		Name:          name,
		QualifiedName: name,
		Domains:       map[string]resolution.Domain{"key": {}},
		Form: resolution.StructForm{
			HasKeyDomain: true,
			Fields:       fields,
		},
	}
}

func plainStruct(name string, fields []resolution.Field) resolution.Type {
	return resolution.Type{
		Name:          name,
		QualifiedName: name,
		Form:          resolution.StructForm{Fields: fields},
	}
}

func field(name, typeName string) resolution.Field {
	return resolution.Field{Name: name, Type: resolution.TypeRef{Name: typeName}}
}

var _ = Describe("Deps", func() {
	Describe("AffectedEntries", func() {
		It("Should detect a direct dependency", func() {
			table := resolution.NewTable()
			Expect(table.Add(plainStruct("Config", []resolution.Field{
				field("value", "int32"),
			}))).To(Succeed())
			Expect(table.Add(gorpStruct("Entry", []resolution.Field{
				field("config", "Config"),
			}))).To(Succeed())

			g := deps.Build(table)
			affected := g.AffectedEntries([]string{"Config"})
			Expect(affected).To(ConsistOf("Entry"))
		})

		It("Should detect transitive dependencies", func() {
			table := resolution.NewTable()
			Expect(table.Add(plainStruct("Inner", []resolution.Field{
				field("x", "int32"),
			}))).To(Succeed())
			Expect(table.Add(plainStruct("Middle", []resolution.Field{
				field("inner", "Inner"),
			}))).To(Succeed())
			Expect(table.Add(gorpStruct("Outer", []resolution.Field{
				field("middle", "Middle"),
			}))).To(Succeed())

			g := deps.Build(table)
			affected := g.AffectedEntries([]string{"Inner"})
			Expect(affected).To(ConsistOf("Outer"))
		})

		It("Should detect shared dependencies", func() {
			table := resolution.NewTable()
			Expect(table.Add(plainStruct("Shared", []resolution.Field{
				field("v", "string"),
			}))).To(Succeed())
			Expect(table.Add(gorpStruct("A", []resolution.Field{
				field("s", "Shared"),
			}))).To(Succeed())
			Expect(table.Add(gorpStruct("B", []resolution.Field{
				field("s", "Shared"),
			}))).To(Succeed())

			g := deps.Build(table)
			affected := g.AffectedEntries([]string{"Shared"})
			Expect(affected).To(ConsistOf("A", "B"))
		})

		It("Should not include independent types", func() {
			table := resolution.NewTable()
			Expect(table.Add(gorpStruct("A", []resolution.Field{
				field("x", "int32"),
			}))).To(Succeed())
			Expect(table.Add(gorpStruct("B", []resolution.Field{
				field("y", "string"),
			}))).To(Succeed())

			g := deps.Build(table)
			affected := g.AffectedEntries([]string{"A"})
			Expect(affected).To(ConsistOf("A"))
		})

		It("Should only return gorp entries (HasKeyDomain)", func() {
			table := resolution.NewTable()
			Expect(table.Add(plainStruct("Leaf", []resolution.Field{
				field("v", "int32"),
			}))).To(Succeed())
			Expect(table.Add(plainStruct("NonGorp", []resolution.Field{
				field("leaf", "Leaf"),
			}))).To(Succeed())
			Expect(table.Add(gorpStruct("GorpEntry", []resolution.Field{
				field("leaf", "Leaf"),
			}))).To(Succeed())

			g := deps.Build(table)
			affected := g.AffectedEntries([]string{"Leaf"})
			Expect(affected).To(ConsistOf("GorpEntry"))
		})

		It("Should handle deep nesting (5+ levels)", func() {
			table := resolution.NewTable()
			Expect(table.Add(plainStruct("L5", []resolution.Field{
				field("v", "int32"),
			}))).To(Succeed())
			Expect(table.Add(plainStruct("L4", []resolution.Field{
				field("l5", "L5"),
			}))).To(Succeed())
			Expect(table.Add(plainStruct("L3", []resolution.Field{
				field("l4", "L4"),
			}))).To(Succeed())
			Expect(table.Add(plainStruct("L2", []resolution.Field{
				field("l3", "L3"),
			}))).To(Succeed())
			Expect(table.Add(plainStruct("L1", []resolution.Field{
				field("l2", "L2"),
			}))).To(Succeed())
			Expect(table.Add(gorpStruct("Root", []resolution.Field{
				field("l1", "L1"),
			}))).To(Succeed())

			g := deps.Build(table)
			affected := g.AffectedEntries([]string{"L5"})
			Expect(affected).To(ConsistOf("Root"))
		})

		It("Should terminate safely with cycles", func() {
			table := resolution.NewTable()
			Expect(table.Add(gorpStruct("CycleA", []resolution.Field{
				field("b", "CycleB"),
			}))).To(Succeed())
			Expect(table.Add(gorpStruct("CycleB", []resolution.Field{
				field("a", "CycleA"),
			}))).To(Succeed())

			g := deps.Build(table)
			affected := g.AffectedEntries([]string{"CycleA"})
			Expect(affected).To(ConsistOf("CycleA", "CycleB"))
		})

		It("Should include a changed type that is itself a gorp entry", func() {
			table := resolution.NewTable()
			Expect(table.Add(gorpStruct("SelfChanged", []resolution.Field{
				field("x", "int32"),
			}))).To(Succeed())

			g := deps.Build(table)
			affected := g.AffectedEntries([]string{"SelfChanged"})
			Expect(affected).To(ConsistOf("SelfChanged"))
		})
	})
})
