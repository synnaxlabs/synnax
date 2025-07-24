// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp_test

import (
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/fhttp"
	. "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/httputil"
)

type implementation struct{ app *fiber.App }

var _ UnaryImplementation = &implementation{}

func (i *implementation) Start(host address.Address) (UnaryServer, UnaryClient) {
	i.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	router := fhttp.NewRouter(fhttp.RouterConfig{})
	factory := fhttp.NewClientFactory(fhttp.ClientFactoryConfig{
		Codec: httputil.JSONCodec,
	})
	server := fhttp.UnaryServer[Request, Response](router, "/")
	client := fhttp.UnaryClient[Request, Response](factory)
	router.BindTo(i.app)
	i.app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	go func() {
		if err := i.app.Listen(host.PortString()); err != nil {
			log.Fatal(err)
		}
	}()
	Eventually(func(g Gomega) {
		g.Expect(http.Get("http://" + host.String() + "/health")).
			Error().ToNot(HaveOccurred())
	}).WithPolling(1 * time.Millisecond).Should(Succeed())
	return server, client
}

func (i *implementation) Stop() error { return i.app.Shutdown() }

var _ = Describe("Unary", func() {
	AssertUnary(&implementation{})
})
