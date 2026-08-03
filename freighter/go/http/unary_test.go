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
	"strings"
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
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/net"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	unaryServer                 freighter.UnaryServer[test.Request, test.Response]
	unaryServerJSONOnly         freighter.UnaryServer[test.Request, test.Response]
	unaryServerMsgpackOnly      freighter.UnaryServer[test.Request, test.Response]
	unaryServerFailingEncoder   freighter.UnaryServer[test.Request, test.Response]
	unaryClient                 freighter.UnaryClient[test.Request, test.Response]
	unaryAddr                   address.Address
	unaryApp                    *fiber.App
	errFailingEncoderEncodeFail = errors.New("failing encoder: encode rejected by test")
)

// failingEncoder is a unary-server response encoder whose Encode/EncodeStream methods
// always return an error. It is used to drive the encodeAndWrite error path in
// unary_server.go without modifying production codecs.
type failingEncoder struct{}

func (failingEncoder) ContentType() string { return "application/x-fail" }

func (failingEncoder) Encode(context.Context, any) ([]byte, error) {
	return nil, errFailingEncoderEncodeFail
}

func (failingEncoder) EncodeStream(context.Context, io.Writer, any) error {
	return errFailingEncoderEncodeFail
}

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	unaryAddr = address.Newf("localhost:%d", MustSucceed(net.FindOpenPort()))
	unaryApp = newFiberApp(fiber.Config{DisableKeepalive: true})
	router := MustSucceed(fhttp.NewRouter())
	unaryApp.Get("/health", func(ctx fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusOK)
	})
	// Endpoint that always responds with a content type the default unary client has
	// no decoder for. Used to exercise the client's decoder-resolution failure path
	// against a real server response.
	unaryApp.Post("/text-plain", func(ctx fiber.Ctx) error {
		ctx.Set(fiber.HeaderContentType, "text/plain")
		return ctx.SendString("just text")
	})
	unaryServer = fhttp.NewUnaryServer[test.Request, test.Response](router, "/")
	unaryServerJSONOnly = fhttp.NewUnaryServer[test.Request, test.Response](
		router,
		"/json-only",
		fhttp.WithRequestDecoders(json.Codec),
		fhttp.WithResponseEncoders(json.Codec),
	)
	unaryServerMsgpackOnly = fhttp.NewUnaryServer[test.Request, test.Response](
		router,
		"/msgpack-only",
		fhttp.WithRequestDecoders(msgpack.Codec),
		fhttp.WithResponseEncoders(msgpack.Codec),
	)
	unaryServerFailingEncoder = fhttp.NewUnaryServer[test.Request, test.Response](
		router,
		"/encode-failure",
		fhttp.WithRequestDecoders(json.Codec),
		fhttp.WithResponseEncoders(failingEncoder{}),
	)
	unaryClient = MustSucceed(fhttp.NewUnaryClient[test.Request, test.Response]())
	router.BindTo(unaryApp)
	go func() {
		defer GinkgoRecover()
		Expect(unaryApp.Listen(unaryAddr.PortString(), fiber.ListenConfig{
			DisableStartupMessage: true,
		})).To(Succeed())
	}()
	Eventually(func(g Gomega) {
		g.Expect(pollHealth("http://" + unaryAddr.String() + "/health")).To(Succeed())
	}).WithPolling(1 * time.Millisecond).Should(Succeed())
})

var _ = AfterSuite(func() { Expect(unaryApp.Shutdown()).To(Succeed()) })

var _ = Describe("Unary", func() {
	test.UnarySuite(func() (
		freighter.UnaryServer[test.Request, test.Response],
		freighter.UnaryClient[test.Request, test.Response],
		address.Address,
	) {
		return unaryServer, unaryClient, unaryAddr
	})

	Describe("Content Negotiation", func() {
		bindEcho := func() {
			unaryServer.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
		}
		bindError := func() {
			unaryServer.BindHandler(func(_ context.Context, _ test.Request) (test.Response, error) {
				return test.Response{}, test.ErrCustom
			})
		}
		roundTrip := func(
			ctx context.Context,
			contentType string,
			accept string,
			body []byte,
		) (*http.Response, []byte) {
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx, http.MethodPost, "http://"+unaryAddr.String()+"/", bytes.NewReader(body),
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

		It("should decode JSON request and encode msgpack response when Accept asks for msgpack", func(ctx context.Context) {
			bindEcho()
			req := test.Request{ID: 7, Message: "hello"}
			body := MustSucceed(json.Codec.Encode(ctx, req))
			httpRes, respBody := roundTrip(ctx, "application/json", "application/msgpack", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/msgpack"))
			var got test.Response
			Expect(msgpack.Codec.Decode(ctx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})

		It("should decode msgpack request and encode JSON response when Accept asks for JSON", func(ctx context.Context) {
			bindEcho()
			req := test.Request{ID: 8, Message: "world"}
			body := MustSucceed(msgpack.Codec.Encode(ctx, req))
			httpRes, respBody := roundTrip(ctx, "application/msgpack", "application/json", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var got test.Response
			Expect(json.Codec.Decode(ctx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})

		It("should honor q-values and pick the highest-quality offer", func(ctx context.Context) {
			bindEcho()
			req := test.Request{ID: 1, Message: "q"}
			body := MustSucceed(json.Codec.Encode(ctx, req))
			httpRes, respBody := roundTrip(
				ctx, "application/json",
				"application/json, application/msgpack;q=0.5",
				body,
			)
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var got test.Response
			Expect(json.Codec.Decode(ctx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})

		It("should fall back to the first registered encoder when Accept is omitted", func(ctx context.Context) {
			bindEcho()
			req := test.Request{ID: 2, Message: "no-accept"}
			body := MustSucceed(msgpack.Codec.Encode(ctx, req))
			httpRes, respBody := roundTrip(ctx, "application/msgpack", "", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var got test.Response
			Expect(json.Codec.Decode(ctx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})

		It("should fall back to the first registered encoder when Accept is */*", func(ctx context.Context) {
			bindEcho()
			req := test.Request{ID: 3, Message: "wildcard"}
			body := MustSucceed(json.Codec.Encode(ctx, req))
			httpRes, respBody := roundTrip(ctx, "application/json", "*/*", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var got test.Response
			Expect(json.Codec.Decode(ctx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})

		It("should return 406 Not Acceptable when no registered encoder matches Accept", func(ctx context.Context) {
			bindEcho()
			req := test.Request{ID: 4, Message: "nope"}
			body := MustSucceed(json.Codec.Encode(ctx, req))
			httpRes, _ := roundTrip(ctx, "application/json", "application/octet-stream", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusNotAcceptable))
		})

		It("should encode handler errors via the response codec selected by Accept", func(ctx context.Context) {
			bindError()
			req := test.Request{ID: 5, Message: "err"}
			body := MustSucceed(msgpack.Codec.Encode(ctx, req))
			httpRes, respBody := roundTrip(ctx, "application/msgpack", "application/json", body)
			Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var pld errors.Payload
			Expect(json.Codec.Decode(ctx, respBody, &pld)).To(Succeed())
			Expect(errors.Decode(ctx, pld)).To(MatchError(test.ErrCustom))
		})

		It("should return 415 Unsupported Media Type when the request Content-Type has no registered decoder", func(ctx context.Context) {
			unaryServer.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
			httpRes, _ := roundTrip(
				ctx,
				"application/x-not-a-real-codec",
				"application/json",
				[]byte("anything"),
			)
			Expect(httpRes.StatusCode).To(Equal(http.StatusUnsupportedMediaType))
		})

		It("should return 400 Bad Request with an encoded error payload when the request body fails to decode", func(ctx context.Context) {
			unaryServer.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
			httpRes, respBody := roundTrip(
				ctx,
				"application/json",
				"application/json",
				[]byte("not valid json"),
			)
			Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/json"))
			var pld errors.Payload
			Expect(json.Codec.Decode(ctx, respBody, &pld)).To(Succeed())
			Expect(pld.Type).ToNot(Equal(errors.TypeNil))
		})

		It("should return 500 Internal Server Error when the response encoder fails to encode the handler's result", func(ctx context.Context) {
			unaryServerFailingEncoder.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
			body := MustSucceed(json.Codec.Encode(ctx, test.Request{ID: 9, Message: "encode-fail"}))
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx, http.MethodPost, "http://"+unaryAddr.String()+"/encode-failure",
				bytes.NewReader(body),
			))
			httpReq.Header.Set(fiber.HeaderContentType, "application/json")
			httpReq.Header.Set(fiber.HeaderAccept, "application/x-fail")
			httpRes := MustSucceed((&http.Client{}).Do(httpReq))
			DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
			Expect(httpRes.StatusCode).To(Equal(http.StatusInternalServerError))
			respBody := MustSucceed(io.ReadAll(httpRes.Body))
			Expect(string(respBody)).To(ContainSubstring(errFailingEncoderEncodeFail.Error()))
		})
	})

	Describe("Query Params", func() {
		// Binds a handler that echoes the named request params back in the response
		// message, joined by "|", so the test can observe what parseRequestCtx put in
		// freighter.Context.Params.
		bindParamEcho := func(keys ...string) {
			unaryServer.BindHandler(func(ctx context.Context, _ test.Request) (test.Response, error) {
				params := freighter.MDFromContext(ctx).Params
				values := make([]string, len(keys))
				for i, k := range keys {
					if v, ok := params.Get(k); ok {
						values[i], _ = v.(string)
					}
				}
				return test.Response{Message: strings.Join(values, "|")}, nil
			})
		}
		post := func(ctx context.Context, query string) test.Response {
			body := MustSucceed(json.Codec.Encode(ctx, test.Request{}))
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx, http.MethodPost,
				"http://"+unaryAddr.String()+"/?"+query,
				bytes.NewReader(body),
			))
			httpReq.Header.Set(fiber.HeaderContentType, "application/json")
			httpRes := MustSucceed((&http.Client{}).Do(httpReq))
			DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
			respBody := MustSucceed(io.ReadAll(httpRes.Body))
			var res test.Response
			Expect(json.Codec.Decode(ctx, respBody, &res)).To(Succeed())
			return res
		}

		It("should expose freighterctx-prefixed query params with the prefix stripped", func(ctx context.Context) {
			bindParamEcho("file_name", "project")
			res := post(
				ctx,
				"freighterctxfile_name=Metrics%20Log.json&freighterctxproject=project:abc",
			)
			Expect(res.Message).To(Equal("Metrics Log.json|project:abc"))
		})

		It("should not expose unprefixed query params to the handler", func(ctx context.Context) {
			bindParamEcho("file_name", "project")
			res := post(ctx, "file_name=Metrics%20Log.json&project=project:abc")
			Expect(res.Message).To(Equal("|"))
		})
	})

	Describe("Codec Configuration", func() {
		It("should restrict the request decoders to the codecs passed via WithRequestDecoders", func(ctx context.Context) {
			unaryServerJSONOnly.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
			req := test.Request{ID: 1, Message: "json-only"}
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx, http.MethodPost, "http://"+unaryAddr.String()+"/json-only",
				bytes.NewReader(MustSucceed(msgpack.Codec.Encode(ctx, req))),
			))
			httpReq.Header.Set(fiber.HeaderContentType, "application/msgpack")
			httpReq.Header.Set(fiber.HeaderAccept, "application/json")
			httpRes := MustSucceed((&http.Client{}).Do(httpReq))
			DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
			Expect(httpRes.StatusCode).To(Equal(http.StatusUnsupportedMediaType))
		})

		It("should restrict the response encoders to the codecs passed via WithResponseEncoders", func(ctx context.Context) {
			unaryServerMsgpackOnly.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
			req := test.Request{ID: 2, Message: "msgpack-only"}
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx, http.MethodPost, "http://"+unaryAddr.String()+"/msgpack-only",
				bytes.NewReader(MustSucceed(msgpack.Codec.Encode(ctx, req))),
			))
			httpReq.Header.Set(fiber.HeaderContentType, "application/msgpack")
			httpReq.Header.Set(fiber.HeaderAccept, "application/json")
			httpRes := MustSucceed((&http.Client{}).Do(httpReq))
			DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
			Expect(httpRes.StatusCode).To(Equal(http.StatusNotAcceptable))
		})

		It("should round-trip end-to-end when the client and server agree on the restricted codec", func(ctx context.Context) {
			unaryServerMsgpackOnly.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
			req := test.Request{ID: 3, Message: "round-trip"}
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx, http.MethodPost, "http://"+unaryAddr.String()+"/msgpack-only",
				bytes.NewReader(MustSucceed(msgpack.Codec.Encode(ctx, req))),
			))
			httpReq.Header.Set(fiber.HeaderContentType, "application/msgpack")
			httpReq.Header.Set(fiber.HeaderAccept, "application/msgpack")
			httpRes := MustSucceed((&http.Client{}).Do(httpReq))
			DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
			Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			Expect(httpRes.Header.Get(fiber.HeaderContentType)).To(Equal("application/msgpack"))
			respBody := MustSucceed(io.ReadAll(httpRes.Body))
			var got test.Response
			Expect(msgpack.Codec.Decode(ctx, respBody, &got)).To(Succeed())
			Expect(got).To(Equal(test.Response(req)))
		})
	})

	Describe("Report", func() {
		It("should report the unary server's protocol and accepted/emitted content types", func() {
			report := unaryServer.Report()
			Expect(report["protocol"]).To(Equal("http"))
			Expect(report["acceptedContentTypes"]).To(Equal([]string{
				"application/json", "application/msgpack",
			}))
			Expect(report["emittedContentTypes"]).To(Equal([]string{
				"application/json", "application/msgpack",
			}))
		})

		It("should reflect WithRequestDecoders and WithResponseEncoders in the server report", func() {
			report := unaryServerJSONOnly.Report()
			Expect(report["acceptedContentTypes"]).To(Equal([]string{"application/json"}))
			Expect(report["emittedContentTypes"]).To(Equal([]string{"application/json"}))
		})

		It("should report the unary client's protocol, sent Content-Type, and accepted Content-Types", func() {
			report := unaryClient.Report()
			Expect(report["protocol"]).To(Equal("http"))
			Expect(report["sentContentType"]).To(Equal("application/json"))
			Expect(report["acceptedContentTypes"]).To(Equal([]string{
				"application/json", "application/msgpack",
			}))
		})

		It("should reflect a custom Encoder in the client report", func() {
			c := MustSucceed(fhttp.NewUnaryClient[test.Request, test.Response](
				fhttp.UnaryClientConfig{
					Encoder:  msgpack.Codec,
					Decoders: []xhttp.Decoder{msgpack.Codec},
				},
			))
			report := c.Report()
			Expect(report["sentContentType"]).To(Equal("application/msgpack"))
			Expect(report["acceptedContentTypes"]).To(Equal([]string{"application/msgpack"}))
		})
	})

	Describe("Client Decoder Resolution", func() {
		It("should fail with an unresolved-decoder error when the server response Content-Type has no registered client decoder", func(ctx context.Context) {
			_, err := unaryClient.Send(ctx, unaryAddr+"/text-plain", test.Request{ID: 1, Message: "x"})
			Expect(err).To(MatchError(ContainSubstring("text/plain")))
		})
	})
})
