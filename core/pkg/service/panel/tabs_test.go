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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Tabs", func() {
	retrieve := func(ctx context.Context, db *gorp.DB, key panel.Key) panel.Panel {
		GinkgoHelper()
		var res panel.Panel
		Expect(gorp.NewRetrieve[panel.Key, panel.Panel]().
			Where(gorp.MatchKeys[panel.Key, panel.Panel](key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	It("Should keep the tab of a deleted line plot", func(ctx SpecContext) {
		deleted := uuid.New()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			createPlot(ctx, tx, deleted)
			return nil
		})).To(Succeed())
		p := panel.Panel{
			Name:    "deleted",
			Root:    leafNode(tab(deleted), viewTab(uuid.New(), "docs")),
			Project: &proj.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &p)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, p.Key)
			})).To(Succeed())
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return plotSvc.NewWriter(tx).Delete(ctx, deleted)
		})).To(Succeed())
		leaf := MustBeOk(asLeaf(retrieve(ctx, db, p.Key).Root))
		Expect(leaf.Tabs).To(HaveLen(2))
		Expect(leaf.Tabs[0].Key()).To(Equal(deleted))
	})
})
