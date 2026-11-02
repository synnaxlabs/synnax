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
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	xhttp "github.com/synnaxlabs/x/http"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	unaryServer                 freighter.UnaryServer[testutil.Request, testutil.Response]
	unaryServerJSONOnly         freighter.UnaryServer[testutil.Request, testutil.Response]
	unaryServerMsgpackOnly      freighter.UnaryServer[testutil.Request, testutil.Response]
	unaryServerFailingEncoder   freighter.UnaryServer[testutil.Request, testutil.Response]
	unaryServerStreaming        freighter.UnaryServer[testutil.Request, testutil.Response]
	unaryServerClosing          freighter.UnaryServer[testutil.Request, *closingResponse]
	closingResponses            = make(chan *closingResponse, 1)
	unaryClient                 freighter.UnaryClient[testutil.Request, testutil.Response]
	unaryAddr                   address.Address
	unaryApp                    *fiber.App
	errFailingEncoderEncodeFail = errors.New("failing encoder: encode rejected by test")
	errTruncatingEncoderFail    = errors.New("truncating encoder: failed partway")
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

// repeatingEncoder is a unary-server response encoder that writes the response message
// once per byte of its ID, in as many writes. It is used to drive the streaming
// response path in unary_server.go.
type repeatingEncoder struct{}

func (repeatingEncoder) ContentType() string { return "application/x-repeat" }

func (e repeatingEncoder) Encode(ctx context.Context, v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(ctx, &buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (repeatingEncoder) EncodeStream(_ context.Context, w io.Writer, v any) error {
	res, ok := v.(testutil.Response)
	if !ok {
		return errors.Newf("repeating encoder cannot encode %T", v)
	}
	for range res.ID {
		if _, err := w.Write([]byte(res.Message)); err != nil {
			return err
		}
	}
	return nil
}

// truncatingEncoder is a unary-server response encoder that writes the response
// message once per byte of its ID, then fails.
type truncatingEncoder struct{}

func (truncatingEncoder) ContentType() string { return "application/x-truncate" }

func (e truncatingEncoder) Encode(ctx context.Context, v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(ctx, &buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (truncatingEncoder) EncodeStream(ctx context.Context, w io.Writer, v any) error {
	if err := (repeatingEncoder{}).EncodeStream(ctx, w, v); err != nil {
		return err
	}
	return errTruncatingEncoderFail
}

// echoingEncoder is a unary-server response encoder that writes the response message
// once. The streaming route leaves it out of WithStreamingResponse, so it drives the
// uncompressed half of the streaming path.
type echoingEncoder struct{}

func (echoingEncoder) ContentType() string { return "application/x-echo" }

func (e echoingEncoder) Encode(ctx context.Context, v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(ctx, &buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (echoingEncoder) EncodeStream(_ context.Context, w io.Writer, v any) error {
	res, ok := v.(testutil.Response)
	if !ok {
		return errors.Newf("echoing encoder cannot encode %T", v)
	}
	_, err := w.Write([]byte(res.Message))
	return err
}

// failAfterHandlerHeader asks the closing route to fail after its handler returns.
const failAfterHandlerHeader = "Fail-After-Handler"

// closingResponse is a unary-server response that records whether it was closed.
type closingResponse struct{ closed atomic.Bool }

// Close implements io.Closer.
func (r *closingResponse) Close() error {
	r.closed.Store(true)
	return nil
}

// closingEncoder is a unary-server response encoder that writes a fixed body for a
// closingResponse.
type closingEncoder struct{}

func (closingEncoder) ContentType() string { return "application/x-closing" }

func (e closingEncoder) Encode(ctx context.Context, v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(ctx, &buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (closingEncoder) EncodeStream(_ context.Context, w io.Writer, v any) error {
	if _, ok := v.(*closingResponse); !ok {
		return errors.Newf("closing encoder cannot encode %T", v)
	}
	_, err := w.Write([]byte("ok"))
	return err
}

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
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
	unaryServer = router.NewUnaryServer[testutil.Request, testutil.Response]("/")
	unaryServerJSONOnly = router.NewUnaryServer[testutil.Request, testutil.Response](
		"/json-only",
		fhttp.WithRequestDecoders(json.Codec),
		fhttp.WithResponseEncoders(json.Codec),
	)
	unaryServerMsgpackOnly = router.NewUnaryServer[testutil.Request, testutil.Response](
		"/msgpack-only",
		fhttp.WithRequestDecoders(msgpack.Codec),
		fhttp.WithResponseEncoders(msgpack.Codec),
	)
	unaryServerFailingEncoder = router.NewUnaryServer[testutil.Request, testutil.Response](
		"/encode-failure",
		fhttp.WithRequestDecoders(json.Codec),
		fhttp.WithResponseEncoders(failingEncoder{}),
	)
	unaryServerStreaming = router.NewUnaryServer[testutil.Request, testutil.Response](
		"/streaming",
		fhttp.WithRequestDecoders(json.Codec),
		fhttp.WithResponseEncoders(
			repeatingEncoder{},
			echoingEncoder{},
			truncatingEncoder{},
		),
		fhttp.WithStreamingResponse(repeatingEncoder{}, truncatingEncoder{}),
	)
	unaryServerStreaming.BindHandler(
		func(_ context.Context, req testutil.Request) (testutil.Response, error) {
			return testutil.Response(req), nil
		},
	)
	unaryServerClosing = router.NewUnaryServer[testutil.Request, *closingResponse](
		"/closing",
		fhttp.WithRequestDecoders(json.Codec),
		fhttp.WithResponseEncoders(closingEncoder{}),
		fhttp.WithStreamingResponse(),
	)
	unaryServerClosing.BindHandler(
		func(context.Context, testutil.Request) (*closingResponse, error) {
			res := &closingResponse{}
			closingResponses <- res
			return res, nil
		},
	)
	unaryServerClosing.Use(freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		oCtx, err := next(ctx)
		if _, fail := ctx.Get(failAfterHandlerHeader); fail && err == nil {
			err = errors.New("failed after the handler")
		}
		return oCtx, err
	}))
	unaryClient = MustSucceed(
		fhttp.NewUnaryClient[testutil.Request, testutil.Response](),
	)
	router.BindTo(unaryApp)
	unaryAddr = serveApp(unaryApp)
	Eventually(func(g Gomega) {
		g.Expect(pollHealth("http://" + unaryAddr.String() + "/health")).To(Succeed())
	}).WithPolling(1 * time.Millisecond).Should(Succeed())
})

var _ = Describe("Unary", func() {
	testutil.UnarySuite(func() (
		freighter.UnaryServer[testutil.Request, testutil.Response],
		freighter.UnaryClient[testutil.Request, testutil.Response],
		address.Address,
	) {
		return unaryServer, unaryClient, unaryAddr
	})

	Describe("Content Negotiation", func() {
		bindEcho := func() {
			unaryServer.BindHandler(
				func(_ context.Context, req testutil.Request) (testutil.Response, error) {
					return testutil.Response(req), nil
				},
			)
		}
		bindError := func() {
			unaryServer.BindHandler(
				func(_ context.Context, _ testutil.Request) (testutil.Response, error) {
					return testutil.Response{}, testutil.ErrCustom
				},
			)
		}
		roundTrip := func(
			ctx context.Context,
			contentType string,
			accept string,
			body []byte,
		) (*http.Response, []byte) {
			GinkgoHelper()
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx,
				http.MethodPost,
				"http://"+unaryAddr.String()+"/",
				bytes.NewReader(body),
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

		It(
			"should decode JSON request and encode msgpack response when Accept asks for msgpack",
			func(ctx context.Context) {
				bindEcho()
				req := testutil.Request{ID: 7, Message: "hello"}
				body := MustSucceed(json.Codec.Encode(ctx, req))
				httpRes, respBody := roundTrip(
					ctx,
					"application/json",
					"application/msgpack",
					body,
				)
				Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentType),
				).To(Equal("application/msgpack"))
				var got testutil.Response
				Expect(msgpack.Codec.Decode(ctx, respBody, &got)).To(Succeed())
				Expect(got).To(Equal(testutil.Response(req)))
			},
		)

		It(
			"should decode msgpack request and encode JSON response when Accept asks for JSON",
			func(ctx context.Context) {
				bindEcho()
				req := testutil.Request{ID: 8, Message: "world"}
				body := MustSucceed(msgpack.Codec.Encode(ctx, req))
				httpRes, respBody := roundTrip(
					ctx,
					"application/msgpack",
					"application/json",
					body,
				)
				Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentType),
				).To(Equal("application/json"))
				var got testutil.Response
				Expect(json.Codec.Decode(ctx, respBody, &got)).To(Succeed())
				Expect(got).To(Equal(testutil.Response(req)))
			},
		)

		It(
			"should honor q-values and pick the highest-quality offer",
			func(ctx context.Context) {
				bindEcho()
				req := testutil.Request{ID: 1, Message: "q"}
				body := MustSucceed(json.Codec.Encode(ctx, req))
				httpRes, respBody := roundTrip(
					ctx, "application/json",
					"application/json, application/msgpack;q=0.5",
					body,
				)
				Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentType),
				).To(Equal("application/json"))
				var got testutil.Response
				Expect(json.Codec.Decode(ctx, respBody, &got)).To(Succeed())
				Expect(got).To(Equal(testutil.Response(req)))
			},
		)

		It(
			"should fall back to the first registered encoder when Accept is omitted",
			func(ctx context.Context) {
				bindEcho()
				req := testutil.Request{ID: 2, Message: "no-accept"}
				body := MustSucceed(msgpack.Codec.Encode(ctx, req))
				httpRes, respBody := roundTrip(ctx, "application/msgpack", "", body)
				Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentType),
				).To(Equal("application/json"))
				var got testutil.Response
				Expect(json.Codec.Decode(ctx, respBody, &got)).To(Succeed())
				Expect(got).To(Equal(testutil.Response(req)))
			},
		)

		It(
			"should fall back to the first registered encoder when Accept is */*",
			func(ctx context.Context) {
				bindEcho()
				req := testutil.Request{ID: 3, Message: "wildcard"}
				body := MustSucceed(json.Codec.Encode(ctx, req))
				httpRes, respBody := roundTrip(ctx, "application/json", "*/*", body)
				Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentType),
				).To(Equal("application/json"))
				var got testutil.Response
				Expect(json.Codec.Decode(ctx, respBody, &got)).To(Succeed())
				Expect(got).To(Equal(testutil.Response(req)))
			},
		)

		It(
			"should return 406 Not Acceptable when no registered encoder matches Accept",
			func(ctx context.Context) {
				bindEcho()
				req := testutil.Request{ID: 4, Message: "nope"}
				body := MustSucceed(json.Codec.Encode(ctx, req))
				httpRes, _ := roundTrip(
					ctx,
					"application/json",
					"application/octet-stream",
					body,
				)
				Expect(httpRes.StatusCode).To(Equal(http.StatusNotAcceptable))
			},
		)

		It(
			"should encode handler errors via the response codec selected by Accept",
			func(ctx context.Context) {
				bindError()
				req := testutil.Request{ID: 5, Message: "err"}
				body := MustSucceed(msgpack.Codec.Encode(ctx, req))
				httpRes, respBody := roundTrip(
					ctx,
					"application/msgpack",
					"application/json",
					body,
				)
				Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentType),
				).To(Equal("application/json"))
				var pld errors.Payload
				Expect(json.Codec.Decode(ctx, respBody, &pld)).To(Succeed())
				Expect(errors.Decode(ctx, pld)).To(MatchError(testutil.ErrCustom))
			},
		)

		It(
			"should return 415 Unsupported Media Type when the request Content-Type has no registered decoder",
			func(ctx context.Context) {
				unaryServer.BindHandler(
					func(_ context.Context, req testutil.Request) (testutil.Response, error) {
						return testutil.Response(req), nil
					},
				)
				httpRes, _ := roundTrip(
					ctx,
					"application/x-not-a-real-codec",
					"application/json",
					[]byte("anything"),
				)
				Expect(httpRes.StatusCode).To(Equal(http.StatusUnsupportedMediaType))
			},
		)

		It(
			"should return 400 Bad Request with an encoded error payload when the request body fails to decode",
			func(ctx context.Context) {
				unaryServer.BindHandler(
					func(_ context.Context, req testutil.Request) (testutil.Response, error) {
						return testutil.Response(req), nil
					},
				)
				httpRes, respBody := roundTrip(
					ctx,
					"application/json",
					"application/json",
					[]byte("not valid json"),
				)
				Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentType),
				).To(Equal("application/json"))
				var pld errors.Payload
				Expect(json.Codec.Decode(ctx, respBody, &pld)).To(Succeed())
				Expect(pld.Type).ToNot(Equal(errors.TypeNil))
			},
		)

		It(
			"should return 500 Internal Server Error when the response encoder fails to encode the handler's result",
			func(ctx context.Context) {
				unaryServerFailingEncoder.BindHandler(
					func(_ context.Context, req testutil.Request) (testutil.Response, error) {
						return testutil.Response(req), nil
					},
				)
				body := MustSucceed(
					json.Codec.Encode(
						ctx,
						testutil.Request{ID: 9, Message: "encode-fail"},
					),
				)
				httpReq := MustSucceed(http.NewRequestWithContext(
					ctx,
					http.MethodPost,
					"http://"+unaryAddr.String()+"/encode-failure",
					bytes.NewReader(body),
				))
				httpReq.Header.Set(fiber.HeaderContentType, "application/json")
				httpReq.Header.Set(fiber.HeaderAccept, "application/x-fail")
				httpRes := MustSucceed((&http.Client{}).Do(httpReq))
				DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
				Expect(httpRes.StatusCode).To(Equal(http.StatusInternalServerError))
				respBody := MustSucceed(io.ReadAll(httpRes.Body))
				Expect(
					string(respBody),
				).To(ContainSubstring(errFailingEncoderEncodeFail.Error()))
			},
		)

		It(
			"should encode handler errors with the default encoder the request accepts",
			func(ctx context.Context) {
				unaryServerFailingEncoder.BindHandler(
					func(_ context.Context, _ testutil.Request) (testutil.Response, error) {
						return testutil.Response{}, testutil.ErrCustom
					},
				)
				body := MustSucceed(
					json.Codec.Encode(ctx, testutil.Request{ID: 11, Message: "err"}),
				)
				httpReq := MustSucceed(http.NewRequestWithContext(
					ctx,
					http.MethodPost,
					"http://"+unaryAddr.String()+"/encode-failure",
					bytes.NewReader(body),
				))
				httpReq.Header.Set(fiber.HeaderContentType, "application/json")
				httpReq.Header.Set(
					fiber.HeaderAccept,
					"application/x-fail, application/msgpack",
				)
				httpRes := MustSucceed((&http.Client{}).Do(httpReq))
				DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
				Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentType),
				).To(Equal("application/msgpack"))
				var pld errors.Payload
				respBody := MustSucceed(io.ReadAll(httpRes.Body))
				Expect(msgpack.Codec.Decode(ctx, respBody, &pld)).To(Succeed())
				Expect(errors.Decode(ctx, pld)).To(MatchError(testutil.ErrCustom))
			},
		)

		It(
			"should encode handler errors as JSON when the response encoder carries no error payload",
			func(ctx context.Context) {
				unaryServerFailingEncoder.BindHandler(
					func(_ context.Context, _ testutil.Request) (testutil.Response, error) {
						return testutil.Response{}, testutil.ErrCustom
					},
				)
				body := MustSucceed(
					json.Codec.Encode(ctx, testutil.Request{ID: 10, Message: "err"}),
				)
				httpReq := MustSucceed(http.NewRequestWithContext(
					ctx,
					http.MethodPost,
					"http://"+unaryAddr.String()+"/encode-failure",
					bytes.NewReader(body),
				))
				httpReq.Header.Set(fiber.HeaderContentType, "application/json")
				httpReq.Header.Set(fiber.HeaderAccept, "application/x-fail")
				httpRes := MustSucceed((&http.Client{}).Do(httpReq))
				DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
				Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentType),
				).To(Equal("application/json"))
				var pld errors.Payload
				respBody := MustSucceed(io.ReadAll(httpRes.Body))
				Expect(json.Codec.Decode(ctx, respBody, &pld)).To(Succeed())
				Expect(errors.Decode(ctx, pld)).To(MatchError(testutil.ErrCustom))
			},
		)
	})

	Describe("Query Params", func() {
		// Binds a handler that echoes the named request params back in the response
		// message, joined by "|", so the test can observe what parseRequestCtx put in
		// freighter.Context.Params.
		bindParamEcho := func(keys ...string) {
			unaryServer.BindHandler(
				func(ctx context.Context, _ testutil.Request) (testutil.Response, error) {
					params := freighter.MDFromContext(ctx).Params
					values := make([]string, len(keys))
					for i, k := range keys {
						if v, ok := params.Get(k); ok {
							values[i], _ = v.(string)
						}
					}
					return testutil.Response{Message: strings.Join(values, "|")}, nil
				},
			)
		}
		post := func(ctx context.Context, query string) testutil.Response {
			GinkgoHelper()
			body := MustSucceed(json.Codec.Encode(ctx, testutil.Request{}))
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx, http.MethodPost,
				"http://"+unaryAddr.String()+"/?"+query,
				bytes.NewReader(body),
			))
			httpReq.Header.Set(fiber.HeaderContentType, "application/json")
			httpRes := MustSucceed((&http.Client{}).Do(httpReq))
			DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
			respBody := MustSucceed(io.ReadAll(httpRes.Body))
			var res testutil.Response
			Expect(json.Codec.Decode(ctx, respBody, &res)).To(Succeed())
			return res
		}

		It("should expose the Cookie header as a param", func(ctx context.Context) {
			bindParamEcho(fiber.HeaderCookie)
			body := MustSucceed(json.Codec.Encode(ctx, testutil.Request{}))
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx, http.MethodPost,
				"http://"+unaryAddr.String()+"/",
				bytes.NewReader(body),
			))
			httpReq.Header.Set(fiber.HeaderContentType, "application/json")
			httpReq.Header.Set(fiber.HeaderCookie, "session=abc; theme=dark")
			httpRes := MustSucceed((&http.Client{}).Do(httpReq))
			DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
			respBody := MustSucceed(io.ReadAll(httpRes.Body))
			var res testutil.Response
			Expect(json.Codec.Decode(ctx, respBody, &res)).To(Succeed())
			Expect(res.Message).To(Equal("session=abc; theme=dark"))
		})

		It(
			"should expose freighterctx-prefixed query params with the prefix stripped",
			func(ctx context.Context) {
				bindParamEcho("file_name", "project")
				res := post(
					ctx,
					"freighterctxfile_name=Metrics%20Log.json&freighterctxproject=project:abc",
				)
				Expect(res.Message).To(Equal("Metrics Log.json|project:abc"))
			},
		)

		It(
			"should not expose unprefixed query params to the handler",
			func(ctx context.Context) {
				bindParamEcho("file_name", "project")
				res := post(ctx, "file_name=Metrics%20Log.json&project=project:abc")
				Expect(res.Message).To(Equal("|"))
			},
		)
	})

	Describe("Codec Configuration", func() {
		It(
			"should restrict the request decoders to the codecs passed via WithRequestDecoders",
			func(ctx context.Context) {
				unaryServerJSONOnly.BindHandler(
					func(_ context.Context, req testutil.Request) (testutil.Response, error) {
						return testutil.Response(req), nil
					},
				)
				req := testutil.Request{ID: 1, Message: "json-only"}
				httpReq := MustSucceed(http.NewRequestWithContext(
					ctx, http.MethodPost, "http://"+unaryAddr.String()+"/json-only",
					bytes.NewReader(MustSucceed(msgpack.Codec.Encode(ctx, req))),
				))
				httpReq.Header.Set(fiber.HeaderContentType, "application/msgpack")
				httpReq.Header.Set(fiber.HeaderAccept, "application/json")
				httpRes := MustSucceed((&http.Client{}).Do(httpReq))
				DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
				Expect(httpRes.StatusCode).To(Equal(http.StatusUnsupportedMediaType))
			},
		)

		It(
			"should restrict the response encoders to the codecs passed via WithResponseEncoders",
			func(ctx context.Context) {
				unaryServerMsgpackOnly.BindHandler(
					func(_ context.Context, req testutil.Request) (testutil.Response, error) {
						return testutil.Response(req), nil
					},
				)
				req := testutil.Request{ID: 2, Message: "msgpack-only"}
				httpReq := MustSucceed(http.NewRequestWithContext(
					ctx, http.MethodPost, "http://"+unaryAddr.String()+"/msgpack-only",
					bytes.NewReader(MustSucceed(msgpack.Codec.Encode(ctx, req))),
				))
				httpReq.Header.Set(fiber.HeaderContentType, "application/msgpack")
				httpReq.Header.Set(fiber.HeaderAccept, "application/json")
				httpRes := MustSucceed((&http.Client{}).Do(httpReq))
				DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
				Expect(httpRes.StatusCode).To(Equal(http.StatusNotAcceptable))
			},
		)

		It(
			"should round-trip end-to-end when the client and server agree on the restricted codec",
			func(ctx context.Context) {
				unaryServerMsgpackOnly.BindHandler(
					func(_ context.Context, req testutil.Request) (testutil.Response, error) {
						return testutil.Response(req), nil
					},
				)
				req := testutil.Request{ID: 3, Message: "round-trip"}
				httpReq := MustSucceed(http.NewRequestWithContext(
					ctx, http.MethodPost, "http://"+unaryAddr.String()+"/msgpack-only",
					bytes.NewReader(MustSucceed(msgpack.Codec.Encode(ctx, req))),
				))
				httpReq.Header.Set(fiber.HeaderContentType, "application/msgpack")
				httpReq.Header.Set(fiber.HeaderAccept, "application/msgpack")
				httpRes := MustSucceed((&http.Client{}).Do(httpReq))
				DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
				Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentType),
				).To(Equal("application/msgpack"))
				respBody := MustSucceed(io.ReadAll(httpRes.Body))
				var got testutil.Response
				Expect(msgpack.Codec.Decode(ctx, respBody, &got)).To(Succeed())
				Expect(got).To(Equal(testutil.Response(req)))
			},
		)
	})

	Describe("Streaming Response", func() {
		const (
			repeat   = "application/x-repeat"
			echo     = "application/x-echo"
			truncate = "application/x-truncate"
		)
		// streamingPost sends req to the streaming route, asking for the given content
		// encoding, and returns the response.
		streamingPost := func(
			ctx context.Context,
			req testutil.Request,
			accept string,
			acceptEncoding string,
		) *http.Response {
			GinkgoHelper()
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx, http.MethodPost, "http://"+unaryAddr.String()+"/streaming",
				bytes.NewReader(MustSucceed(json.Codec.Encode(ctx, req))),
			))
			httpReq.Header.Set(fiber.HeaderContentType, "application/json")
			httpReq.Header.Set(fiber.HeaderAccept, accept)
			if acceptEncoding != "" {
				httpReq.Header.Set(fiber.HeaderAcceptEncoding, acceptEncoding)
			}
			res := MustSucceed((&http.Client{
				Transport: &http.Transport{DisableCompression: true},
			}).Do(httpReq))
			DeferCleanup(func() { Expect(res.Body.Close()).To(Succeed()) })
			return res
		}

		It("should write the encoder's output as a chunked body", func(
			ctx context.Context,
		) {
			res := streamingPost(
				ctx, testutil.Request{ID: 3, Message: "ab"}, repeat, "identity",
			)
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(res.TransferEncoding).To(ConsistOf("chunked"))
			Expect(res.Header.Get(fiber.HeaderContentEncoding)).To(BeEmpty())
			Expect(io.ReadAll(res.Body)).To(Equal([]byte("ababab")))
		})

		DescribeTable(
			"should gzip a compressed encoder's body only when the request accepts gzip",
			func(ctx context.Context, acceptEncoding, contentEncoding string) {
				res := streamingPost(
					ctx, testutil.Request{ID: 3, Message: "ab"}, repeat, acceptEncoding,
				)
				Expect(res.StatusCode).To(Equal(http.StatusOK))
				Expect(res.Header.Get(fiber.HeaderContentEncoding)).
					To(Equal(contentEncoding))
				var body io.Reader = res.Body
				if contentEncoding == "gzip" {
					r := MustSucceed(gzip.NewReader(res.Body))
					DeferCleanup(func() { Expect(r.Close()).To(Succeed()) })
					body = r
				}
				Expect(io.ReadAll(body)).To(Equal([]byte("ababab")))
			},
			Entry("when it names gzip", "gzip", "gzip"),
			Entry("when it weighs gzip below another coding", "br, gzip;q=0.5", "gzip"),
			Entry("when it accepts any coding", "*", "gzip"),
			Entry("unless it gives gzip a zero weight", "gzip;q=0, identity", ""),
			Entry("unless it names only other codings", "br, deflate", ""),
			Entry("unless it has no Accept-Encoding", "", ""),
		)

		DescribeTable(
			"should send an empty body when the encoder writes nothing",
			func(ctx context.Context, acceptEncoding, contentEncoding string) {
				res := streamingPost(
					ctx, testutil.Request{ID: 0, Message: "ab"}, repeat, acceptEncoding,
				)
				Expect(res.StatusCode).To(Equal(http.StatusOK))
				Expect(res.Header.Get(fiber.HeaderContentEncoding)).
					To(Equal(contentEncoding))
				var body io.Reader = res.Body
				if contentEncoding == "gzip" {
					r := MustSucceed(gzip.NewReader(res.Body))
					DeferCleanup(func() { Expect(r.Close()).To(Succeed()) })
					body = r
				}
				Expect(io.ReadAll(body)).To(BeEmpty())
			},
			Entry("in an uncompressed body", "identity", ""),
			Entry("in a gzipped body", "gzip", "gzip"),
		)

		It("should leave an encoder the route did not name uncompressed", func(
			ctx context.Context,
		) {
			res := streamingPost(
				ctx,
				testutil.Request{ID: 3, Message: "ab"},
				echo,
				"gzip",
			)
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(res.TransferEncoding).To(ConsistOf("chunked"))
			Expect(res.Header.Get(fiber.HeaderContentEncoding)).To(BeEmpty())
			Expect(io.ReadAll(res.Body)).To(Equal([]byte("ab")))
		})

		DescribeTable(
			"should answer with a 400 when the encoder fails before the body leaves",
			func(ctx context.Context, writes int, acceptEncoding string) {
				res := streamingPost(
					ctx,
					testutil.Request{ID: writes, Message: "ab"},
					truncate,
					acceptEncoding,
				)
				Expect(res.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(res.Header.Get(fiber.HeaderContentEncoding)).To(BeEmpty())
				Expect(io.ReadAll(res.Body)).
					To(ContainSubstring(errTruncatingEncoderFail.Error()))
			},
			Entry("before it writes", 0, "identity"),
			Entry("before it writes a gzipped body", 0, "gzip"),
			Entry("after it buffers part of the body", 3, "identity"),
			Entry("after it buffers part of a gzipped body", 3, "gzip"),
		)

		It("should cut the body short when the encoder fails after part of it leaves",
			func(ctx context.Context) {
				res := streamingPost(
					ctx,
					testutil.Request{ID: 8192, Message: "ab"},
					truncate,
					"identity",
				)
				Expect(res.StatusCode).To(Equal(http.StatusOK))
				body, err := io.ReadAll(res.Body)
				Expect(err).To(MatchError(io.ErrUnexpectedEOF))
				Expect(len(body)).To(BeNumerically("<", 2*8192))
			},
		)
	})

	Describe("Unencoded Response", func() {
		// closingPost sends a request to the closing route and returns the response
		// along with the value the handler produced.
		closingPost := func(
			ctx context.Context,
			failAfterHandler bool,
		) (*http.Response, *closingResponse) {
			GinkgoHelper()
			httpReq := MustSucceed(http.NewRequestWithContext(
				ctx,
				http.MethodPost,
				"http://"+unaryAddr.String()+"/closing",
				bytes.NewReader(
					MustSucceed(json.Codec.Encode(ctx, testutil.Request{})),
				),
			))
			httpReq.Header.Set(fiber.HeaderContentType, "application/json")
			if failAfterHandler {
				httpReq.Header.Set(failAfterHandlerHeader, "true")
			}
			res := MustSucceed(http.DefaultClient.Do(httpReq))
			DeferCleanup(func() { Expect(res.Body.Close()).To(Succeed()) })
			var value *closingResponse
			Eventually(closingResponses).Should(Receive(&value))
			return res, value
		}

		It("should close a response that a failed middleware keeps from the encoder",
			func(ctx context.Context) {
				res, value := closingPost(ctx, true)
				Expect(res.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(value.closed.Load()).To(BeTrue())
			},
		)

		It("should leave a response it encodes open", func(ctx context.Context) {
			res, value := closingPost(ctx, false)
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(io.ReadAll(res.Body)).To(Equal([]byte("ok")))
			Expect(value.closed.Load()).To(BeFalse())
		})
	})

	Describe("Report", func() {
		It(
			"should report the unary server's protocol and accepted/emitted content types",
			func() {
				report := unaryServer.Report()
				Expect(report["protocol"]).To(Equal("http"))
				Expect(report["acceptedContentTypes"]).To(Equal([]string{
					"application/json", "application/msgpack",
				}))
				Expect(report["emittedContentTypes"]).To(Equal([]string{
					"application/json", "application/msgpack",
				}))
			},
		)

		It(
			"should reflect WithRequestDecoders and WithResponseEncoders in the server report",
			func() {
				report := unaryServerJSONOnly.Report()
				Expect(
					report["acceptedContentTypes"],
				).To(Equal([]string{"application/json"}))
				Expect(
					report["emittedContentTypes"],
				).To(Equal([]string{"application/json"}))
			},
		)

		It(
			"should report the unary client's protocol, sent Content-Type, and accepted Content-Types",
			func() {
				report := unaryClient.Report()
				Expect(report["protocol"]).To(Equal("http"))
				Expect(report["sentContentType"]).To(Equal("application/json"))
				Expect(report["acceptedContentTypes"]).To(Equal([]string{
					"application/json", "application/msgpack",
				}))
			},
		)

		It("should reflect a custom Encoder in the client report", func() {
			c := MustSucceed(fhttp.NewUnaryClient[testutil.Request, testutil.Response](
				fhttp.UnaryClientConfig{
					Encoder:  msgpack.Codec,
					Decoders: []xhttp.Decoder{msgpack.Codec},
				},
			))
			report := c.Report()
			Expect(report["sentContentType"]).To(Equal("application/msgpack"))
			Expect(
				report["acceptedContentTypes"],
			).To(Equal([]string{"application/msgpack"}))
		})
	})

	Describe("Client Decoder Resolution", func() {
		It(
			"should fail with an unresolved-decoder error when the server response Content-Type has no registered client decoder",
			func(ctx context.Context) {
				_, err := unaryClient.Send(
					ctx,
					unaryAddr+"/text-plain",
					testutil.Request{ID: 1, Message: "x"},
				)
				Expect(err).To(MatchError(ContainSubstring("text/plain")))
			},
		)
	})
})
