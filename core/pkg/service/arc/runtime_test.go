// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Runtime", func() {
	Describe("handleChange behavior on delete", func() {
		It("Should not write a Stopped status when an arc is deleted", func() {
			a := arc.Arc{
				Key:    uuid.New(),
				Name:   "runtime-test-arc",
				Graph:  graph.Graph{},
				Text:   text.Text{},
				Deploy: false,
			}

			commitTx := db.OpenTx()
			Expect(svc.NewWriter(commitTx).Create(ctx, &a)).To(Succeed())
			Expect(commitTx.Commit(ctx)).To(Succeed())

			Eventually(func() xstatus.Variant {
				var s status.Status[arc.StatusDetails]
				if err := gorp.NewRetrieve[string, status.Status[arc.StatusDetails]]().
					WhereKeys(a.Key.String()).
					Entry(&s).
					Exec(ctx, db); err != nil {
					return ""
				}
				return s.Variant
			}).Should(Equal(xstatus.LoadingVariant))

			deleteTx := db.OpenTx()
			Expect(svc.NewWriter(deleteTx).Delete(ctx, a.Key)).To(Succeed())
			Expect(deleteTx.Commit(ctx)).To(Succeed())

			Eventually(func() error {
				return status.NewRetrieve[arc.StatusDetails](statusSvc).
					WhereKeys(a.Key.String()).
					Exec(ctx, db)
			}).Should(HaveOccurredAs(query.NotFound))
		})

		It("Should properly clean up status when deployed arc is deleted", func() {
			a := arc.Arc{
				Key:    uuid.New(),
				Name:   "deployed-arc-delete-test",
				Graph:  graph.Graph{},
				Text:   text.Text{},
				Deploy: true,
			}

			commitTx := db.OpenTx()
			Expect(svc.NewWriter(commitTx).Create(ctx, &a)).To(Succeed())
			Expect(commitTx.Commit(ctx)).To(Succeed())

			Eventually(func() bool {
				var s status.Status[arc.StatusDetails]
				if err := gorp.NewRetrieve[string, status.Status[arc.StatusDetails]]().
					WhereKeys(a.Key.String()).
					Entry(&s).
					Exec(ctx, db); err != nil {
					return false
				}
				return s.Variant == xstatus.LoadingVariant ||
					s.Variant == xstatus.VariantError ||
					s.Variant == xstatus.VariantSuccess
			}).Should(BeTrue())

			deleteTx := db.OpenTx()
			Expect(svc.NewWriter(deleteTx).Delete(ctx, a.Key)).To(Succeed())
			Expect(deleteTx.Commit(ctx)).To(Succeed())

			Eventually(func() error {
				return status.NewRetrieve[arc.StatusDetails](statusSvc).
					WhereKeys(a.Key.String()).
					Exec(ctx, db)
			}).Should(HaveOccurredAs(query.NotFound))
		})

		It("Should write Stopped status when arc is updated not deleted", func() {
			a := arc.Arc{
				Key:    uuid.New(),
				Name:   "update-test-arc",
				Graph:  graph.Graph{},
				Text:   text.Text{},
				Deploy: true,
			}

			commitTx := db.OpenTx()
			Expect(svc.NewWriter(commitTx).Create(ctx, &a)).To(Succeed())
			Expect(commitTx.Commit(ctx)).To(Succeed())

			Eventually(func() bool {
				var s status.Status[arc.StatusDetails]
				err := gorp.NewRetrieve[string, status.Status[arc.StatusDetails]]().
					WhereKeys(a.Key.String()).
					Entry(&s).
					Exec(ctx, db)
				return err == nil
			}).Should(BeTrue())

			a.Deploy = false
			updateTx := db.OpenTx()
			Expect(svc.NewWriter(updateTx).Create(ctx, &a)).To(Succeed())
			Expect(updateTx.Commit(ctx)).To(Succeed())

			Eventually(func() string {
				var s status.Status[arc.StatusDetails]
				if err := gorp.NewRetrieve[string, status.Status[arc.StatusDetails]]().
					WhereKeys(a.Key.String()).
					Entry(&s).
					Exec(ctx, db); err != nil {
					return ""
				}
				return s.Key
			}).Should(Equal(a.Key.String()))

			cleanupTx := db.OpenTx()
			Expect(svc.NewWriter(cleanupTx).Delete(ctx, a.Key)).To(Succeed())
			Expect(cleanupTx.Commit(ctx)).To(Succeed())
		})
	})
})
