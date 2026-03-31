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
			p := project.Project{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(p.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Rename", func() {
		It("Should rename a project", func(ctx SpecContext) {
			p := project.Project{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, p.Key, "test2")).To(Succeed())
			var res project.Project
			Expect(svc.NewRetrieve().WhereKeys(p.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("Delete", func() {
		It("Should delete a project", func(ctx SpecContext) {
			p := project.Project{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, p.Key)).To(Succeed())
			var res project.Project
			Expect(svc.NewRetrieve().WhereKeys(p.Key).Entry(&res).Exec(ctx, tx)).ToNot(Succeed())
		})
	})
})
