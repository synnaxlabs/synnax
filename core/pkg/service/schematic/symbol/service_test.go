// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Service", func() {
	Describe("OpenService", func() {
		It("Should create a service", func(ctx SpecContext) {
			testDB := DeferClose(gorp.Wrap(memkv.New()))
			testGroup := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB: testDB,
			}))

			testSvc := MustOpen(symbol.OpenService(ctx, symbol.ServiceConfig{
				DB:    testDB,
				Group: testGroup,
				ImEx:  imex.NewService(),
			}))
			Expect(testSvc).ToNot(BeNil())
		})

		DescribeTable("Should fail with invalid configuration",
			func(ctx SpecContext, cfg symbol.ServiceConfig, missing string) {
				Expect(symbol.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(missing + ": must be non-nil")))
			},
			Entry("no db", symbol.ServiceConfig{DB: nil}, "db"),
			Entry("no group", symbol.ServiceConfig{DB: &gorp.DB{}}, "group"),
		)

		It("Should apply later configurations as overrides", func(ctx SpecContext) {
			testDB := DeferClose(gorp.Wrap(memkv.New()))
			testGroup := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB: testDB,
			}))
			// cfg1 omits the DB; cfg2 supplies it. OpenService merges later configs
			// over earlier ones, so the service opens only because cfg2 fills the gap.
			cfg1 := symbol.ServiceConfig{
				Group: testGroup,
				ImEx:  imex.NewService(),
			}
			cfg2 := symbol.ServiceConfig{DB: testDB}
			Expect(MustOpen(symbol.OpenService(ctx, cfg1, cfg2))).ToNot(BeNil())
		})
	})

	Describe("NewWriter", func() {
		It("Should create a writer with transaction", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			writer := svc.NewWriter(tx)
			Expect(writer).ToNot(BeNil())
		})
	})

	Describe("NewRetrieve", func() {
		It("Should create a retrieve query builder", func(ctx SpecContext) {
			retrieve := svc.NewRetrieve()
			Expect(retrieve).ToNot(BeNil())
		})
	})

	Describe("Close", func() {
		It("Should close the service cleanly", func(ctx SpecContext) {
			testDB := DeferClose(gorp.Wrap(memkv.New()))
			testGroup := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB: testDB,
			}))

			testSvc := MustOpen(symbol.OpenService(ctx, symbol.ServiceConfig{
				DB:    testDB,
				Group: testGroup,
				ImEx:  imex.NewService(),
			}))

			Expect(testSvc.Close()).To(Succeed())
			// Should be idempotent
			Expect(testSvc.Close()).To(Succeed())
		})
	})

	Describe("Integration", func() {
		It("Should handle concurrent operations", func(ctx SpecContext) {
			done := make(chan bool, 3)

			// Writer goroutine
			go func() {
				defer GinkgoRecover()
				for range 10 {
					sym := defaults.Fill(symbol.Symbol{
						Name: "concurrent-write",
						Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
					})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &sym)
					})).To(Succeed())
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, sym.Key)
					})).To(Succeed())
				}
				done <- true
			}()

			// Reader goroutine 1
			go func() {
				defer GinkgoRecover()
				for range 10 {
					var symbols []symbol.Symbol
					Expect(
						svc.NewRetrieve().Entries(&symbols).Exec(ctx, db),
					).To(Succeed())
				}
				done <- true
			}()

			// Reader goroutine 2
			go func() {
				defer GinkgoRecover()
				for range 10 {
					var symbols []symbol.Symbol
					Expect(
						svc.NewRetrieve().Entries(&symbols).Exec(ctx, db),
					).To(Succeed())
				}
				done <- true
			}()

			// Wait for all goroutines
			for range 3 {
				Eventually(done, time.Second*5).Should(Receive())
			}
		})
	})
})
