// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel/verification"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Verification", func() {
	Describe("Config", func() {
		Describe("Validate", func() {
			It("should return an error if the DB is nil", func() {
				Expect(verification.DefaultConfig.Validate()).To(HaveOccurred())
			})
			It("should return an error if the WarningTime is zero", func() {
				cfg := verification.DefaultConfig.
					Override(verification.Config{WarningTime: 0})
				Expect(cfg.Validate()).To(HaveOccurred())
			})
			It("should return an error if the CheckInterval is zero", func() {
				cfg := verification.DefaultConfig.
					Override(verification.Config{CheckInterval: 0})
				Expect(cfg.Validate()).To(HaveOccurred())
			})
			It("should not error if there is a valid DB", func() {
				db := memkv.New()
				cfg := verification.DefaultConfig.Override(verification.Config{DB: db})
				Expect(cfg.Validate()).To(Succeed())
				Expect(db.Close()).To(Succeed())
			})
		})
		Describe("Override", func() {
			It("should override the DB", func() {
				db := memkv.New()
				cfg := verification.DefaultConfig.Override(verification.Config{DB: db})
				Expect(cfg.DB).To(BeEquivalentTo(db))
				Expect(db.Close()).To(Succeed())
			})
			It("should override the WarningTime if it is not zero", func() {
				cfg := verification.DefaultConfig.
					Override(verification.Config{WarningTime: 1 * time.Hour})
				Expect(cfg.WarningTime).To(BeEquivalentTo(1 * time.Hour))
			})
			It("should not override the WarningTime if it is zero", func() {
				cfg := verification.DefaultConfig.
					Override(verification.Config{WarningTime: 0})
				Expect(cfg.WarningTime).
					To(BeEquivalentTo(verification.DefaultConfig.WarningTime))
			})
			It("should override the CheckInterval if it is not zero", func() {
				cfg := verification.DefaultConfig.
					Override(verification.Config{CheckInterval: 1 * time.Hour})
				Expect(cfg.CheckInterval).To(BeEquivalentTo(1 * time.Hour))
			})
			It("should not override the CheckInterval if it is zero", func() {
				cfg := verification.DefaultConfig.
					Override(verification.Config{CheckInterval: 0})
				Expect(cfg.CheckInterval).
					To(BeEquivalentTo(verification.DefaultConfig.CheckInterval))
			})
			It("should override the Verifier if it is not empty", func() {
				cfg := verification.DefaultConfig.
					Override(verification.Config{Verifier: "test"})
				Expect(cfg.Verifier).To(BeEquivalentTo("test"))
			})
		})
	})
	Describe("Service", func() {
		var db kv.DB
		var ctx context.Context
		BeforeEach(func() {
			db = memkv.New()
			ctx = context.Background()
		})
		AfterEach(func() {
			Expect(db.Close()).To(Succeed())
		})
		It("should open with no verifier", func() {
			svc := MustSucceed(verification.OpenService(ctx, verification.Config{DB: db}))
			Expect(svc).ToNot(BeNil())
			Expect(svc.IsOverflowed(0)).To(Succeed())
			Expect(svc.IsOverflowed(verification.FreeCount)).To(Succeed())
			Expect(svc.IsOverflowed(verification.FreeCount + 1)).
				To(MatchError(verification.ErrFree))
			Expect(svc.Close()).To(Succeed())
		})
		It("should ")
	})
})
