// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enum_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("CollectReferenced", func() {
	It("should return empty for structs without enum fields", func() {
		structs := []*resolution.StructEntry{{
			Fields: []*resolution.FieldEntry{{
				TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive},
			}},
		}}
		Expect(enum.CollectReferenced(structs)).To(BeEmpty())
	})

	It("should collect enums from struct fields", func() {
		taskState := &resolution.EnumEntry{Name: "TaskState", QualifiedName: "task.TaskState"}
		structs := []*resolution.StructEntry{{
			Fields: []*resolution.FieldEntry{{
				TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindEnum, EnumRef: taskState},
			}},
		}}
		result := enum.CollectReferenced(structs)
		Expect(result).To(HaveLen(1))
		Expect(result[0]).To(Equal(taskState))
	})

	It("should deduplicate enums by qualified name", func() {
		taskState := &resolution.EnumEntry{Name: "TaskState", QualifiedName: "task.TaskState"}
		structs := []*resolution.StructEntry{
			{Fields: []*resolution.FieldEntry{{
				TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindEnum, EnumRef: taskState},
			}}},
			{Fields: []*resolution.FieldEntry{{
				TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindEnum, EnumRef: taskState},
			}}},
		}
		Expect(enum.CollectReferenced(structs)).To(HaveLen(1))
	})

	It("should collect multiple different enums", func() {
		taskState := &resolution.EnumEntry{Name: "TaskState", QualifiedName: "task.TaskState"}
		dataType := &resolution.EnumEntry{Name: "DataType", QualifiedName: "telem.DataType"}
		structs := []*resolution.StructEntry{{
			Fields: []*resolution.FieldEntry{
				{TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindEnum, EnumRef: taskState}},
				{TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindEnum, EnumRef: dataType}},
			},
		}}
		Expect(enum.CollectReferenced(structs)).To(HaveLen(2))
	})

	It("should skip fields with nil EnumRef", func() {
		structs := []*resolution.StructEntry{{
			Fields: []*resolution.FieldEntry{{
				TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindEnum, EnumRef: nil},
			}},
		}}
		Expect(enum.CollectReferenced(structs)).To(BeEmpty())
	})

	It("should handle empty structs slice", func() {
		Expect(enum.CollectReferenced(nil)).To(BeEmpty())
		Expect(enum.CollectReferenced([]*resolution.StructEntry{})).To(BeEmpty())
	})
})

var _ = Describe("FindOutputPath", func() {
	It("should find output path from struct in same namespace", func() {
		e := &resolution.EnumEntry{Name: "TaskState", Namespace: "task"}
		table := &resolution.Table{
			Structs: []*resolution.StructEntry{
				{
					Name:          "Task",
					QualifiedName: "task.Task",
					Namespace:     "task",
					Domains: map[string]*resolution.DomainEntry{
						"ts": {Expressions: []*resolution.ExpressionEntry{{
							Name:   "output",
							Values: []resolution.ExpressionValue{{StringValue: "client/ts/task"}},
						}}},
					},
				},
			},
		}
		Expect(enum.FindOutputPath(e, table, "ts")).To(Equal("client/ts/task"))
	})

	It("should return empty for enum with no matching namespace", func() {
		e := &resolution.EnumEntry{Name: "TaskState", Namespace: "task"}
		table := &resolution.Table{
			Structs: []*resolution.StructEntry{
				{
					Name:          "Other",
					QualifiedName: "other.Other",
					Namespace:     "other",
					Domains: map[string]*resolution.DomainEntry{
						"ts": {Expressions: []*resolution.ExpressionEntry{{
							Name:   "output",
							Values: []resolution.ExpressionValue{{StringValue: "client/ts/other"}},
						}}},
					},
				},
			},
		}
		Expect(enum.FindOutputPath(e, table, "ts")).To(BeEmpty())
	})

	It("should return empty for struct without output domain", func() {
		e := &resolution.EnumEntry{Name: "TaskState", Namespace: "task"}
		table := &resolution.Table{
			Structs: []*resolution.StructEntry{
				{
					Name:          "Task",
					QualifiedName: "task.Task",
					Namespace:     "task",
					Domains:       map[string]*resolution.DomainEntry{},
				},
			},
		}
		Expect(enum.FindOutputPath(e, table, "ts")).To(BeEmpty())
	})

	It("should work with different domain names", func() {
		e := &resolution.EnumEntry{Name: "State", Namespace: "test"}
		table := &resolution.Table{
			Structs: []*resolution.StructEntry{
				{
					Name:          "Test",
					QualifiedName: "test.Test",
					Namespace:     "test",
					Domains: map[string]*resolution.DomainEntry{
						"go": {Expressions: []*resolution.ExpressionEntry{{
							Name:   "output",
							Values: []resolution.ExpressionValue{{StringValue: "core/test"}},
						}}},
						"py": {Expressions: []*resolution.ExpressionEntry{{
							Name:   "output",
							Values: []resolution.ExpressionValue{{StringValue: "client/py/test"}},
						}}},
					},
				},
			},
		}
		Expect(enum.FindOutputPath(e, table, "go")).To(Equal("core/test"))
		Expect(enum.FindOutputPath(e, table, "py")).To(Equal("client/py/test"))
		Expect(enum.FindOutputPath(e, table, "ts")).To(BeEmpty())
	})
})
