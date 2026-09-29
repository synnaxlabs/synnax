// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package license_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

const keyID = "test"

var (
	now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	day = 24 * time.Hour
)

// pausedDB holds the first armed scan after it takes its snapshot, until resume closes.
type pausedDB struct {
	kv.DB
	armed  atomic.Bool
	paused chan struct{}
	resume chan struct{}
}

func (p *pausedDB) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	iter, err := p.DB.OpenIterator(opts)
	if p.armed.CompareAndSwap(true, false) {
		close(p.paused)
		<-p.resume
	}
	return iter, err
}

func seconds(t time.Time) *uint32 { return new(uint32(t.Unix())) }

func newLicense() license.License {
	return license.License{
		Jti:               uuid.New(),
		Iat:               uint32(now.Add(-day).Unix()),
		Exp:               seconds(now.Add(30 * day)),
		ClaimsVersion:     1,
		Organization:      uuid.New(),
		Edition:           "e",
		FingerprintScheme: 1,
		Machines:          1,
	}
}

var _ = Describe("License", func() {
	var (
		db      kv.DB
		private ed25519.PrivateKey
		anchors license.Anchors
		cfg     license.ServiceConfig
	)
	sign := func(lic license.License) string {
		GinkgoHelper()
		return MustSucceed(license.Sign(private, keyID, lic))
	}
	// store writes lic the way Activate does, bypassing its checks.
	store := func(ctx SpecContext, lic license.License) {
		GinkgoHelper()
		key := append([]byte("license/"), lic.Jti.String()...)
		Expect(db.Set(ctx, key, []byte(sign(lic)))).To(Succeed())
	}
	open := func(ctx SpecContext, cfgs ...license.ServiceConfig) *license.Service {
		GinkgoHelper()
		return MustOpen(license.OpenService(ctx, append(
			[]license.ServiceConfig{cfg}, cfgs...,
		)...))
	}
	BeforeEach(func() {
		db = DeferClose(memkv.New())
		public, priv := MustSucceed2(ed25519.GenerateKey(rand.Reader))
		private = priv
		anchors = license.Anchors{keyID: public}
		cfg = license.ServiceConfig{
			DB:      db,
			Anchors: anchors,
			Version: "0.60.1",
			Now:     func() time.Time { return now },
		}
	})

	Describe("ServiceConfig", func() {
		DescribeTable(
			"should reject a missing field",
			func(field string, clear func(c *license.ServiceConfig)) {
				c := license.DefaultServiceConfig.Override(cfg)
				clear(&c)
				Expect(c.Validate()).To(MatchError(ContainSubstring(field)))
			},
			Entry("db", "db", func(c *license.ServiceConfig) { c.DB = nil }),
			Entry("anchors", "anchors", func(c *license.ServiceConfig) {
				c.Anchors = nil
			}),
			Entry("now", "now", func(c *license.ServiceConfig) { c.Now = nil }),
			Entry("check interval", "check_interval",
				func(c *license.ServiceConfig) { c.CheckInterval = 0 }),
			Entry("warning time", "warning_time",
				func(c *license.ServiceConfig) { c.WarningTime = 0 }),
			Entry(
				"grace",
				"grace",
				func(c *license.ServiceConfig) { c.Grace = 0 },
			),
			Entry("rollback", "rollback",
				func(c *license.ServiceConfig) { c.Rollback = 0 }),
		)
		It("should keep defaults the override leaves zero", func() {
			c := license.DefaultServiceConfig.Override(cfg)
			Expect(c.Grace).To(Equal(license.DefaultServiceConfig.Grace))
			Expect(c.Rollback).To(Equal(license.DefaultServiceConfig.Rollback))
			Expect(c.Validate()).To(Succeed())
		})
	})

	Describe("Verify", func() {
		It("should return the license a valid token carries", func() {
			lic := newLicense()
			Expect(license.Verify(anchors, sign(lic))).To(Equal(lic))
		})
		It("should reject an unknown key", func() {
			Expect(license.Verify(
				license.Anchors{"other": anchors[keyID]}, sign(newLicense()),
			)).Error().To(MatchError(license.ErrInvalid))
		})
		It("should reject a token signed under another algorithm", func() {
			tk := jwt.NewWithClaims(
				jwt.SigningMethodHS256,
				jwt.MapClaims{"claims_version": 1},
			)
			tk.Header["kid"] = keyID
			s := MustSucceed(tk.SignedString([]byte(anchors[keyID])))
			Expect(license.Verify(anchors, s)).Error().
				To(MatchError(license.ErrInvalid))
		})
		It("should reject a token signed by another key", func() {
			_, other := MustSucceed2(ed25519.GenerateKey(rand.Reader))
			s := MustSucceed(license.Sign(other, keyID, newLicense()))
			Expect(license.Verify(anchors, s)).Error().
				To(MatchError(license.ErrInvalid))
		})
		It("should reject garbage", func() {
			Expect(license.Verify(anchors, "not a token")).Error().
				To(MatchError(license.ErrInvalid))
		})
		It("should reject an unsupported claim set version", func() {
			lic := newLicense()
			lic.ClaimsVersion = 2
			Expect(license.Verify(anchors, sign(lic))).Error().
				To(MatchError(license.ErrInvalid))
		})
		It("should accept a token requiring every claim the license carries", func() {
			lic := newLicense()
			for f := range reflect.TypeFor[license.License]().Fields() {
				tag := f.Tag.Get("json")
				lic.Required = append(lic.Required, strings.Split(tag, ",")[0])
			}
			Expect(license.Verify(anchors, sign(lic))).To(Equal(lic))
		})
		It("should refuse a token requiring a claim this Core does not know", func() {
			lic := newLicense()
			lic.Required = []string{"channels", "seats"}
			Expect(license.Verify(anchors, sign(lic))).Error().To(And(
				MatchError(license.ErrInvalid),
				MatchError(ContainSubstring("understands seats")),
			))
		})
	})

	Describe("Fingerprint", func() {
		fingerprint := license.Fingerprint{"aaaa", "bbbb"}
		DescribeTable(
			"Covers",
			func(scheme uint8, hashes []string, expected bool) {
				Expect(fingerprint.Covers(scheme, hashes)).To(Equal(expected))
			},
			Entry("unbound", uint8(1), nil, true),
			Entry(
				"one hash of this fingerprint",
				uint8(1),
				[]string{"0000", "bbbb"},
				true,
			),
			Entry("other machines only", uint8(1), []string{"0000"}, false),
			Entry("another scheme", uint8(2), []string{"aaaa"}, false),
			Entry(
				"unbound license with an unknown scheme is covered",
				uint8(2),
				nil,
				true,
			),
		)
	})

	Describe("Open", func() {
		It("should open missing when nothing is stored", func(ctx SpecContext) {
			svc := open(ctx)
			info := svc.Retrieve()
			Expect(info.State).To(Equal(license.StateMissing))
			Expect(info.License).To(BeNil())
			Expect(svc.Check()).To(MatchError(license.ErrMissing))
			Expect(svc.CheckChannelLimit(1000)).To(Succeed())
		})
		It(
			"should accept a token on open and load it on the next",
			func(ctx SpecContext) {
				lic := newLicense()
				svc := open(ctx, license.ServiceConfig{Token: sign(lic)})
				Expect(svc.Retrieve().State).To(Equal(license.StateOk))
				Expect(svc.Close()).To(Succeed())
				svc = open(ctx)
				info := svc.Retrieve()
				Expect(info.State).To(Equal(license.StateOk))
				Expect(*info.License).To(Equal(lic))
				Expect(svc.Check()).To(Succeed())
			},
		)
		It(
			"should fail to open on a licenseToken that does not verify",
			func(ctx SpecContext) {
				Expect(license.OpenService(ctx, cfg, license.ServiceConfig{
					Token: "garbage",
				})).Error().To(MatchError(license.ErrInvalid))
			},
		)
		It("should remove the previous format's entry", func(ctx SpecContext) {
			legacy := []byte("bGljZW5zZUtleQ==")
			Expect(db.Set(ctx, legacy, []byte("old"))).To(Succeed())
			svc := open(ctx)
			Expect(svc.Close()).To(Succeed())
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		})
		It("should prefer a covering entry over an expired one", func(ctx SpecContext) {
			expired := newLicense()
			expired.Exp = seconds(now.Add(-100 * day))
			expired.Iat = uint32(now.Unix())
			store(ctx, newLicense())
			store(ctx, expired)
			Expect(open(ctx).Retrieve().State).To(Equal(license.StateOk))
		})
		It("should prefer the most recently issued entry", func(ctx SpecContext) {
			older, newer := newLicense(), newLicense()
			older.Jti, newer.Jti = uuid.UUID{}, uuid.Max()
			older.Channels, newer.Channels = 100, 500
			newer.Iat = uint32(now.Unix())
			store(ctx, older)
			store(ctx, newer)
			Expect(open(ctx).Retrieve().License.Channels).To(BeEquivalentTo(500))
		})
		It(
			"should skip a stored entry issued for another machine",
			func(ctx SpecContext) {
				lic := newLicense()
				lic.Fingerprints = []string{"0000"}
				store(ctx, lic)
				svc := open(ctx)
				Expect(svc.Retrieve().State).To(Equal(license.StateMissing))
				Expect(svc.Check()).To(MatchError(license.ErrMissing))
			},
		)
		It(
			"should skip a stored entry that no longer verifies",
			func(ctx SpecContext) {
				store(ctx, newLicense())
				other, _ := MustSucceed2(ed25519.GenerateKey(rand.Reader))
				svc := open(ctx, license.ServiceConfig{
					Anchors: license.Anchors{keyID: other},
				})
				Expect(svc.Retrieve().State).To(Equal(license.StateMissing))
				Expect(svc.Check()).To(MatchError(license.ErrMissing))
			},
		)
		It("should report a stored entry that no longer covers", func(ctx SpecContext) {
			lic := newLicense()
			svc := open(ctx, license.ServiceConfig{Token: sign(lic)})
			Expect(svc.Close()).To(Succeed())
			later := now.Add(60 * day)
			svc = open(ctx, license.ServiceConfig{
				Now: func() time.Time { return later },
			})
			Expect(svc.Retrieve().State).To(Equal(license.StateExpired))
			Expect(svc.Check()).To(MatchError(license.ErrExpired))
		})
		It(
			"should not cap channels while the license is expired",
			func(ctx SpecContext) {
				lic := newLicense()
				lic.Channels = 1
				store(ctx, lic)
				later := now.Add(60 * day)
				svc := open(ctx, license.ServiceConfig{
					Now: func() time.Time { return later },
				})
				Expect(svc.Retrieve().State).To(Equal(license.StateExpired))
				Expect(svc.CheckChannelLimit(1000)).To(Succeed())
			},
		)
		It(
			"should treat every entry as expired after a clock rollback",
			func(ctx SpecContext) {
				svc := open(ctx, license.ServiceConfig{Token: sign(newLicense())})
				Expect(svc.Close()).To(Succeed())
				earlier := now.Add(-2 * day)
				svc = open(ctx, license.ServiceConfig{
					Now: func() time.Time { return earlier },
				})
				info := svc.Retrieve()
				Expect(info.State).To(Equal(license.StateExpired))
				Expect(info.Warning).To(ContainSubstring("clock"))
			},
		)
		It("should tolerate a clock inside the rollback window", func(ctx SpecContext) {
			svc := open(ctx, license.ServiceConfig{Token: sign(newLicense())})
			Expect(svc.Close()).To(Succeed())
			earlier := now.Add(-time.Hour)
			svc = open(ctx, license.ServiceConfig{
				Now: func() time.Time { return earlier },
			})
			Expect(svc.Retrieve().State).To(Equal(license.StateOk))
		})
		It("should refuse to open over a corrupt clock mark", func(ctx SpecContext) {
			Expect(db.Set(ctx, []byte("highWater"), []byte{1, 2, 3})).To(Succeed())
			Expect(license.OpenService(ctx, cfg)).Error().To(MatchError(
				ContainSubstring("license clock mark has 3 bytes, want 8"),
			))
		})
	})

	Describe("Clock", func() {
		var (
			clock   atomic.Int64
			clocked license.ServiceConfig
		)
		BeforeEach(func() {
			clock.Store(now.UnixNano())
			clocked = license.ServiceConfig{
				Now:           func() time.Time { return time.Unix(0, clock.Load()) },
				CheckInterval: 10 * time.Millisecond,
			}
		})
		state := func(svc *license.Service) func() license.State {
			return func() license.State { return svc.Retrieve().State }
		}
		mark := func(ctx SpecContext) func() int64 {
			return func() int64 {
				GinkgoHelper()
				raw, closer := MustSucceed2(db.Get(ctx, []byte("highWater")))
				Expect(closer.Close()).To(Succeed())
				return int64(binary.LittleEndian.Uint64(raw))
			}
		}
		It(
			"should recover once a clock that was behind catches up",
			func(ctx SpecContext) {
				svc := open(ctx, license.ServiceConfig{Token: sign(newLicense())})
				Expect(svc.Close()).To(Succeed())
				clock.Store(now.Add(-2 * day).UnixNano())
				svc = open(ctx, clocked)
				Expect(svc.Retrieve().State).To(Equal(license.StateExpired))
				clock.Store(now.Add(time.Hour).UnixNano())
				Eventually(state(svc)).Should(Equal(license.StateOk))
				Expect(svc.Retrieve().Warning).To(BeEmpty())
			},
		)
		It("should never move the recorded time back", func(ctx SpecContext) {
			svc := open(ctx, clocked, license.ServiceConfig{Token: sign(newLicense())})
			Expect(mark(ctx)()).To(Equal(now.UnixNano()))
			clock.Store(now.Add(-2 * day).UnixNano())
			Consistently(mark(ctx)).
				WithTimeout(100 * time.Millisecond).
				Should(Equal(now.UnixNano()))
			Expect(svc.Close()).To(Succeed())
			svc = open(ctx, clocked)
			Expect(svc.Retrieve().State).To(Equal(license.StateExpired))
		})
		It(
			"should ignore the clock for a license without an expiry",
			func(ctx SpecContext) {
				lic := newLicense()
				lic.Exp = nil
				lic.MaxVersion = new("0.60")
				svc := open(ctx, license.ServiceConfig{Token: sign(lic)})
				Expect(svc.Close()).To(Succeed())
				clock.Store(now.Add(-2 * day).UnixNano())
				info := open(ctx, clocked).Retrieve()
				Expect(info.State).To(Equal(license.StateOk))
				Expect(info.Warning).To(BeEmpty())
			},
		)
		It(
			"should fall back to the version ceiling while the clock is behind",
			func(ctx SpecContext) {
				lic := newLicense()
				lic.MaxVersion = new("0.60")
				svc := open(ctx, license.ServiceConfig{Token: sign(lic)})
				Expect(svc.Close()).To(Succeed())
				clock.Store(now.Add(-2 * day).UnixNano())
				info := open(ctx, clocked).Retrieve()
				Expect(info.State).To(Equal(license.StateOk))
				Expect(info.Warning).To(ContainSubstring("clock"))
			},
		)
	})

	Describe("Activate", func() {
		var svc *license.Service
		BeforeEach(func(ctx SpecContext) { svc = open(ctx) })

		It("should accept a subscription before expiry", func(ctx SpecContext) {
			info := MustSucceed(svc.Activate(ctx, sign(newLicense())))
			Expect(info.State).To(Equal(license.StateOk))
			Expect(info.Warning).To(BeEmpty())
		})
		It(
			"should keep a newer license over an older activation",
			func(ctx SpecContext) {
				older, newer := newLicense(), newLicense()
				older.Channels, newer.Channels = 100, 500
				newer.Iat = uint32(now.Unix())
				MustSucceed(svc.Activate(ctx, sign(newer)))
				info := MustSucceed(svc.Activate(ctx, sign(older)))
				Expect(info.License.Channels).To(BeEquivalentTo(500))
				Expect(svc.Retrieve().License.Channels).To(BeEquivalentTo(500))
			},
		)
		It(
			"should hold a load until the one in flight publishes",
			func(ctx SpecContext) {
				paused := &pausedDB{
					DB:     db,
					paused: make(chan struct{}),
					resume: make(chan struct{}),
				}
				svc := open(ctx, license.ServiceConfig{DB: paused})
				release := sync.OnceFunc(func() { close(paused.resume) })
				DeferCleanup(release)
				older, newer := newLicense(), newLicense()
				older.Channels, newer.Channels = 100, 500
				newer.Iat = uint32(now.Unix())
				paused.armed.Store(true)
				olderDone, newerDone := make(chan struct{}), make(chan struct{})
				go func() {
					defer GinkgoRecover()
					MustSucceed(svc.Activate(ctx, sign(older)))
					close(olderDone)
				}()
				Eventually(paused.paused).Should(BeClosed())
				go func() {
					defer GinkgoRecover()
					MustSucceed(svc.Activate(ctx, sign(newer)))
					close(newerDone)
				}()
				Consistently(newerDone, 100*time.Millisecond).ShouldNot(BeClosed())
				release()
				Eventually(olderDone).Should(BeClosed())
				Eventually(newerDone).Should(BeClosed())
				Expect(svc.Retrieve().License.Channels).To(BeEquivalentTo(500))
			},
		)
		It("should warn when expiry is near", func(ctx SpecContext) {
			lic := newLicense()
			lic.Exp = seconds(now.Add(2 * day))
			info := MustSucceed(svc.Activate(ctx, sign(lic)))
			Expect(info.State).To(Equal(license.StateOk))
			Expect(info.Warning).To(ContainSubstring("expires in"))
		})
		It(
			"should accept a subscription inside the grace window",
			func(ctx SpecContext) {
				lic := newLicense()
				lic.Exp = seconds(now.Add(-2 * day))
				info := MustSucceed(svc.Activate(ctx, sign(lic)))
				Expect(info.State).To(Equal(license.StateOk))
				Expect(info.Warning).To(ContainSubstring("grace"))
			},
		)
		It("should refuse a subscription past the grace window", func(ctx SpecContext) {
			lic := newLicense()
			lic.Exp = seconds(now.Add(-20 * day))
			Expect(svc.Activate(ctx, sign(lic))).Error().
				To(MatchError(license.ErrExpired))
			Expect(svc.Retrieve().State).To(Equal(license.StateMissing))
		})
		perpetual := func(ceiling string) license.License {
			lic := newLicense()
			lic.Exp = nil
			lic.MaxVersion = new(ceiling)
			return lic
		}
		DescribeTable("should accept a perpetual license up to its ceiling",
			func(ctx SpecContext, ceiling string) {
				info := MustSucceed(svc.Activate(ctx, sign(perpetual(ceiling))))
				Expect(info.State).To(Equal(license.StateOk))
				Expect(info.Warning).To(BeEmpty())
			},
			Entry("under", "0.62"),
			Entry("at", "0.60"),
		)
		DescribeTable("should refuse a perpetual license past its ceiling",
			func(ctx SpecContext, ceiling string) {
				Expect(svc.Activate(ctx, sign(perpetual(ceiling)))).Error().To(And(
					MatchError(license.ErrExpired),
					MatchError(ContainSubstring(
						"license covers versions up to %s, this Core is 0.60.1",
						ceiling,
					)),
				))
				Expect(svc.Retrieve().State).To(Equal(license.StateMissing))
			},
			Entry("over", "0.59"),
			Entry("over when compared as numbers", "0.9"),
		)
		It(
			"should refuse a license with neither expiry nor ceiling",
			func(ctx SpecContext) {
				lic := newLicense()
				lic.Exp = nil
				Expect(svc.Activate(ctx, sign(lic))).Error().
					To(MatchError(license.ErrInvalid))
			},
		)
		It("should refuse a ceiling that does not parse", func(ctx SpecContext) {
			lic := newLicense()
			lic.MaxVersion = new("latest")
			Expect(svc.Activate(ctx, sign(lic))).Error().
				To(MatchError(license.ErrInvalid))
		})
		It("should fall back to the ceiling past expiry", func(ctx SpecContext) {
			lic := newLicense()
			lic.Exp = seconds(now.Add(-100 * day))
			lic.MaxVersion = new("0.62")
			info := MustSucceed(svc.Activate(ctx, sign(lic)))
			Expect(info.State).To(Equal(license.StateOk))
			Expect(info.Warning).To(ContainSubstring("subscription ended"))
		})
		It(
			"should refuse a fallback whose ceiling is below this version",
			func(ctx SpecContext) {
				lic := newLicense()
				lic.Exp = seconds(now.Add(-100 * day))
				lic.MaxVersion = new("0.59")
				Expect(svc.Activate(ctx, sign(lic))).Error().
					To(MatchError(license.ErrExpired))
			},
		)
		It("should refuse a license bound to other machines", func(ctx SpecContext) {
			lic := newLicense()
			lic.Fingerprints = []string{"0000"}
			Expect(svc.Activate(ctx, sign(lic))).Error().
				To(MatchError(license.ErrFingerprint))
		})
		It("should accept a license bound to this fingerprint", func(ctx SpecContext) {
			fingerprint := svc.Retrieve().Fingerprint
			if len(fingerprint) == 0 {
				Skip("this machine has no hashable network interface")
			}
			lic := newLicense()
			lic.Fingerprints = []string{"0000", fingerprint[len(fingerprint)-1]}
			MustSucceed(svc.Activate(ctx, sign(lic)))
		})
		It("should refuse hashes from a scheme this Core does not implement", func(
			ctx SpecContext,
		) {
			fingerprint := svc.Retrieve().Fingerprint
			if len(fingerprint) == 0 {
				Skip("this machine has no hashable network interface")
			}
			lic := newLicense()
			lic.FingerprintScheme = 2
			lic.Fingerprints = []string{fingerprint[0]}
			Expect(svc.Activate(ctx, sign(lic))).Error().
				To(MatchError(license.ErrFingerprint))
		})
		It("should reject an invalid token", func(ctx SpecContext) {
			Expect(svc.Activate(ctx, "nope")).Error().
				To(MatchError(license.ErrInvalid))
		})
		It("should enforce the channel cap", func(ctx SpecContext) {
			lic := newLicense()
			lic.Channels = 10
			MustSucceed(svc.Activate(ctx, sign(lic)))
			Expect(svc.CheckChannelLimit(10)).To(Succeed())
			Expect(svc.CheckChannelLimit(11)).To(MatchError(license.ErrTooMany))
		})
		It("should carry the channel cap across the wire", func(ctx SpecContext) {
			lic := newLicense()
			lic.Channels = 10
			MustSucceed(svc.Activate(ctx, sign(lic)))
			pld := errors.Encode(ctx, svc.CheckChannelLimit(11), false)
			Expect(pld.Type).To(Equal("sy.license.too_many"))
			Expect(pld.Data).To(Equal(
				"limit is 10 channels: using more channels than allowed by the " +
					"license: license error",
			))
			Expect(errors.Decode(ctx, pld)).To(And(
				MatchError(license.ErrTooMany),
				MatchError(ContainSubstring("limit is 10 channels")),
			))
		})
		It("should not cap a license with a zero cap", func(ctx SpecContext) {
			MustSucceed(svc.Activate(ctx, sign(newLicense())))
			Expect(svc.CheckChannelLimit(1 << 19)).To(Succeed())
		})
	})
})
