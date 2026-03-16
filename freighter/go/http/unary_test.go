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
	"time"

	"github.com/gofiber/fiber/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/httputil"
)

var _ = Describe("Unary", Ordered, Serial, func() {
	var (
		server freighter.UnaryServer[test.Request, test.Response]
		client freighter.UnaryClient[test.Request, test.Response]
		addr   address.Address
		app    *fiber.App
	)

	BeforeAll(func() {
		addr = "localhost:8081"
		app = fiber.New(fiber.Config{DisableStartupMessage: true})
		router := fhttp.NewRouter(fhttp.RouterConfig{})
		factory := fhttp.NewClientFactory(fhttp.ClientFactoryConfig{
			Codec: httputil.JSONCodec,
		})
		app.Get("/health", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})
		server = fhttp.UnaryServer[test.Request, test.Response](router, "/")
		client = fhttp.UnaryClient[test.Request, test.Response](factory)
		router.BindTo(app)
		go func() {
			defer GinkgoRecover()
			Expect(app.Listen(addr.PortString())).To(Succeed())
		}()
		Eventually(func(g Gomega) {
			_, err := http.Get("http://" + addr.String() + "/health")
			g.Expect(err).ToNot(HaveOccurred())
		}).WithPolling(1 * time.Millisecond).Should(Succeed())
	})

	AfterAll(func() { Expect(app.Shutdown()).To(Succeed()) })

	test.UnarySuite(func() (
		freighter.UnaryServer[test.Request, test.Response],
		freighter.UnaryClient[test.Request, test.Response],
		address.Address,
	) {
		return server, client, addr
	})
})
