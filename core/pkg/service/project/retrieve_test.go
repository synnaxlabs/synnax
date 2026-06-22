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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/project"
)

var _ = Describe("Retrieve", func() {
	Describe("By Keys", func() {
		It("Should retrieve only the projects matching the given keys", func(ctx SpecContext) {
			a := project.Project{Name: "a"}
			b := project.Project{Name: "b"}
			c := project.Project{Name: "c"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			var res []project.Project
			Expect(svc.NewRetrieve().Where(project.MatchKeys(a.Key, c.Key)).
				Entries(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(ConsistOf(a, c))
		})
	})

	Describe("Limit and Offset", func() {
		It("Should bound and page the result set", func(ctx SpecContext) {
			keys := make([]project.Key, 3)
			for i := range keys {
				p := project.Project{Name: "page"}
				Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
				keys[i] = p.Key
			}
			var limited []project.Project
			Expect(svc.NewRetrieve().Where(project.MatchKeys(keys...)).Limit(2).
				Entries(&limited).Exec(ctx, tx)).To(Succeed())
			Expect(limited).To(HaveLen(2))

			var offset []project.Project
			Expect(svc.NewRetrieve().Where(project.MatchKeys(keys...)).Offset(2).
				Entries(&offset).Exec(ctx, tx)).To(Succeed())
			Expect(offset).To(HaveLen(1))
		})
	})
})
