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
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
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
		app = fiber.New(fiber.Config{})
		router := MustSucceed(fhttp.NewRouter(fhttp.RouterConfig{}))
		app.Get("/health", func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})
		server = fhttp.NewUnaryServer[test.Request, test.Response](router, "/")
		client = MustSucceed(fhttp.NewUnaryClient[test.Request, test.Response](
			fhttp.UnaryClientConfig{
				Encoder:  fhttp.JSONCodec,
				Decoders: []fhttp.Decoder{fhttp.JSONCodec},
			},
		))
		router.BindTo(app)
		go func() {
			defer GinkgoRecover()
			Expect(app.Listen(addr.PortString(), fiber.ListenConfig{
				DisableStartupMessage: true,
			})).To(Succeed())
		}()
		Eventually(func(g Gomega) {
			_, err := http.Get("http://" + addr.String() + "/health")
			g.Expect(err).To(Succeed())
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

	Describe("Content Negotiation", func() {
		bgCtx := context.Background()

		bindEcho := func() {
			server.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
		}
		bindError := func() {
			server.BindHandler(func(_ context.Context, _ test.Request) (test.Response, error) {
				return test.Response{}, test.ErrCustom
			})
		}

		roundTrip := func(
			contentType string,
			accept string,
			body []byte,
		) (*http.Response, []byte) {
			httpReq := MustSucceed(http.NewRequestWithContext(
				bgCtx, "POST", "http://"+addr.String()+"/", bytes.NewReader(body),
			))
			httpReq.Header.Set(fiber.HeaderContentType, contentType)
			if accept != "" {
				httpReq.Header.Set(fiber.HeaderAccept, accept)
			}
			httpRes := MustSucceed((&http.Client{}).Do(httpReq))
			DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
			respBody := MustSucceed(io.ReadAll(httpRes.Body))
			return httpRes, respBody
		}

		It("should decode JSON request and encode msgpack response when Accept asks for msgpack", func() {
			bindEcho()
			req := test.Request{ID: 7, Message: "hello"}
			body := MustSucceed(json.Codec.Encode(bgCtx, req))
			httpRes, respBody := roundTrip("application/json", "application/msgpack", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/msgpack"))
			var got test.Response
			Expect(msgpack.Codec.Decode(bgCtx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})

		It("should decode msgpack request and encode JSON response when Accept asks for JSON", func() {
			bindEcho()
			req := test.Request{ID: 8, Message: "world"}
			body := MustSucceed(msgpack.Codec.Encode(bgCtx, req))
			httpRes, respBody := roundTrip("application/msgpack", "application/json", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var got test.Response
			Expect(json.Codec.Decode(bgCtx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})

		It("should honor q-values and pick the highest-quality offer", func() {
			bindEcho()
			req := test.Request{ID: 1, Message: "q"}
			body := MustSucceed(json.Codec.Encode(bgCtx, req))
			httpRes, respBody := roundTrip(
				"application/json",
				"application/json, application/msgpack;q=0.5",
				body,
			)
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var got test.Response
			Expect(json.Codec.Decode(bgCtx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})

		It("should fall back to the first registered encoder when Accept is omitted", func() {
			bindEcho()
			req := test.Request{ID: 2, Message: "no-accept"}
			body := MustSucceed(msgpack.Codec.Encode(bgCtx, req))
			httpRes, respBody := roundTrip("application/msgpack", "", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var got test.Response
			Expect(json.Codec.Decode(bgCtx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})

		It("should fall back to the first registered encoder when Accept is */*", func() {
			bindEcho()
			req := test.Request{ID: 3, Message: "wildcard"}
			body := MustSucceed(json.Codec.Encode(bgCtx, req))
			httpRes, respBody := roundTrip("application/json", "*/*", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var got test.Response
			Expect(json.Codec.Decode(bgCtx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})

		It("should return 406 Not Acceptable when no registered encoder matches Accept", func() {
			bindEcho()
			req := test.Request{ID: 4, Message: "nope"}
			body := MustSucceed(json.Codec.Encode(bgCtx, req))
			httpRes, _ := roundTrip("application/json", "application/octet-stream", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusNotAcceptable))
		})

		It("should encode handler errors via the response codec selected by Accept", func() {
			bindError()
			req := test.Request{ID: 5, Message: "err"}
			body := MustSucceed(msgpack.Codec.Encode(bgCtx, req))
			httpRes, respBody := roundTrip("application/msgpack", "application/json", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var pld errors.Payload
			Expect(json.Codec.Decode(bgCtx, respBody, &pld)).To(Succeed())
			Expect(errors.Decode(bgCtx, pld)).To(MatchError(test.ErrCustom))
		})
	})
})
