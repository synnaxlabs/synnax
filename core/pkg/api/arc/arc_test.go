// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("api.Service.Retrieve", func() {
	It("Should resolve the task key with IncludeTask", func(ctx SpecContext) {
		a := MustSucceed(arcSvc.NewWriter(nil).Create(ctx, arc.New{
			Name: "resolve-task",
			Rack: &testRack.Key,
		}))
		grant(ctx, user.OntologyID(author.Key), access.ActionRetrieve, arc.OntologyID(a.Key))
		res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
			Keys:        []arc.Key{a.Key},
			IncludeTask: true,
		}))
		Expect(res.Arcs).To(HaveLen(1))
		Expect(res.Arcs[0].Task).ToNot(BeNil())
		Expect(*res.Arcs[0].Task).To(Equal(*a.Task))
	})

	It("Should leave the task nil for an undeployed arc", func(ctx SpecContext) {
		a := MustSucceed(arcSvc.NewWriter(nil).Create(ctx, arc.New{Name: "resolve-none"}))
		grant(ctx, user.OntologyID(author.Key), access.ActionRetrieve, arc.OntologyID(a.Key))
		res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
			Keys:        []arc.Key{a.Key},
			IncludeTask: true,
		}))
		Expect(res.Arcs).To(HaveLen(1))
		Expect(res.Arcs[0].Task).To(BeNil())
	})

	It("Should resolve the task status with IncludeStatus", func(ctx SpecContext) {
		a := MustSucceed(arcSvc.NewWriter(nil).Create(ctx, arc.New{
			Name: "resolve-status",
			Rack: &testRack.Key,
		}))
		grant(ctx, user.OntologyID(author.Key), access.ActionRetrieve, arc.OntologyID(a.Key))
		res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
			Keys:          []arc.Key{a.Key},
			IncludeStatus: true,
		}))
		Expect(res.Arcs).To(HaveLen(1))
		Expect(res.Arcs[0].Task).ToNot(BeNil())
		Expect(res.Arcs[0].Status).ToNot(BeNil())
	})
})
