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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fhttp "github.com/synnaxlabs/freighter/integration/http"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/net"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	app  *fiber.App
	addr address.Address
)

var _ = BeforeSuite(func() {
	addr = address.Newf("localhost:%d", MustSucceed(net.FindOpenPort()))
	app = fiber.New(fiber.Config{})
	Expect(fhttp.BindTo(app)).To(Succeed())
	go func() {
		defer GinkgoRecover()
		Expect(app.Listen(addr.PortString(), fiber.ListenConfig{
			DisableStartupMessage: true,
		})).To(Succeed())
	}()
	Eventually(func(g Gomega) {
		_, err := http.Get("http://" + addr.String() + "/unary/echo")
		g.Expect(err).To(Succeed())
	}).WithPolling(2 * time.Millisecond).WithTimeout(2 * time.Second).Should(Succeed())
})

var _ = AfterSuite(func() { Expect(app.Shutdown()).To(Succeed()) })

func post(
	ctx context.Context,
	path string,
	headers map[string]string,
	body []byte,
) *http.Response {
	req := MustSucceed(http.NewRequestWithContext(
		ctx, http.MethodPost, "http://"+addr.String()+path, bytes.NewReader(body),
	))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res := MustSucceed((&http.Client{}).Do(req))
	DeferCleanup(func() { Expect(res.Body.Close()).To(Succeed()) })
	return res
}

var _ = Describe("BindTo", func() {
	It("Should register every integration endpoint on the provided Fiber app", func() {
		registered := set.New[string]()
		for _, r := range app.GetRoutes() {
			registered = registered.Add(r.Path)
		}
		routes := []string{
			"/stream/echo",
			"/stream/sendMessageAfterClientClose",
			"/stream/receiveAndExitWithErr",
			"/stream/immediatelyExitWithErr",
			"/stream/immediatelyExitNominally",
			"/stream/respondWithTenMessages",
			"/stream/middlewareCheck",
			"/stream/slamMessages",
			"/stream/eventuallyResponseWithMessage",
			"/unary/echo",
			"/unary/middlewareCheck",
			"/unary/slamMessagesTimeoutCheck",
		}
		for _, path := range routes {
			Expect(registered.Contains(path)).To(BeTrue())
		}
	})

	Describe("/unary/echo", func() {
		It("Should increment the request ID and echo the message", func(ctx context.Context) {
			body := MustSucceed(json.Marshal(fhttp.Message{Message: "hi", ID: 1}))
			res := post(ctx, "/unary/echo", map[string]string{
				fiber.HeaderContentType: "application/json",
			}, body)
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			var got fhttp.Message
			Expect(json.NewDecoder(res.Body).Decode(&got)).To(Succeed())
			Expect(got.ID).To(Equal(2))
			Expect(got.Message).To(Equal("hi"))
		})
	})

	Describe("/unary/middlewareCheck", func() {
		It("Should pass through when the Test header is set", func(ctx context.Context) {
			body := MustSucceed(json.Marshal(fhttp.Message{Message: "hi", ID: 1}))
			res := post(ctx, "/unary/middlewareCheck", map[string]string{
				fiber.HeaderContentType: "application/json",
				"Test":                  "test",
			}, body)
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			var got fhttp.Message
			Expect(json.NewDecoder(res.Body).Decode(&got)).To(Succeed())
			Expect(got.ID).To(Equal(2))
			Expect(got.Message).To(Equal("hi"))
		})

		It("Should return the registered TestError when the Test header is absent", func(ctx context.Context) {
			body := MustSucceed(json.Marshal(fhttp.Message{Message: "hi", ID: 1}))
			res := post(ctx, "/unary/middlewareCheck", map[string]string{
				fiber.HeaderContentType: "application/json",
			}, body)
			Expect(res.StatusCode).To(Equal(http.StatusBadRequest))
			var pld errors.Payload
			Expect(json.NewDecoder(res.Body).Decode(&pld)).To(Succeed())
			Expect(pld.Type).To(Equal("integration.error"))
			Expect(pld.Data).To(Equal("1,test param not found"))
		})

		It("Should reject a wrong Test header value", func(ctx context.Context) {
			body := MustSucceed(json.Marshal(fhttp.Message{Message: "hi", ID: 1}))
			res := post(ctx, "/unary/middlewareCheck", map[string]string{
				fiber.HeaderContentType: "application/json",
				"Test":                  "not-test",
			}, body)
			Expect(res.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Unknown path", func() {
		It("Should return 404", func(ctx context.Context) {
			res := post(ctx, "/unary/does-not-exist", nil, nil)
			Expect(res.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})

var _ = Describe("TestError", func() {
	It("Should expose its message via Error", func() {
		err := fhttp.TestError{Code: 42, Message: "something broke"}
		Expect(err.Error()).To(Equal("something broke"))
	})

	It("Should round-trip through the freighter errors registry", func() {
		original := fhttp.TestError{Code: 7, Message: "boom"}
		payload := errors.Encode(context.Background(), original, false)
		Expect(payload.Type).To(Equal("integration.error"))

		decoded := errors.Decode(context.Background(), payload)
		var got fhttp.TestError
		Expect(errors.As(decoded, &got)).To(BeTrue())
		Expect(got).To(Equal(original))
	})
})
