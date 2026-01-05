// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framework_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/resolution"
)

func TestFramework(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Framework Suite")
}

var _ = Describe("MergeTypes", func() {
	It("Should combine two slices without duplicates by QualifiedName", func() {
		a := []resolution.Type{
			{Name: "TypeA", QualifiedName: "pkg.TypeA"},
			{Name: "TypeB", QualifiedName: "pkg.TypeB"},
		}
		b := []resolution.Type{
			{Name: "TypeB", QualifiedName: "pkg.TypeB"}, // Duplicate
			{Name: "TypeC", QualifiedName: "pkg.TypeC"},
		}

		result := framework.MergeTypes(a, b)

		Expect(result).To(HaveLen(3))
		names := make([]string, len(result))
		for i, t := range result {
			names[i] = t.Name
		}
		Expect(names).To(Equal([]string{"TypeA", "TypeB", "TypeC"}))
	})

	It("Should preserve order from first slice", func() {
		a := []resolution.Type{
			{Name: "Zebra", QualifiedName: "pkg.Zebra"},
			{Name: "Apple", QualifiedName: "pkg.Apple"},
		}
		b := []resolution.Type{
			{Name: "Mango", QualifiedName: "pkg.Mango"},
		}

		result := framework.MergeTypes(a, b)

		Expect(result).To(HaveLen(3))
		Expect(result[0].Name).To(Equal("Zebra"))
		Expect(result[1].Name).To(Equal("Apple"))
		Expect(result[2].Name).To(Equal("Mango"))
	})

	It("Should handle empty first slice", func() {
		a := []resolution.Type{}
		b := []resolution.Type{
			{Name: "TypeA", QualifiedName: "pkg.TypeA"},
		}

		result := framework.MergeTypes(a, b)

		Expect(result).To(HaveLen(1))
		Expect(result[0].Name).To(Equal("TypeA"))
	})

	It("Should handle empty second slice", func() {
		a := []resolution.Type{
			{Name: "TypeA", QualifiedName: "pkg.TypeA"},
		}
		b := []resolution.Type{}

		result := framework.MergeTypes(a, b)

		Expect(result).To(HaveLen(1))
		Expect(result[0].Name).To(Equal("TypeA"))
	})

	It("Should not modify original slices", func() {
		a := []resolution.Type{
			{Name: "TypeA", QualifiedName: "pkg.TypeA"},
		}
		b := []resolution.Type{
			{Name: "TypeB", QualifiedName: "pkg.TypeB"},
		}

		originalALen := len(a)
		originalBLen := len(b)

		_ = framework.MergeTypes(a, b)

		Expect(len(a)).To(Equal(originalALen))
		Expect(len(b)).To(Equal(originalBLen))
	})
})

var _ = Describe("MergeTypesByName", func() {
	It("Should deduplicate by Name instead of QualifiedName", func() {
		a := []resolution.Type{
			{Name: "TypeA", QualifiedName: "pkg1.TypeA"},
		}
		b := []resolution.Type{
			{Name: "TypeA", QualifiedName: "pkg2.TypeA"}, // Same Name, different QualifiedName
			{Name: "TypeB", QualifiedName: "pkg2.TypeB"},
		}

		result := framework.MergeTypesByName(a, b)

		Expect(result).To(HaveLen(2))
		Expect(result[0].QualifiedName).To(Equal("pkg1.TypeA")) // Keeps first
		Expect(result[1].Name).To(Equal("TypeB"))
	})
})

var _ = Describe("Collector", func() {
	var (
		table *resolution.Table
		req   *plugin.Request
	)

	BeforeEach(func() {
		table = resolution.NewTable()
		req = &plugin.Request{
			Resolutions: table,
		}
	})

	Describe("NewCollector", func() {
		It("Should create an empty collector", func() {
			c := framework.NewCollector("go", req)
			Expect(c.Empty()).To(BeTrue())
			Expect(c.Count()).To(Equal(0))
			Expect(c.Paths()).To(BeEmpty())
		})
	})

	Describe("Add", func() {
		It("Should add types with output paths", func() {
			c := framework.NewCollector("go", req)

			typ := resolution.Type{
				Name:          "MyType",
				QualifiedName: "test.MyType",
				Domains: map[string]resolution.Domain{
					"go": {
						Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
						},
					},
				},
			}
			Expect(c.Add(typ)).To(Succeed())
			Expect(c.Empty()).To(BeFalse())
			Expect(c.Count()).To(Equal(1))
			Expect(c.Has("pkg/types.go")).To(BeTrue())
			Expect(c.Get("pkg/types.go")).To(HaveLen(1))
			Expect(c.Get("pkg/types.go")[0].Name).To(Equal("MyType"))
		})

		It("Should skip types without output paths", func() {
			c := framework.NewCollector("go", req)

			typ := resolution.Type{
				Name:          "NoOutput",
				QualifiedName: "test.NoOutput",
				Domains:       map[string]resolution.Domain{},
			}
			Expect(c.Add(typ)).To(Succeed())
			Expect(c.Empty()).To(BeTrue())
		})

		It("Should maintain path order", func() {
			c := framework.NewCollector("go", req)

			types := []resolution.Type{
				{
					Name: "TypeA",
					Domains: map[string]resolution.Domain{
						"go": {Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "path/a"}}},
						}},
					},
				},
				{
					Name: "TypeB",
					Domains: map[string]resolution.Domain{
						"go": {Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "path/b"}}},
						}},
					},
				},
				{
					Name: "TypeC",
					Domains: map[string]resolution.Domain{
						"go": {Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "path/a"}}}, // Same as first
						}},
					},
				},
			}

			for _, typ := range types {
				Expect(c.Add(typ)).To(Succeed())
			}

			Expect(c.Paths()).To(Equal([]string{"path/a", "path/b"}))
			Expect(c.Get("path/a")).To(HaveLen(2))
		})
	})

	Describe("Remove", func() {
		It("Should remove path and return its types", func() {
			c := framework.NewCollector("go", req)

			typ := resolution.Type{
				Name: "TypeA",
				Domains: map[string]resolution.Domain{
					"go": {Expressions: []resolution.Expression{
						{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "pkg/types.go"}}},
					}},
				},
			}
			Expect(c.Add(typ)).To(Succeed())

			removed := c.Remove("pkg/types.go")
			Expect(removed).To(HaveLen(1))
			Expect(removed[0].Name).To(Equal("TypeA"))
			Expect(c.Has("pkg/types.go")).To(BeFalse())
		})
	})

	Describe("ForEach", func() {
		It("Should iterate in path order", func() {
			c := framework.NewCollector("go", req)

			types := []resolution.Type{
				{
					Name: "TypeA",
					Domains: map[string]resolution.Domain{
						"go": {Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "path/first"}}},
						}},
					},
				},
				{
					Name: "TypeB",
					Domains: map[string]resolution.Domain{
						"go": {Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "path/second"}}},
						}},
					},
				},
			}

			for _, typ := range types {
				Expect(c.Add(typ)).To(Succeed())
			}

			var paths []string
			err := c.ForEach(func(path string, types []resolution.Type) error {
				paths = append(paths, path)
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(Equal([]string{"path/first", "path/second"}))
		})

		It("Should skip removed paths", func() {
			c := framework.NewCollector("go", req)

			types := []resolution.Type{
				{
					Name: "TypeA",
					Domains: map[string]resolution.Domain{
						"go": {Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "path/first"}}},
						}},
					},
				},
				{
					Name: "TypeB",
					Domains: map[string]resolution.Domain{
						"go": {Expressions: []resolution.Expression{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "path/second"}}},
						}},
					},
				},
			}

			for _, typ := range types {
				Expect(c.Add(typ)).To(Succeed())
			}

			c.Remove("path/first")

			var paths []string
			err := c.ForEach(func(path string, types []resolution.Type) error {
				paths = append(paths, path)
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(Equal([]string{"path/second"}))
		})
	})

	Describe("WithPathFunc", func() {
		It("Should use custom path function", func() {
			c := framework.NewCollector("go", req).WithPathFunc(func(typ resolution.Type) string {
				return "custom/" + typ.Name
			})

			typ := resolution.Type{Name: "MyType"}
			Expect(c.Add(typ)).To(Succeed())
			Expect(c.Has("custom/MyType")).To(BeTrue())
		})
	})
})
