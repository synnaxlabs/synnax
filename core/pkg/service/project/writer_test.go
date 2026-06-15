// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/project"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a project", func(ctx SpecContext) {
			proj := project.Project{
				Name:   "test",
				Author: author.Key,
				Layout: map[string]any{"key": "data"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			Expect(proj.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("CreateMany", func() {
		It("Should create multiple projects", func(ctx SpecContext) {
			projects := []project.Project{
				{Name: "project-1", Author: author.Key},
				{Name: "project-2", Author: author.Key},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &projects)).To(Succeed())

			var retrieved []project.Project
			Expect(svc.NewRetrieve().Where(project.MatchKeys(
				projects[0].Key,
				projects[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})
	Describe("Update", func() {
		It("Should rename a project", func(ctx SpecContext) {
			proj := project.Project{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, proj.Key, "test2")).To(Succeed())
			var res project.Project
			Expect(svc.NewRetrieve().Where(project.MatchKeys(proj.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetLayout", func() {
		It("Should set the layout of a project", func(ctx SpecContext) {
			proj := project.Project{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			Expect(svc.NewWriter(tx).SetLayout(ctx, proj.Key, map[string]any{"key": "data"})).To(Succeed())
			var res project.Project
			Expect(svc.NewRetrieve().Where(project.MatchKeys(proj.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Layout["key"]).To(Equal("data"))
		})
	})
	Describe("DeleteChannel", func() {
		It("Should delete a project", func(ctx SpecContext) {
			proj := project.Project{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, proj.Key)).To(Succeed())
			var res project.Project
			Expect(svc.NewRetrieve().Where(project.MatchKeys(proj.Key)).Entry(&res).Exec(ctx, tx)).ToNot(Succeed())
		})
	})
})
