// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
	createN := func(ctx context.Context, n int) []panel.Key {
		GinkgoHelper()
		keys := make([]panel.Key, n)
		for i := range keys {
			p := panel.Panel{Name: "test", Root: leafNode(), Project: &proj.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			keys[i] = p.Key
		}
		return keys
	}

	It(
		"Should retrieve a panel by key with its full tree intact",
		func(ctx SpecContext) {
			key := uuid.New()
			createPlot(ctx, tx, key)
			p := panel.Panel{
				Name:    "test",
				Root:    leafNode(tab(key)),
				Project: &proj.Key,
			}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			var res panel.Panel
			Expect(
				svc.NewRetrieve().
					Where(panel.MatchKeys(p.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).
				To(Succeed())
			Expect(res).To(Equal(p))
		},
	)

	It("Should retrieve the panels of a project", func(ctx SpecContext) {
		other := project.Project{Name: "retrieved"}
		Expect(projectSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
		p := panel.Panel{Name: "owned", Project: &other.Key}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		createN(ctx, 1)
		var res []panel.Panel
		Expect(svc.NewRetrieve().
			Where(panel.MatchProjects(other.Key)).
			Entries(&res).
			Exec(ctx, tx)).To(Succeed())
		Expect(res).To(HaveLen(1))
		Expect(res[0].Key).To(Equal(p.Key))
	})

	It("Should retrieve multiple panels via Entries", func(ctx SpecContext) {
		keys := createN(ctx, 3)
		var res []panel.Panel
		Expect(
			svc.NewRetrieve().
				Where(panel.MatchKeys(keys...)).
				Entries(&res).
				Exec(ctx, tx),
		).
			To(Succeed())
		Expect(res).To(HaveLen(3))
	})

	It("Should return ErrNotFound when the key does not exist", func(ctx SpecContext) {
		Expect(
			svc.NewRetrieve().Where(panel.MatchKeys(uuid.New())).Entry(&panel.Panel{}).
				Exec(ctx, tx),
		).To(MatchError(query.ErrNotFound))
	})

	It("Should cap the result set with Limit", func(ctx SpecContext) {
		keys := createN(ctx, 3)
		var res []panel.Panel
		Expect(svc.NewRetrieve().Where(panel.MatchKeys(keys...)).Limit(2).Entries(&res).
			Exec(ctx, tx)).To(Succeed())
		Expect(res).To(HaveLen(2))
	})

	It("Should skip results with Offset", func(ctx SpecContext) {
		keys := createN(ctx, 3)
		var res []panel.Panel
		Expect(
			svc.NewRetrieve().Where(panel.MatchKeys(keys...)).Offset(1).Entries(&res).
				Exec(ctx, tx),
		).To(Succeed())
		Expect(res).To(HaveLen(2))
	})

	It("Should count the matching panels", func(ctx SpecContext) {
		keys := createN(ctx, 3)
		Expect(
			svc.NewRetrieve().Where(panel.MatchKeys(keys...)).Count(ctx, tx),
		).To(Equal(3))
	})

	It("Should report existence", func(ctx SpecContext) {
		keys := createN(ctx, 1)
		Expect(
			svc.NewRetrieve().Where(panel.MatchKeys(keys[0])).Exists(ctx, tx),
		).To(BeTrue())
		Expect(svc.NewRetrieve().Where(panel.MatchKeys(uuid.New())).Exists(ctx, tx)).
			To(BeFalse())
	})
})
