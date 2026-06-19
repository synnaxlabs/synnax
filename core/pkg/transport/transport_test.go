// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package transport_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/transport"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Transport", func() {
	Describe("NewLayer", func() {
		DescribeTable("Should return a validation error when a required field is missing",
			func(field string, omit func(*transport.LayerConfig)) {
				cfg := transport.LayerConfig{
					API:     apiLayer,
					Channel: channelSvc,
					Router:  MustSucceed(fhttp.NewRouter()),
				}
				omit(&cfg)
				Expect(transport.NewLayer(cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("missing API", "api", func(c *transport.LayerConfig) { c.API = nil }),
			Entry("missing Channel", "channel", func(c *transport.LayerConfig) { c.Channel = nil }),
			Entry("missing Router", "router", func(c *transport.LayerConfig) { c.Router = nil }),
		)

		It("Should bind the API layer to both transport protocols", func() {
			router := MustSucceed(fhttp.NewRouter())
			tl := MustSucceed(transport.NewLayer(transport.LayerConfig{
				API:     apiLayer,
				Channel: channelSvc,
				Router:  router,
			}))

			Expect(tl.GRPC).To(HaveLen(13))

			app := fiber.New()
			defer func() { Expect(app.Shutdown()).To(Succeed()) }()
			router.BindTo(app)
			res := MustSucceed(app.Test(
				httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil),
			))
			Expect(res.StatusCode).ToNot(Equal(http.StatusNotFound))
		})
	})
})
