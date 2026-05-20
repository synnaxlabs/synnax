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
	It("Should retrieve a project by key", func(ctx SpecContext) {
		p := project.Project{Name: "test"}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		var res project.Project
		Expect(svc.NewRetrieve().Where(project.MatchKeys(p.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
		Expect(res.Name).To(Equal("test"))
	})
})
