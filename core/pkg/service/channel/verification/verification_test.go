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
	"crypto/ed25519"
	"crypto/rand"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel/verification"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

const keyID = "test"

var (
	now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	day = 24 * time.Hour
)

func seconds(t time.Time) *uint32 { return new(uint32(t.Unix())) }

func grant() verification.Grant {
	return verification.Grant{
		Jti: uuid.New(),
		Iat: uint32(now.Add(-day).Unix()),
		Exp: seconds(now.Add(30 * day)),
		V:   1,
		Org: uuid.New(),
		Ed:  "e",
		Fs:  1,
		N:   1,
	}
}

var _ = Describe("Verification", func() {
	var (
		db      kv.DB
		private ed25519.PrivateKey
		anchors verification.Anchors
		cfg     verification.ServiceConfig
	)
	sign := func(g verification.Grant) string {
		return MustSucceed(verification.Sign(private, keyID, g))
	}
	open := func(ctx SpecContext, cfgs ...verification.ServiceConfig) *verification.Service {
		return MustSucceed(verification.OpenService(ctx, append(
			[]verification.ServiceConfig{cfg}, cfgs...,
		)...))
	}
	BeforeEach(func() {
		db = memkv.New()
		public, priv := MustSucceed2(ed25519.GenerateKey(rand.Reader))
		private = priv
		anchors = verification.Anchors{keyID: public}
		cfg = verification.ServiceConfig{
			DB:      db,
			Anchors: anchors,
			Version: "0.60.1",
			Now:     func() time.Time { return now },
		}
	})
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })

	Describe("ServiceConfig", func() {
		It("should reject a missing DB", func() {
			Expect(verification.DefaultServiceConfig.Validate()).To(HaveOccurred())
		})
		It("should keep defaults the override leaves zero", func() {
			c := verification.DefaultServiceConfig.Override(cfg)
			Expect(c.Grace).To(Equal(verification.DefaultServiceConfig.Grace))
			Expect(c.Rollback).To(Equal(verification.DefaultServiceConfig.Rollback))
			Expect(c.Validate()).To(Succeed())
		})
	})

	Describe("Verify", func() {
		It("should return the grant a valid token carries", func() {
			g := grant()
			Expect(verification.Verify(anchors, sign(g))).To(Equal(g))
		})
		It("should reject an unknown key", func() {
			Expect(verification.Verify(
				verification.Anchors{"other": anchors[keyID]}, sign(grant()),
			)).Error().To(MatchError(verification.ErrInvalid))
		})
		It("should reject a token signed under another algorithm", func() {
			tk := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"v": 1})
			tk.Header["kid"] = keyID
			s := MustSucceed(tk.SignedString([]byte(anchors[keyID])))
			Expect(verification.Verify(anchors, s)).Error().
				To(MatchError(verification.ErrInvalid))
		})
		It("should reject a token signed by another key", func() {
			_, other := MustSucceed2(ed25519.GenerateKey(rand.Reader))
			s := MustSucceed(verification.Sign(other, keyID, grant()))
			Expect(verification.Verify(anchors, s)).Error().
				To(MatchError(verification.ErrInvalid))
		})
		It("should reject garbage", func() {
			Expect(verification.Verify(anchors, "not a token")).Error().
				To(MatchError(verification.ErrInvalid))
		})
		It("should reject an unsupported claim set version", func() {
			g := grant()
			g.V = 2
			Expect(verification.Verify(anchors, sign(g))).Error().
				To(MatchError(verification.ErrInvalid))
		})
	})

	Describe("Open", func() {
		It("should open missing when nothing is stored", func(ctx SpecContext) {
			svc := open(ctx)
			defer func() { Expect(svc.Close()).To(Succeed()) }()
			info := svc.Retrieve()
			Expect(info.State).To(Equal(verification.StateMissing))
			Expect(info.Grant).To(BeNil())
			Expect(svc.Check()).To(MatchError(verification.ErrMissing))
			Expect(svc.IsOverflowed(1000)).To(Succeed())
		})
		It(
			"should accept a verifier on open and load it on the next",
			func(ctx SpecContext) {
				g := grant()
				svc := open(ctx, verification.ServiceConfig{Verifier: sign(g)})
				Expect(svc.Retrieve().State).To(Equal(verification.StateOK))
				Expect(svc.Close()).To(Succeed())
				svc = open(ctx)
				defer func() { Expect(svc.Close()).To(Succeed()) }()
				info := svc.Retrieve()
				Expect(info.State).To(Equal(verification.StateOK))
				Expect(*info.Grant).To(Equal(g))
				Expect(svc.Check()).To(Succeed())
			},
		)
		It(
			"should fail to open on a verifier that does not verify",
			func(ctx SpecContext) {
				Expect(verification.OpenService(ctx, cfg, verification.ServiceConfig{
					Verifier: "garbage",
				})).Error().To(MatchError(verification.ErrInvalid))
			},
		)
		It("should remove the previous format's entry", func(ctx SpecContext) {
			legacy := []byte("bGljZW5zZUtleQ==")
			Expect(db.Set(ctx, legacy, []byte("old"))).To(Succeed())
			svc := open(ctx)
			Expect(svc.Close()).To(Succeed())
			Expect(db.Get(ctx, legacy)).Error().To(HaveOccurred())
		})
		It("should prefer a covering entry over an expired one", func(ctx SpecContext) {
			expired := grant()
			expired.Exp = seconds(now.Add(-100 * day))
			svc := open(ctx)
			Expect(svc.Activate(ctx, sign(grant()))).Error().To(Succeed())
			Expect(svc.Close()).To(Succeed())
			// Store the expired entry directly: Activate refuses it.
			key := append([]byte("bGljZW5zZUtleQ==/"), expired.Jti.String()...)
			Expect(db.Set(ctx, key, []byte(sign(expired)))).To(Succeed())
			svc = open(ctx)
			defer func() { Expect(svc.Close()).To(Succeed()) }()
			Expect(svc.Retrieve().State).To(Equal(verification.StateOK))
		})
		It("should report a stored entry that no longer covers", func(ctx SpecContext) {
			g := grant()
			svc := open(ctx, verification.ServiceConfig{Verifier: sign(g)})
			Expect(svc.Close()).To(Succeed())
			later := now.Add(60 * day)
			svc = open(ctx, verification.ServiceConfig{
				Now: func() time.Time { return later },
			})
			defer func() { Expect(svc.Close()).To(Succeed()) }()
			Expect(svc.Retrieve().State).To(Equal(verification.StateExpired))
			Expect(svc.Check()).To(MatchError(verification.ErrExpired))
		})
		It(
			"should treat every entry as expired after a clock rollback",
			func(ctx SpecContext) {
				svc := open(ctx, verification.ServiceConfig{Verifier: sign(grant())})
				Expect(svc.Close()).To(Succeed())
				earlier := now.Add(-2 * day)
				svc = open(ctx, verification.ServiceConfig{
					Now: func() time.Time { return earlier },
				})
				defer func() { Expect(svc.Close()).To(Succeed()) }()
				info := svc.Retrieve()
				Expect(info.State).To(Equal(verification.StateExpired))
				Expect(info.Warning).To(ContainSubstring("clock"))
			},
		)
		It("should tolerate a clock inside the rollback window", func(ctx SpecContext) {
			svc := open(ctx, verification.ServiceConfig{Verifier: sign(grant())})
			Expect(svc.Close()).To(Succeed())
			earlier := now.Add(-time.Hour)
			svc = open(ctx, verification.ServiceConfig{
				Now: func() time.Time { return earlier },
			})
			defer func() { Expect(svc.Close()).To(Succeed()) }()
			Expect(svc.Retrieve().State).To(Equal(verification.StateOK))
		})
	})

	Describe("Activate", func() {
		var svc *verification.Service
		BeforeEach(func(ctx SpecContext) { svc = open(ctx) })
		AfterEach(func() { Expect(svc.Close()).To(Succeed()) })

		It("should accept a subscription before expiry", func(ctx SpecContext) {
			info := MustSucceed(svc.Activate(ctx, sign(grant())))
			Expect(info.State).To(Equal(verification.StateOK))
			Expect(info.Warning).To(BeEmpty())
		})
		It("should warn when expiry is near", func(ctx SpecContext) {
			g := grant()
			g.Exp = seconds(now.Add(2 * day))
			info := MustSucceed(svc.Activate(ctx, sign(g)))
			Expect(info.State).To(Equal(verification.StateOK))
			Expect(info.Warning).To(ContainSubstring("expires in"))
		})
		It(
			"should accept a subscription inside the grace window",
			func(ctx SpecContext) {
				g := grant()
				g.Exp = seconds(now.Add(-2 * day))
				info := MustSucceed(svc.Activate(ctx, sign(g)))
				Expect(info.State).To(Equal(verification.StateOK))
				Expect(info.Warning).To(ContainSubstring("grace"))
			},
		)
		It("should refuse a subscription past the grace window", func(ctx SpecContext) {
			g := grant()
			g.Exp = seconds(now.Add(-20 * day))
			Expect(svc.Activate(ctx, sign(g))).Error().
				To(MatchError(verification.ErrExpired))
			Expect(svc.Retrieve().State).To(Equal(verification.StateMissing))
		})
		It("should accept a perpetual grant under its ceiling", func(ctx SpecContext) {
			g := grant()
			g.Exp = nil
			g.Mv = new("0.62")
			info := MustSucceed(svc.Activate(ctx, sign(g)))
			Expect(info.State).To(Equal(verification.StateOK))
			Expect(info.Warning).To(BeEmpty())
		})
		It("should accept a perpetual grant at its ceiling", func(ctx SpecContext) {
			g := grant()
			g.Exp = nil
			g.Mv = new("0.60")
			Expect(svc.Activate(ctx, sign(g))).Error().To(Succeed())
		})
		It("should refuse a perpetual grant over its ceiling", func(ctx SpecContext) {
			g := grant()
			g.Exp = nil
			g.Mv = new("0.59")
			Expect(svc.Activate(ctx, sign(g))).Error().
				To(MatchError(verification.ErrExpired))
		})
		It(
			"should refuse a grant with neither expiry nor ceiling",
			func(ctx SpecContext) {
				g := grant()
				g.Exp = nil
				Expect(svc.Activate(ctx, sign(g))).Error().
					To(MatchError(verification.ErrInvalid))
			},
		)
		It("should refuse a ceiling that does not parse", func(ctx SpecContext) {
			g := grant()
			g.Mv = new("latest")
			Expect(svc.Activate(ctx, sign(g))).Error().
				To(MatchError(verification.ErrInvalid))
		})
		It("should fall back to the ceiling past expiry", func(ctx SpecContext) {
			g := grant()
			g.Exp = seconds(now.Add(-100 * day))
			g.Mv = new("0.62")
			info := MustSucceed(svc.Activate(ctx, sign(g)))
			Expect(info.State).To(Equal(verification.StateOK))
			Expect(info.Warning).To(ContainSubstring("subscription ended"))
		})
		It(
			"should refuse a fallback whose ceiling is below this version",
			func(ctx SpecContext) {
				g := grant()
				g.Exp = seconds(now.Add(-100 * day))
				g.Mv = new("0.59")
				Expect(svc.Activate(ctx, sign(g))).Error().
					To(MatchError(verification.ErrExpired))
			},
		)
		It("should refuse a grant bound to other hosts", func(ctx SpecContext) {
			g := grant()
			g.Fp = []string{"0000"}
			Expect(svc.Activate(ctx, sign(g))).Error().
				To(MatchError(verification.ErrHost))
		})
		It("should accept a grant bound to this host", func(ctx SpecContext) {
			host := svc.Retrieve().Host
			if len(host) == 0 {
				Skip("this machine has no hashable network interface")
			}
			g := grant()
			g.Fp = []string{"0000", host[len(host)-1]}
			Expect(svc.Activate(ctx, sign(g))).Error().To(Succeed())
		})
		It("should refuse hashes from a scheme this Core does not implement", func(
			ctx SpecContext,
		) {
			host := svc.Retrieve().Host
			if len(host) == 0 {
				Skip("this machine has no hashable network interface")
			}
			g := grant()
			g.Fs = 2
			g.Fp = []string{host[0]}
			Expect(svc.Activate(ctx, sign(g))).Error().
				To(MatchError(verification.ErrHost))
		})
		It("should reject an invalid token", func(ctx SpecContext) {
			Expect(svc.Activate(ctx, "nope")).Error().
				To(MatchError(verification.ErrInvalid))
		})
		It("should enforce the channel cap", func(ctx SpecContext) {
			g := grant()
			g.Ch = 10
			Expect(svc.Activate(ctx, sign(g))).Error().To(Succeed())
			Expect(svc.IsOverflowed(10)).To(Succeed())
			Expect(svc.IsOverflowed(11)).To(MatchError(verification.ErrTooMany))
		})
		It("should not cap a grant with a zero cap", func(ctx SpecContext) {
			Expect(svc.Activate(ctx, sign(grant()))).Error().To(Succeed())
			Expect(svc.IsOverflowed(1 << 19)).To(Succeed())
		})
	})
})
