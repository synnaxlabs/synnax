// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Service", func() {
	Describe("Observe", func() {
		// The suite holds the write lock of db for each spec, so commits go to a DB of
		// their own.
		var (
			obsDB  *gorp.DB
			obsSvc *icd.Service
		)
		BeforeEach(func(ctx SpecContext) {
			obsDB = DeferClose(gorp.Wrap(memkv.New()))
			obsSvc = MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
				DB:   obsDB,
				ImEx: imex.NewService(),
			}))
		})

		It("Should notify an observer of each committed change to an ICD",
			func(ctx SpecContext) {
				changes := make(chan gorp.EntryChange[icd.Key, icd.ICD], 3)
				DeferCleanup(obsSvc.Observe().OnChange(
					func(_ context.Context, r gorp.TxReader[icd.Key, icd.ICD]) {
						for c := range r {
							changes <- c
						}
					},
				))
				ic := icd.ICD{Name: "Bus"}
				commit := func(write func(icd.Writer) error) {
					GinkgoHelper()
					Expect(obsDB.WithTx(ctx, func(tx gorp.Tx) error {
						return write(obsSvc.NewWriter(tx))
					})).To(Succeed())
				}
				commit(func(w icd.Writer) error { return w.Create(ctx, &ic) })
				var c gorp.EntryChange[icd.Key, icd.ICD]
				Eventually(changes).Should(Receive(&c))
				Expect(c.Key).To(Equal(ic.Key))
				Expect(c.Kind).To(Equal(gorp.ChangeCreate))
				Expect(c.Value.Name).To(Equal("Bus"))
				commit(func(w icd.Writer) error { return w.Delete(ctx, ic.Key) })
				Eventually(changes).Should(Receive(&c))
				Expect(c.Key).To(Equal(ic.Key))
				Expect(c.Kind).To(Equal(gorp.ChangeDelete))
			},
		)
	})
})
