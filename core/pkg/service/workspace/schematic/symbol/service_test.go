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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic/symbol"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("OpenService", func() {
		It("Should create a service with minimal configuration", func() {
			testDB := gorp.Wrap(memkv.New())
			testOtg := MustSucceed(ontology.Open(ctx, ontology.Config{
				EnableSearch: new(false),
				DB:           testDB,
			}))

			testSvc, err := symbol.OpenService(ctx, symbol.ServiceConfig{
				DB:       testDB,
				Ontology: testOtg,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(testSvc).ToNot(BeNil())

			Expect(testSvc.Close()).To(Succeed())
			Expect(testOtg.Close()).To(Succeed())
			Expect(testDB.Close()).To(Succeed())
		})

		It("Should create a service with group configuration", func() {
			testDB := gorp.Wrap(memkv.New())
			testOtg := MustSucceed(ontology.Open(ctx, ontology.Config{
				EnableSearch: new(false),
				DB:           testDB,
			}))
			testGroup := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
				DB:       testDB,
				Ontology: testOtg,
			}))

			testSvc, err := symbol.OpenService(ctx, symbol.ServiceConfig{
				DB:       testDB,
				Ontology: testOtg,
				Group:    testGroup,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(testSvc).ToNot(BeNil())
			Expect(testSvc.Group()).ToNot(BeNil())
			Expect(testSvc.Group().Name).To(Equal("Schematic Symbols"))

			Expect(testSvc.Close()).To(Succeed())
			Expect(testOtg.Close()).To(Succeed())
			Expect(testDB.Close()).To(Succeed())
		})

		It("Should fail with invalid configuration", func() {
			_, err := symbol.OpenService(ctx, symbol.ServiceConfig{
				DB: nil,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("db: must be non-nil"))

			_, err = symbol.OpenService(ctx, symbol.ServiceConfig{
				Ontology: otg,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("db"))
		})

		It("Should handle configuration override correctly", func() {
			testDB1 := gorp.Wrap(memkv.New())
			testDB2 := gorp.Wrap(memkv.New())
			testOtg1 := MustSucceed(ontology.Open(ctx, ontology.Config{
				EnableSearch: new(false),
				DB:           testDB1,
			}))
			testOtg2 := MustSucceed(ontology.Open(ctx, ontology.Config{
				EnableSearch: new(false),
				DB:           testDB2,
			}))

			cfg1 := symbol.ServiceConfig{
				DB:       testDB1,
				Ontology: testOtg1,
			}
			cfg2 := symbol.ServiceConfig{
				DB:       testDB2,
				Ontology: testOtg2,
			}

			testSvc, err := symbol.OpenService(ctx, cfg1, cfg2)
			Expect(err).ToNot(HaveOccurred())
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
		It("Should create a writer with transaction", func() {
			writer := svc.NewWriter(tx)
			Expect(writer).ToNot(BeNil())
		})

		It("Should create a writer without transaction", func() {
			writer := svc.NewWriter(nil)
			Expect(writer).ToNot(BeNil())
		})
	})

	Describe("NewRetrieve", func() {
		It("Should create a retrieve query builder", func() {
			retrieve := svc.NewRetrieve()
			Expect(retrieve).ToNot(BeNil())
		})
	})

	Describe("Close", func() {
		It("Should close the service cleanly", func() {
			testDB := gorp.Wrap(memkv.New())
			testOtg := MustSucceed(ontology.Open(ctx, ontology.Config{
				EnableSearch: new(false),
				DB:           testDB,
			}))

			testSvc, err := symbol.OpenService(ctx, symbol.ServiceConfig{
				DB:       testDB,
				Ontology: testOtg,
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(testSvc.Close()).To(Succeed())
			// Should be idempotent
			Expect(testSvc.Close()).To(Succeed())

			Expect(testOtg.Close()).To(Succeed())
			Expect(testDB.Close()).To(Succeed())
		})
	})

	Describe("Integration", func() {
		It("Should handle concurrent operations", func() {
			done := make(chan bool, 3)

			// Writer goroutine
			go func() {
				defer GinkgoRecover()
				for range 10 {
					sym := symbol.Symbol{
						Name: "concurrent-write",
						Data: map[string]any{"svg": "<svg>...</svg>"},
					}
					Expect(svc.NewWriter(nil).Create(context.Background(), &sym, ws.OntologyID())).To(Succeed())
					Expect(svc.NewWriter(nil).Delete(context.Background(), sym.Key)).To(Succeed())
				}
				done <- true
			}()

			// Reader goroutine 1
			go func() {
				defer GinkgoRecover()
				for range 10 {
					var symbols []symbol.Symbol
					_ = svc.NewRetrieve().Entries(&symbols).Exec(context.Background(), nil)
				}
				done <- true
			}()

			// Reader goroutine 2
			go func() {
				defer GinkgoRecover()
				for range 10 {
					var symbols []symbol.Symbol
					_ = svc.NewRetrieve().Entries(&symbols).Exec(context.Background(), nil)
				}
				done <- true
			}()

			// Wait for all goroutines
			for i := 0; i < 3; i++ {
				Eventually(done, "5s").Should(Receive())
			}
		})
	})
})
