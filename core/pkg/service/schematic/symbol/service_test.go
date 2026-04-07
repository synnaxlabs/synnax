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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("OpenService", func() {
		It("Should create a service with minimal configuration", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			testOtg := MustSucceed(ontology.Open(ctx, ontology.Config{
				DB: testDB,
			}))
			testSearchIdx := MustSucceed(search.Open())

			testSvc := MustSucceed(symbol.OpenService(ctx, symbol.ServiceConfig{
				DB:       testDB,
				Ontology: testOtg,
				Search:   testSearchIdx,
			}))
			Expect(testSvc).ToNot(BeNil())

			Expect(testSvc.Close()).To(Succeed())
			Expect(testOtg.Close()).To(Succeed())
			Expect(testDB.Close()).To(Succeed())
		})

		It("Should create a service with group configuration", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			testOtg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: testDB}))
			testSearchIdx := MustSucceed(search.Open())
			DeferCleanup(func() {
				Expect(testSearchIdx.Close()).To(Succeed())
			})
			testGroup := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
				DB:       testDB,
				Ontology: testOtg,
				Search:   testSearchIdx,
			}))

			testSvc := MustSucceed(symbol.OpenService(ctx, symbol.ServiceConfig{
				DB:       testDB,
				Ontology: testOtg,
				Group:    testGroup,
				Search:   testSearchIdx,
			}))
			Expect(testSvc).ToNot(BeNil())
			Expect(testSvc.Group()).ToNot(BeNil())
			Expect(testSvc.Group().Name).To(Equal("Schematic Symbols"))

			Expect(testSvc.Close()).To(Succeed())
			Expect(testOtg.Close()).To(Succeed())
			Expect(testDB.Close()).To(Succeed())
		})

		It("Should fail with invalid configuration", func(ctx SpecContext) {
			_, err := symbol.OpenService(ctx, symbol.ServiceConfig{
				DB: nil,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("db: must be non-nil"))

			Expect(symbol.OpenService(ctx, symbol.ServiceConfig{Ontology: otg})).
				Error().To(MatchError(ContainSubstring("db")))
		})

		It("Should handle configuration override correctly", func(ctx SpecContext) {
			testDB1 := gorp.Wrap(memkv.New())
			testDB2 := gorp.Wrap(memkv.New())
			testOtg1 := MustSucceed(ontology.Open(ctx, ontology.Config{
				DB: testDB1,
			}))
			testOtg2 := MustSucceed(ontology.Open(ctx, ontology.Config{
				DB: testDB2,
			}))
			testSearchIdx := MustSucceed(search.Open())

			cfg1 := symbol.ServiceConfig{
				DB:       testDB1,
				Ontology: testOtg1,
				Search:   testSearchIdx,
			}
			cfg2 := symbol.ServiceConfig{
				DB:       testDB2,
				Ontology: testOtg2,
				Search:   testSearchIdx,
			}

			testSvc := MustSucceed(symbol.OpenService(ctx, cfg1, cfg2))
			// Should use cfg2's values
			Expect(testSvc.ServiceConfig.DB).To(Equal(testDB2))
			Expect(testSvc.ServiceConfig.Ontology).To(Equal(testOtg2))

			Expect(testSvc.Close()).To(Succeed())
			Expect(testOtg1.Close()).To(Succeed())
			Expect(testOtg2.Close()).To(Succeed())
			Expect(testDB1.Close()).To(Succeed())
			Expect(testDB2.Close()).To(Succeed())
		})
	})

	Describe("NewWriter", func() {
		It("Should create a writer with transaction", func(ctx SpecContext) {
			writer := svc.NewWriter(tx)
			Expect(writer).ToNot(BeNil())
		})

		It("Should create a writer without transaction", func(ctx SpecContext) {
			writer := svc.NewWriter(nil)
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
			testDB := gorp.Wrap(memkv.New())
			testOtg := MustSucceed(ontology.Open(ctx, ontology.Config{
				DB: testDB,
			}))
			testSearchIdx := MustSucceed(search.Open())

			testSvc := MustSucceed(symbol.OpenService(ctx, symbol.ServiceConfig{
				DB:       testDB,
				Ontology: testOtg,
				Search:   testSearchIdx,
			}))

			Expect(testSvc.Close()).To(Succeed())
			// Should be idempotent
			Expect(testSvc.Close()).To(Succeed())

			Expect(testOtg.Close()).To(Succeed())
			Expect(testDB.Close()).To(Succeed())
		})
	})

	Describe("Integration", func() {
		It("Should handle concurrent operations", func(ctx SpecContext) {
			done := make(chan bool, 3)

			// Writer goroutine
			go func() {
				defer GinkgoRecover()
				for range 10 {
					sym := symbol.Symbol{
						Name: "concurrent-write",
						Data: map[string]any{"svg": "<svg>...</svg>"},
					}
					Expect(svc.NewWriter(nil).Create(ctx, &sym, ws.OntologyID())).To(Succeed())
					Expect(svc.NewWriter(nil).Delete(ctx, sym.Key)).To(Succeed())
				}
				done <- true
			}()

			// Reader goroutine 1
			go func() {
				defer GinkgoRecover()
				for range 10 {
					var symbols []symbol.Symbol
					_ = svc.NewRetrieve().Entries(&symbols).Exec(ctx, nil)
				}
				done <- true
			}()

			// Reader goroutine 2
			go func() {
				defer GinkgoRecover()
				for range 10 {
					var symbols []symbol.Symbol
					_ = svc.NewRetrieve().Entries(&symbols).Exec(ctx, nil)
				}
				done <- true
			}()

			// Wait for all goroutines
			for range 3 {
				Eventually(done, "5s").Should(Receive())
			}
		})
	})
})
