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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Retrieve", func() {
	createN := func(ctx context.Context, n int) []panel.Key {
		keys := make([]panel.Key, n)
		for i := range keys {
			p := panel.Panel{Name: "test", Root: leafNode(), Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			keys[i] = p.Key
		}
		return keys
	}

	It(
		"Should retrieve a panel by key with its full tree intact",
		func(ctx SpecContext) {
			p := panel.Panel{
				Name:   "test",
				Root:   leafNode(tab(uuid.New())),
				Parent: &parentID,
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
			// Parent lives in the ontology graph, not on the record, so it is absent
			// on retrieve.
			p.Parent = nil
			Expect(res).To(Equal(p))
		},
	)

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
