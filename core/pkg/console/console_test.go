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
	"io"
	"net/http"
	"net/http/httptest"
	"testing/fstest"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/console"
	. "github.com/synnaxlabs/x/testutil"
)

var testFS = fstest.MapFS{
	"index.html": &fstest.MapFile{
		Data: []byte(
			`<!doctype html><html><head><title>Synnax Console</title></head><body><div id="root"></div></body></html>`,
		),
	},
	"favicon.svg": &fstest.MapFile{
		Data: []byte(
			`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><rect width="1" height="1"/></svg>`,
		),
	},
}

var _ = Describe("Config", func() {
	Describe("Override", func() {
		It("Should override nil FS with a non-nil value", func() {
			base := console.Config{}
			overridden := base.Override(console.Config{FS: testFS})
			Expect(overridden.FS).To(Equal(testFS))
		})
		It("Should override non-nil FS with another non-nil value", func() {
			other := fstest.MapFS{}
			base := console.Config{FS: testFS}
			overridden := base.Override(console.Config{FS: other})
			Expect(overridden.FS).To(Equal(other))
		})
		It("Should not override non-nil FS with nil", func() {
			base := console.Config{FS: testFS}
			overridden := base.Override(console.Config{})
			Expect(overridden.FS).To(Equal(testFS))
		})
		It("Should do nothing when both are nil", func() {
			base := console.Config{}
			overridden := base.Override(console.Config{})
			Expect(overridden.FS).To(BeNil())
		})
	})

	Describe("Validate", func() {
		It("Should succeed when FS is nil", func() {
			cfg := console.Config{}
			Expect(cfg.Validate()).To(Succeed())
		})
		It("Should succeed when FS is set", func() {
			cfg := console.Config{FS: testFS}
			Expect(cfg.Validate()).To(Succeed())
		})
	})
})

var _ = Describe("Console", func() {
	Describe("New", func() {
		It("Should use the build tag default when no config is provided", func() {
			cons := MustSucceed(console.New())
			report := cons.Report()
			Expect(report["console"]).To(Or(Equal("enabled"), Equal("disabled")))
		})
	})

	Describe("Enabled", func() {
		var (
			cons *console.Console
			app  *fiber.App
		)
		BeforeEach(func() {
			cons = MustSucceed(console.New(console.Config{FS: testFS}))
			app = fiber.New()
			cons.BindTo(app)
		})
		AfterEach(func() {
			Expect(app.Shutdown()).To(Succeed())
		})

		Describe("BindTo", func() {
			It("Should serve index.html at the root", func() {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(http.StatusOK))
				Expect(res.Header.Get(fiber.HeaderContentType)).
					To(ContainSubstring(fiber.MIMETextHTML))
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
				Expect(res.Header.Get(fiber.HeaderContentType)).
					To(ContainSubstring(fiber.MIMETextHTML))
			})
		})

		Describe("Use", func() {
			It("Should not panic", func() {
				Expect(func() { cons.Use() }).ToNot(Panic())
			})
		})

		Describe("Report", func() {
			It("Should report as enabled", func() {
				Expect(cons.Report()["console"]).To(Equal("enabled"))
			})
		})
	})

	Describe("Disabled", func() {
		var (
			cons *console.Console
			app  *fiber.App
		)
		BeforeEach(func() {
			cons = MustSucceed(console.New())
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
				Expect(res.Header.Get(fiber.HeaderContentType)).
					To(ContainSubstring(fiber.MIMETextHTML))
				body := MustSucceed(io.ReadAll(res.Body))
				html := string(body)
				Expect(html).To(ContainSubstring("<title>Synnax Core</title>"))
				Expect(html).To(ContainSubstring("built without embedded Console"))
				Expect(html).To(ContainSubstring("<code>-tags=console</code>"))
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
