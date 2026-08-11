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

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/encoding/json"
	xhttp "github.com/synnaxlabs/x/http"
	. "github.com/synnaxlabs/x/testutil"
)

// largeMessage is comfortably above the default 1KB compression floor and repetitive
// enough that every encoding shrinks it by a wide margin.
var largeMessage = strings.Repeat("synnax telemetry payload ", 200)

var _ = Describe("Compression", func() {
	bindEcho := func() {
		GinkgoHelper()
		unaryServer.BindHandler(
			func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			},
		)
	}

	// post drives the server directly so the request's Content-Encoding and
	// Accept-Encoding can be set independently of any client's policy.
	post := func(
		ctx context.Context,
		body []byte,
		contentEncoding string,
		acceptEncoding string,
	) *http.Response {
		GinkgoHelper()
		httpReq := MustSucceed(http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			"http://"+unaryAddr.String()+"/",
			bytes.NewReader(body),
		))
		httpReq.Header.Set(fiber.HeaderContentType, "application/json")
		httpReq.Header.Set(fiber.HeaderAccept, "application/json")
		if contentEncoding != "" {
			httpReq.Header.Set(fiber.HeaderContentEncoding, contentEncoding)
		}
		// net/http adds its own Accept-Encoding and silently decompresses the
		// response unless the header is set here, which would hide what the server
		// actually sent.
		httpReq.Header.Set(fiber.HeaderAcceptEncoding, acceptEncoding)
		httpRes := MustSucceed((&http.Client{}).Do(httpReq))
		DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
		return httpRes
	}

	Describe("Requests", func() {
		DescribeTable("should decompress a compressed request body",
			func(ctx context.Context, c xhttp.Compression) {
				bindEcho()
				req := test.Request{ID: 1, Message: largeMessage}
				body := MustSucceed(
					c.Compress(MustSucceed(json.Codec.Encode(ctx, req))),
				)
				httpRes := post(ctx, body, c.ContentEncoding(), "")
				Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
				var got test.Response
				Expect(json.Codec.DecodeStream(ctx, httpRes.Body, &got)).To(Succeed())
				Expect(got).To(Equal(test.Response(req)))
			},
			Entry("gzip", xhttp.Gzip),
			Entry("brotli", xhttp.Brotli),
			Entry("zstd", xhttp.Zstd),
		)

		It("should accept an uncompressed body with no Content-Encoding",
			func(ctx context.Context) {
				bindEcho()
				req := test.Request{ID: 2, Message: "hello"}
				httpRes := post(
					ctx, MustSucceed(json.Codec.Encode(ctx, req)), "", "",
				)
				Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
			})

		It("should reject an unknown Content-Encoding", func(ctx context.Context) {
			bindEcho()
			req := test.Request{ID: 3, Message: "hello"}
			httpRes := post(
				ctx, MustSucceed(json.Codec.Encode(ctx, req)), "snappy", "",
			)
			Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(io.ReadAll(httpRes.Body)).To(ContainSubstring("snappy"))
		})

		It("should reject a body that lies about its encoding",
			func(ctx context.Context) {
				bindEcho()
				httpRes := post(ctx, []byte("not gzip at all"), "gzip", "")
				Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
			})
	})

	Describe("Responses", func() {
		DescribeTable("should compress a response the client accepts",
			func(ctx context.Context, acceptEncoding, expected string) {
				bindEcho()
				req := test.Request{ID: 4, Message: largeMessage}
				body := MustSucceed(json.Codec.Encode(ctx, req))
				httpRes := post(ctx, body, "", acceptEncoding)
				Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
				Expect(
					httpRes.Header.Get(fiber.HeaderContentEncoding),
				).To(Equal(expected))
				Expect(
					httpRes.Header.Get(fiber.HeaderVary),
				).To(Equal(fiber.HeaderAcceptEncoding))
				compression := MustSucceed(xhttp.ResolveCompression(
					expected, []xhttp.Compression{xhttp.Zstd, xhttp.Brotli, xhttp.Gzip},
				))
				raw := MustSucceed(io.ReadAll(httpRes.Body))
				Expect(len(raw)).To(BeNumerically("<", len(body)))
				var got test.Response
				Expect(json.Codec.Decode(
					ctx, MustSucceed(compression.Decompress(raw, 0)), &got,
				)).To(Succeed())
				Expect(got).To(Equal(test.Response(req)))
			},
			Entry("prefers zstd", "gzip, br, zstd", "zstd"),
			Entry("falls back to brotli", "gzip, br", "br"),
			Entry("falls back to gzip", "gzip", "gzip"),
			Entry("honors quality values", "zstd;q=0.1, gzip;q=0.9", "gzip"),
		)

		DescribeTable("should send the body uncompressed",
			func(ctx context.Context, message, acceptEncoding string) {
				bindEcho()
				req := test.Request{ID: 5, Message: message}
				httpRes := post(
					ctx, MustSucceed(json.Codec.Encode(ctx, req)), "", acceptEncoding,
				)
				Expect(httpRes.StatusCode).To(Equal(http.StatusOK))
				Expect(httpRes.Header.Get(fiber.HeaderContentEncoding)).To(BeEmpty())
				var got test.Response
				Expect(json.Codec.DecodeStream(ctx, httpRes.Body, &got)).To(Succeed())
				Expect(got).To(Equal(test.Response(req)))
			},
			Entry("when the client does not advertise", largeMessage, ""),
			Entry("when the client accepts only identity", largeMessage, "identity"),
			Entry("when the client rejects every offer", largeMessage,
				"gzip;q=0, br;q=0, zstd;q=0"),
			Entry("when the body is below the size floor", "hello", "gzip, br, zstd"),
		)

		It("should leave a small error payload uncompressed",
			func(ctx context.Context) {
				unaryServer.BindHandler(
					func(_ context.Context, _ test.Request) (test.Response, error) {
						return test.Response{}, test.ErrCustom
					},
				)
				req := test.Request{ID: 6, Message: largeMessage}
				httpRes := post(
					ctx, MustSucceed(json.Codec.Encode(ctx, req)), "", "gzip",
				)
				Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(httpRes.Header.Get(fiber.HeaderContentEncoding)).To(BeEmpty())
			})

		It("should keep a body that grows under compression uncompressed",
			func(ctx context.Context) {
				// Incompressible bytes make every encoding larger, so the server has
				// to notice and fall back to the original body.
				unaryServer.BindHandler(
					func(_ context.Context, _ test.Request) (test.Response, error) {
						return test.Response{ID: 1, Message: incompressible(4096)}, nil
					},
				)
				req := test.Request{ID: 1, Message: "hello"}
				httpRes := post(
					ctx, MustSucceed(json.Codec.Encode(ctx, req)), "", "gzip",
				)
				Expect(httpRes.Header.Get(fiber.HeaderContentEncoding)).To(BeEmpty())
				var got test.Response
				Expect(json.Codec.DecodeStream(ctx, httpRes.Body, &got)).To(Succeed())
				Expect(got.Message).To(HaveLen(4096))
			})
	})

	Describe("Client", func() {
		It("should round trip a large payload through both directions",
			func(ctx context.Context) {
				bindEcho()
				req := test.Request{ID: 7, Message: largeMessage}
				Expect(unaryClient.Send(ctx, unaryAddr, req)).
					To(Equal(test.Response(req)))
			})

		It("should round trip with compression disabled", func(ctx context.Context) {
			bindEcho()
			client := MustSucceed(
				fhttp.NewUnaryClient[test.Request, test.Response](
					fhttp.UnaryClientConfig{DisableCompression: true},
				),
			)
			req := test.Request{ID: 8, Message: largeMessage}
			Expect(client.Send(ctx, unaryAddr, req)).To(Equal(test.Response(req)))
		})

		DescribeTable("should reject an invalid config",
			func(cfg fhttp.UnaryClientConfig, field string) {
				Expect(fhttp.NewUnaryClient[test.Request, test.Response](cfg)).
					Error().
					To(MatchError(ContainSubstring(field)))
			},
			Entry(
				"negative min compress size",
				fhttp.UnaryClientConfig{MinCompressSize: -1},
				"min_compress_size",
			),
			Entry(
				"negative max decompressed size",
				fhttp.UnaryClientConfig{MaxDecompressedSize: -1},
				"max_decompressed_size",
			),
		)

		It("should report the content encodings it offers", func() {
			Expect(unaryClient.Report()["contentEncodings"]).
				To(Equal([]string{"zstd", "br", "gzip", "deflate"}))
		})

		It("should talk to a server that offers only gzip", func(ctx context.Context) {
			bindEcho()
			client := MustSucceed(fhttp.NewUnaryClient[test.Request, test.Response](
				fhttp.UnaryClientConfig{
					Compressions: []xhttp.Compression{xhttp.Gzip},
				},
			))
			req := test.Request{ID: 10, Message: largeMessage}
			Expect(client.Send(ctx, unaryAddr, req)).To(Equal(test.Response(req)))
		})
	})

	Describe("Server Options", func() {
		It("should offer no encodings when compression is disabled", func() {
			router := MustSucceed(fhttp.NewRouter())
			server := fhttp.NewUnaryServer[test.Request, test.Response](
				router, "/no-compression", fhttp.WithCompressions(),
			)
			Expect(server.Report()["contentEncodings"]).To(BeEmpty())
		})

		It("should reject a request body that expands past the maximum",
			func(ctx context.Context) {
				var (
					router = MustSucceed(fhttp.NewRouter())
					app    = newFiberApp(fiber.Config{DisableKeepalive: true})
					server = fhttp.NewUnaryServer[test.Request, test.Response](
						router, "/", fhttp.WithMaxDecompressedSize(16),
					)
				)
				server.BindHandler(
					func(_ context.Context, req test.Request) (test.Response, error) {
						return test.Response(req), nil
					},
				)
				router.BindTo(app)
				req := test.Request{ID: 9, Message: largeMessage}
				body := MustSucceed(
					xhttp.Gzip.Compress(MustSucceed(json.Codec.Encode(ctx, req))),
				)
				httpReq := MustSucceed(http.NewRequest(
					http.MethodPost, "http://test/", bytes.NewReader(body),
				))
				httpReq.Header.Set(fiber.HeaderContentType, "application/json")
				httpReq.Header.Set(fiber.HeaderContentEncoding, "gzip")
				httpRes := MustSucceed(app.Test(httpReq))
				DeferCleanup(func() { Expect(httpRes.Body.Close()).To(Succeed()) })
				Expect(httpRes.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(io.ReadAll(httpRes.Body)).
					To(ContainSubstring("exceeds the maximum size"))
			})
	})
})

// incompressible returns a string of n bytes drawn from a wide alphabet in a
// non-repeating order, which every encoding grows rather than shrinks.
func incompressible(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var b strings.Builder
	b.Grow(n)
	// A linear congruential walk over the alphabet keeps this deterministic while
	// leaving no repeated substring long enough for a dictionary to exploit.
	state := 1
	for range n {
		state = (state*1103515245 + 12345) & 0x7fffffff
		b.WriteByte(alphabet[state%len(alphabet)])
	}
	return b.String()
}
