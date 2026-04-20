// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package console_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/console"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Config", func() {
	Describe("Override", func() {
		It("Should override nil Enabled with a non-nil value", func() {
			base := console.Config{}
			overridden := base.Override(console.Config{Enabled: new(true)})
			Expect(*overridden.Enabled).To(BeTrue())
		})
		It("Should override non-nil Enabled with another non-nil value", func() {
			base := console.Config{Enabled: new(true)}
			overridden := base.Override(console.Config{Enabled: new(false)})
			Expect(*overridden.Enabled).To(BeFalse())
		})
		It("Should not override non-nil Enabled with nil", func() {
			base := console.Config{Enabled: new(true)}
			overridden := base.Override(console.Config{})
			Expect(*overridden.Enabled).To(BeTrue())
		})
		It("Should do nothing when both are nil", func() {
			base := console.Config{}
			overridden := base.Override(console.Config{})
			Expect(overridden.Enabled).To(BeNil())
		})
	})

	Describe("Validate", func() {
		It("Should fail when Enabled is nil", func() {
			cfg := console.Config{}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("enabled"))
		})
		It("Should succeed when Enabled is true", func() {
			cfg := console.Config{Enabled: new(true)}
			Expect(cfg.Validate()).To(Succeed())
		})
		It("Should succeed when Enabled is false", func() {
			cfg := console.Config{Enabled: new(false)}
			Expect(cfg.Validate()).To(Succeed())
		})
	})
})

var _ = Describe("Console", func() {
	Describe("New", func() {
		It("Should use the build tag default when no config is provided", func() {
			svc := MustSucceed(console.New())
			report := svc.Report()
			Expect(report["console"]).To(Or(Equal("enabled"), Equal("disabled")))
		})
	})

	Describe("Enabled", Pending, func() {
		var (
			svc *console.Console
			app *fiber.App
		)
		BeforeEach(func() {
			probe := MustSucceed(console.New(console.Config{Enabled: new(true)}))
			if probe.Report()["console"] != "enabled" {
				Skip("console assets not embedded (build without -tags console)")
			}
			svc = MustSucceed(console.New(console.Config{Enabled: new(true)}))
			app = fiber.New()
			svc.BindTo(app)
		})
		AfterEach(func() {
			if app != nil {
				Expect(app.Shutdown()).To(Succeed())
			}
		})

		Describe("BindTo", func() {
			It("Should serve index.html at the root", func() {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(http.StatusOK))
				Expect(res.Header.Get("Content-Type")).To(ContainSubstring("text/html"))
			})

			It("Should serve static assets", func() {
				req := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(http.StatusOK))
			})

			It("Should serve index.html for SPA routes", func() {
				req := httptest.NewRequest(http.MethodGet, "/some/spa/route", nil)
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(http.StatusOK))
				Expect(res.Header.Get("Content-Type")).To(ContainSubstring("text/html"))
			})
		})

		Describe("Use", func() {
			It("Should not panic", func() {
				Expect(func() { svc.Use() }).ToNot(Panic())
			})
		})

		Describe("Report", func() {
			It("Should report as enabled", func() {
				Expect(svc.Report()["console"]).To(Equal("enabled"))
			})
		})
	})

	Describe("Disabled", func() {
		var (
			cons *console.Console
			app  *fiber.App
		)
		BeforeEach(func() {
			cons = MustSucceed(console.New(console.Config{Enabled: new(false)}))
			app = fiber.New()
			cons.BindTo(app)
		})
		AfterEach(func() {
			Expect(app.Shutdown()).To(Succeed())
		})

		Describe("BindTo", func() {
			It("Should serve the fallback page at the root", func() {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(http.StatusOK))
				Expect(res.Header.Get("Content-Type")).To(ContainSubstring("text/html"))
			})
		})

		Describe("Use", func() {
			It("Should not panic", func() {
				Expect(func() { cons.Use() }).ToNot(Panic())
			})
		})

		Describe("Report", func() {
			It("Should report as disabled", func() {
				Expect(cons.Report()["console"]).To(Equal("disabled"))
			})
		})
	})
})
