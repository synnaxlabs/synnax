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

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ihttp "github.com/synnaxlabs/freighter/integration/http"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("BindTo", func() {
	It("Should register every integration endpoint on the provided Fiber app", func() {
		app := fiber.New(fiber.Config{})
		Expect(ihttp.BindTo(app)).To(Succeed())
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
			"/unary/flakyUnavailable",
		}
		for _, path := range routes {
			Expect(registered.Contains(path)).To(BeTrue())
		}
	})

	It("Should respond to /unary/echo with the request payload and the ID incremented", func() {
		app := fiber.New(fiber.Config{})
		Expect(ihttp.BindTo(app)).To(Succeed())

		body := MustSucceed(json.Marshal(ihttp.Message{Message: "hello", ID: 1}))
		req := MustSucceed(http.NewRequest(http.MethodPost, "http://localhost/unary/echo", bytes.NewReader(body)))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		req.Header.Set(fiber.HeaderAccept, fiber.MIMEApplicationJSON)

		resp := MustSucceed(app.Test(req))
		DeferCleanup(func() { Expect(resp.Body.Close()).To(Succeed()) })
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		var msg ihttp.Message
		Expect(json.NewDecoder(resp.Body).Decode(&msg)).To(Succeed())
		Expect(msg).To(Equal(ihttp.Message{Message: "hello", ID: 2}))
	})

	It("Should reject /unary/middlewareCheck when the Test header is missing", func() {
		app := fiber.New(fiber.Config{})
		Expect(ihttp.BindTo(app)).To(Succeed())

		body := MustSucceed(json.Marshal(ihttp.Message{Message: "hello", ID: 1}))
		req := MustSucceed(http.NewRequest(http.MethodPost, "http://localhost/unary/middlewareCheck", bytes.NewReader(body)))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		req.Header.Set(fiber.HeaderAccept, fiber.MIMEApplicationJSON)

		resp := MustSucceed(app.Test(req))
		DeferCleanup(func() { Expect(resp.Body.Close()).To(Succeed()) })
		Expect(resp.StatusCode).ToNot(Equal(http.StatusOK))
	})

	It("Should pass /unary/middlewareCheck when the Test header is set to 'test'", func() {
		app := fiber.New(fiber.Config{})
		Expect(ihttp.BindTo(app)).To(Succeed())

		body := MustSucceed(json.Marshal(ihttp.Message{Message: "hello", ID: 7}))
		req := MustSucceed(http.NewRequest(http.MethodPost, "http://localhost/unary/middlewareCheck", bytes.NewReader(body)))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		req.Header.Set(fiber.HeaderAccept, fiber.MIMEApplicationJSON)
		req.Header.Set("Test", "test")

		resp := MustSucceed(app.Test(req))
		DeferCleanup(func() { Expect(resp.Body.Close()).To(Succeed()) })
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		var msg ihttp.Message
		Expect(json.NewDecoder(resp.Body).Decode(&msg)).To(Succeed())
		Expect(msg.ID).To(Equal(8))
	})
})

var _ = Describe("flakyUnavailable", func() {
	post := func(app *fiber.App, msg ihttp.Message) *http.Response {
		body := MustSucceed(json.Marshal(msg))
		req := MustSucceed(http.NewRequest(http.MethodPost, "http://localhost/unary/flakyUnavailable", bytes.NewReader(body)))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		req.Header.Set(fiber.HeaderAccept, fiber.MIMEApplicationJSON)
		resp := MustSucceed(app.Test(req))
		DeferCleanup(func() { Expect(resp.Body.Close()).To(Succeed()) })
		return resp
	}

	It("Should respond with a retryable 503 the first time it sees a message", func() {
		app := fiber.New(fiber.Config{})
		Expect(ihttp.BindTo(app)).To(Succeed())

		resp := post(app, ihttp.Message{Message: "flaky-first-503", ID: 1})
		Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))

		var pld errors.Payload
		Expect(json.NewDecoder(resp.Body).Decode(&pld)).To(Succeed())
		Expect(pld.Type).To(Equal("integration.error"))
	})

	It("Should echo nominally with the ID incremented when the same message is retried", func() {
		app := fiber.New(fiber.Config{})
		Expect(ihttp.BindTo(app)).To(Succeed())

		first := post(app, ihttp.Message{Message: "flaky-recovers", ID: 1})
		Expect(first.StatusCode).To(Equal(http.StatusServiceUnavailable))

		second := post(app, ihttp.Message{Message: "flaky-recovers", ID: 1})
		Expect(second.StatusCode).To(Equal(http.StatusOK))
		var msg ihttp.Message
		Expect(json.NewDecoder(second.Body).Decode(&msg)).To(Succeed())
		Expect(msg).To(Equal(ihttp.Message{Message: "flaky-recovers", ID: 2}))
	})

	It("Should track the first-seen state of each message independently", func() {
		app := fiber.New(fiber.Config{})
		Expect(ihttp.BindTo(app)).To(Succeed())

		Expect(post(app, ihttp.Message{Message: "flaky-a", ID: 1}).StatusCode).
			To(Equal(http.StatusServiceUnavailable))
		Expect(post(app, ihttp.Message{Message: "flaky-b", ID: 1}).StatusCode).
			To(Equal(http.StatusServiceUnavailable))
		Expect(post(app, ihttp.Message{Message: "flaky-a", ID: 1}).StatusCode).
			To(Equal(http.StatusOK))
	})
})

var _ = Describe("TestError", func() {
	It("Should expose its message via Error", func() {
		err := ihttp.TestError{Code: 42, Message: "something broke"}
		Expect(err.Error()).To(Equal("something broke"))
	})

	It("Should round-trip through the freighter errors registry", func() {
		original := ihttp.TestError{Code: 7, Message: "boom"}
		payload := errors.Encode(context.Background(), original, false)
		Expect(payload.Type).To(Equal("integration.error"))

		decoded := errors.Decode(context.Background(), payload)
		var got ihttp.TestError
		Expect(errors.As(decoded, &got)).To(BeTrue())
		Expect(got).To(Equal(original))
	})
})
