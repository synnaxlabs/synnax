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
		table := resolution.NewTable()
		structs := []resolution.Type{{
			Name:          "Test",
			QualifiedName: "test.Test",
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name: "value",
					Type: resolution.TypeRef{Name: "string"},
				}},
			},
		}}
		Expect(enum.CollectReferenced(structs, table)).To(BeEmpty())
	})

	It("should collect enums from struct fields", func() {
		table := resolution.NewTable()
		taskStateType := resolution.Type{
			Name:          "TaskState",
			Namespace:     "task",
			QualifiedName: "task.TaskState",
			Form: resolution.EnumForm{
				Values: []resolution.EnumValue{{Name: "active"}},
			},
		}
		table.Add(taskStateType)

		structs := []resolution.Type{{
			Name:          "Task",
			QualifiedName: "task.Task",
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name: "state",
					Type: resolution.TypeRef{Name: "task.TaskState"},
				}},
			},
		}}
		result := enum.CollectReferenced(structs, table)
		Expect(result).To(HaveLen(1))
		Expect(result[0].Name).To(Equal("TaskState"))
	})

	It("should deduplicate enums by qualified name", func() {
		table := resolution.NewTable()
		taskStateType := resolution.Type{
			Name:          "TaskState",
			Namespace:     "task",
			QualifiedName: "task.TaskState",
			Form: resolution.EnumForm{
				Values: []resolution.EnumValue{{Name: "active"}},
			},
		}
		table.Add(taskStateType)

		structs := []resolution.Type{
			{
				Name:          "Task1",
				QualifiedName: "task.Task1",
				Form: resolution.StructForm{
					Fields: []resolution.Field{{
						Name: "state",
						Type: resolution.TypeRef{Name: "task.TaskState"},
					}},
				},
			},
			{
				Name:          "Task2",
				QualifiedName: "task.Task2",
				Form: resolution.StructForm{
					Fields: []resolution.Field{{
						Name: "state",
						Type: resolution.TypeRef{Name: "task.TaskState"},
					}},
				},
			},
		}
		Expect(enum.CollectReferenced(structs, table)).To(HaveLen(1))
	})

	It("should collect multiple different enums", func() {
		table := resolution.NewTable()
		table.Add(resolution.Type{
			Name:          "TaskState",
			Namespace:     "task",
			QualifiedName: "task.TaskState",
			Form:          resolution.EnumForm{Values: []resolution.EnumValue{{Name: "active"}}},
		})
		table.Add(resolution.Type{
			Name:          "DataType",
			Namespace:     "telem",
			QualifiedName: "telem.DataType",
			Form:          resolution.EnumForm{Values: []resolution.EnumValue{{Name: "float32"}}},
		})

		structs := []resolution.Type{{
			Name:          "Record",
			QualifiedName: "test.Record",
			Form: resolution.StructForm{
				Fields: []resolution.Field{
					{Name: "state", Type: resolution.TypeRef{Name: "task.TaskState"}},
					{Name: "dataType", Type: resolution.TypeRef{Name: "telem.DataType"}},
				},
			},
		}}
		Expect(enum.CollectReferenced(structs, table)).To(HaveLen(2))
	})

	It("should handle empty structs slice", func() {
		table := resolution.NewTable()
		Expect(enum.CollectReferenced(nil, table)).To(BeEmpty())
		Expect(enum.CollectReferenced([]resolution.Type{}, table)).To(BeEmpty())
	})
})

var _ = Describe("FindOutputPath", func() {
	It("should find output path from struct in same namespace", func() {
		e := resolution.Type{Name: "TaskState", Namespace: "task", Form: resolution.EnumForm{}}
		table := resolution.NewTable()
		table.Add(resolution.Type{
			Name:          "Task",
			QualifiedName: "task.Task",
			Namespace:     "task",
			Form:          resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{
					Name:   "output",
					Values: []resolution.ExpressionValue{{StringValue: "client/ts/task"}},
				}}},
			},
		})
		Expect(enum.FindOutputPath(e, table, "ts")).To(Equal("client/ts/task"))
	})

	It("should return empty for enum with no matching namespace", func() {
		e := resolution.Type{Name: "TaskState", Namespace: "task", Form: resolution.EnumForm{}}
		table := resolution.NewTable()
		table.Add(resolution.Type{
			Name:          "Other",
			QualifiedName: "other.Other",
			Namespace:     "other",
			Form:          resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{
					Name:   "output",
					Values: []resolution.ExpressionValue{{StringValue: "client/ts/other"}},
				}}},
			},
		})
		Expect(enum.FindOutputPath(e, table, "ts")).To(BeEmpty())
	})

	It("should return empty for struct without output domain", func() {
		e := resolution.Type{Name: "TaskState", Namespace: "task", Form: resolution.EnumForm{}}
		table := resolution.NewTable()
		table.Add(resolution.Type{
			Name:          "Task",
			QualifiedName: "task.Task",
			Namespace:     "task",
			Form:          resolution.StructForm{},
			Domains:       map[string]resolution.Domain{},
		})
		Expect(enum.FindOutputPath(e, table, "ts")).To(BeEmpty())
	})

	It("should work with different domain names", func() {
		e := resolution.Type{Name: "State", Namespace: "test", Form: resolution.EnumForm{}}
		table := resolution.NewTable()
		table.Add(resolution.Type{
			Name:          "Test",
			QualifiedName: "test.Test",
			Namespace:     "test",
			Form:          resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{{
					Name:   "output",
					Values: []resolution.ExpressionValue{{StringValue: "core/test"}},
				}}},
				"py": {Expressions: []resolution.Expression{{
					Name:   "output",
					Values: []resolution.ExpressionValue{{StringValue: "client/py/test"}},
				}}},
			},
		})
		Expect(enum.FindOutputPath(e, table, "go")).To(Equal("core/test"))
		Expect(enum.FindOutputPath(e, table, "py")).To(Equal("client/py/test"))
		Expect(enum.FindOutputPath(e, table, "ts")).To(BeEmpty())
	})
})
