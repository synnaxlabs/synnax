// Copyright 2026 Synnax Labs, Inc.
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
	"io"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel/verification"
	"github.com/synnaxlabs/x/encoding/base64"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var errDBGetCalled = errors.New("DB.Get called too many times")

type db struct {
	kv.DB
	timesGetCalled int
}

var _ kv.DB = &db{}

func newDB() *db { return &db{DB: memkv.New()} }

func (d *db) Get(ctx context.Context, key []byte, opts ...any) ([]byte, io.Closer, error) {
	d.timesGetCalled++
	if d.timesGetCalled == 2 {
		return nil, nil, errDBGetCalled
	}
	return d.DB.Get(ctx, key, opts...)
}

var _ = Describe("Verification", func() {
	Describe("ConfigValues", func() {
		Describe("Validate", func() {
			It("should return an error if the DB is nil", func() {
				Expect(verification.DefaultServiceConfig.Validate()).To(HaveOccurred())
			})
			It("should return an error if the WarningTime is zero", func() {
				cfg := verification.DefaultServiceConfig.
					Override(verification.ServiceConfig{WarningTime: 0})
				Expect(cfg.Validate()).To(HaveOccurred())
			})
			It("should return an error if the CheckInterval is zero", func() {
				cfg := verification.DefaultServiceConfig.
					Override(verification.ServiceConfig{CheckInterval: 0})
				Expect(cfg.Validate()).To(HaveOccurred())
			})
			It("should not error if there is a valid DB", func() {
				db := memkv.New()
				cfg := verification.DefaultServiceConfig.Override(verification.ServiceConfig{DB: db})
				Expect(cfg.Validate()).To(Succeed())
				Expect(db.Close()).To(Succeed())
			})
		})
		Describe("Override", func() {
			It("should override the DB", func() {
				db := memkv.New()
				cfg := verification.DefaultServiceConfig.Override(verification.ServiceConfig{DB: db})
				Expect(cfg.DB).To(BeEquivalentTo(db))
				Expect(db.Close()).To(Succeed())
			})
			It("should override the WarningTime if it is not zero", func() {
				cfg := verification.DefaultServiceConfig.
					Override(verification.ServiceConfig{WarningTime: 1 * time.Hour})
				Expect(cfg.WarningTime).To(BeEquivalentTo(1 * time.Hour))
			})
			It("should not override the WarningTime if it is zero", func() {
				cfg := verification.DefaultServiceConfig.
					Override(verification.ServiceConfig{WarningTime: 0})
				Expect(cfg.WarningTime).
					To(BeEquivalentTo(verification.DefaultServiceConfig.WarningTime))
			})
			It("should override the CheckInterval if it is not zero", func() {
				cfg := verification.DefaultServiceConfig.
					Override(verification.ServiceConfig{CheckInterval: 1 * time.Hour})
				Expect(cfg.CheckInterval).To(BeEquivalentTo(1 * time.Hour))
			})
			It("should not override the CheckInterval if it is zero", func() {
				cfg := verification.DefaultServiceConfig.
					Override(verification.ServiceConfig{CheckInterval: 0})
				Expect(cfg.CheckInterval).
					To(BeEquivalentTo(verification.DefaultServiceConfig.CheckInterval))
			})
			It("should override the Verifier if it is not empty", func() {
				cfg := verification.DefaultServiceConfig.
					Override(verification.ServiceConfig{Verifier: "test"})
				Expect(cfg.Verifier).To(BeEquivalentTo("test"))
			})
		})
	})
	Describe("Service", func() {
		var ctx context.Context
		BeforeEach(func() {
			ctx = context.Background()
		})
		Describe("Normal DB usage", func() {
			var db kv.DB
			BeforeEach(func() {
				db = memkv.New()
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
			})
			It("should fail to open with invalid config", func() {
				Expect(verification.OpenService(ctx)).Error().To(HaveOccurred())
			})
			It("should open with no verifier", func() {
				svc := MustSucceed(
					verification.OpenService(ctx, verification.ServiceConfig{DB: db}),
				)
				Expect(svc).ToNot(BeNil())
				Expect(svc.IsOverflowed(0)).To(Succeed())
				Expect(svc.IsOverflowed(verification.FreeCount)).To(Succeed())
				Expect(svc.IsOverflowed(verification.FreeCount + 1)).
					To(MatchError(verification.ErrFree))
				Expect(svc.Close()).To(Succeed())
			})
			DescribeTable("Invalid verifier", func(v string) {
				Expect(verification.OpenService(
					ctx,
					verification.ServiceConfig{DB: db, Verifier: v},
				)).
					Error().To(MatchError(verification.ErrInvalid))
			},
				Entry("invalid format", "not a valid format"),
				Entry(
					"invalid date",
					base64.MustDecode("MDAwMDAwLTY0MzE3Mjg0LTA0MDA1MDAwMDU="),
				),
				Entry(
					"invalid checksum",
					base64.MustDecode("ODk0NDc4LTY0MzE3Mjg0LTAwMDAwMDAwMDA="),
				),
			)
			Describe("Valid verifier", func() {
				It("should open with a valid verifier", func() {
					svc := MustSucceed(verification.OpenService(
						ctx,
						verification.ServiceConfig{
							DB: db,
							Verifier: base64.MustDecode(
								"ODg1NTA4LTY0MzE3Mzg0LTA0MDA1MDAwMDU=",
							),
						},
					))
					Expect(svc).ToNot(BeNil())
					Expect(svc.IsOverflowed(0)).To(Succeed())
					Expect(svc.IsOverflowed(100)).To(Succeed())
					Expect(svc.IsOverflowed(101)).To(MatchError(verification.ErrTooMany))
					Expect(svc.Close()).To(Succeed())
				})
				It("should load a verifier from the DB", func() {
					svc := MustSucceed(verification.OpenService(
						ctx,
						verification.ServiceConfig{
							DB: db,
							Verifier: base64.MustDecode(
								"ODg1NTA4LTY0MzE3Mzg0LTA0MDA1MDAwMDU=",
							),
						},
					))
					Expect(svc.Close()).To(Succeed())
					svc = MustSucceed(verification.OpenService(
						ctx,
						verification.ServiceConfig{DB: db},
					))
					Expect(svc.IsOverflowed(0)).To(Succeed())
					Expect(svc.IsOverflowed(100)).To(Succeed())
					Expect(svc.IsOverflowed(101)).
						To(MatchError(verification.ErrTooMany))
					Expect(svc.Close()).To(Succeed())
				})
			})
			Describe("Stale verifier", func() {
				It("should allow loading a stale verifier", func() {
					svc := MustSucceed(verification.OpenService(
						ctx,
						verification.ServiceConfig{
							DB: db,
							Verifier: base64.MustDecode(
								"ODk0NDc4LTY0MzE3Mzg0LTA0MDA1MDAwMDU=",
							),
						},
					))
					Expect(svc).ToNot(BeNil())
					Expect(svc.IsOverflowed(0)).To(Succeed())
					Expect(svc.IsOverflowed(50)).To(Succeed())
					Expect(svc.IsOverflowed(51)).To(MatchError(verification.ErrStale))
					Expect(svc.Close()).To(Succeed())
				})
			})
		})
		Describe("DB errors", func() {
			It("should propagate DB errors to the service", func() {
				db := newDB()
				svc := MustSucceed(verification.OpenService(
					ctx,
					verification.ServiceConfig{DB: db},
				))
				Expect(svc.Close()).To(Succeed())
				Expect(verification.OpenService(
					ctx,
					verification.ServiceConfig{DB: db},
				)).Error().To(MatchError(errDBGetCalled))
				Expect(db.Close()).To(Succeed())
			})
		})
	})
})
