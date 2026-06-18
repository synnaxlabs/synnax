// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fhttp "github.com/synnaxlabs/freighter/http"
	thttp "github.com/synnaxlabs/synnax/pkg/transport/http"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("HTTP", func() {
	Describe("Bind", func() {
		var app *fiber.App
		BeforeEach(func() {
			router := MustSucceed(fhttp.NewRouter())
			thttp.Bind(apiLayer, router, svc.Channel)
			app = fiber.New()
			router.BindTo(app)
		})
		AfterEach(func() { Expect(app.Shutdown()).To(Succeed()) })

		DescribeTable("Should register a route for each API endpoint",
			func(method, path string) {
				res := MustSucceed(app.Test(httptest.NewRequest(method, path, nil)))
				Expect(res.StatusCode).ToNot(Equal(http.StatusNotFound))
			},
			Entry("unary auth login", http.MethodPost, "/api/v1/auth/login"),
			Entry("unary channel create", http.MethodPost, "/api/v1/channel/create"),
			Entry("unary frame delete", http.MethodPost, "/api/v1/frame/delete"),
			Entry("stream frame writer", http.MethodGet, "/api/v1/frame/write"),
		)

		It("Should not register a route for an unknown path", func() {
			res := MustSucceed(app.Test(
				httptest.NewRequest(http.MethodPost, "/api/v1/does-not-exist", nil),
			))
			Expect(res.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})
