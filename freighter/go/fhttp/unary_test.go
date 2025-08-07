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
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gofiber/fiber/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/fhttp"
	. "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/binary"
	. "github.com/synnaxlabs/x/testutil"
)

type implementation struct{ *fiber.App }

var _ UnaryImplementation = (*implementation)(nil)

func (i *implementation) Start(host address.Address) (UnaryServer, UnaryClient) {
	i.App = fiber.New(fiber.Config{DisableStartupMessage: true})
	router := MustSucceed(fhttp.NewRouter())
	server := fhttp.NewUnaryServer[Request, Response](router, "/")
	client := MustSucceed(fhttp.NewUnaryClient[Request, Response]())
	router.BindTo(i.App)
	go func() {
		defer GinkgoRecover()
		Expect(i.Listen(host.PortString())).To(Succeed())
	}()
	return server, client
}

func (i *implementation) Stop() error { return i.Shutdown() }

var _ = Describe("Unary", func() {
	AssertUnary(&implementation{})
	Describe("Client", func() {
		Describe("Config", func() {
			Describe("Validate", func() {
				It("should succeed if the encoder and decoder are both non-nil", func() {
					Expect(fhttp.DefaultUnaryClientConfig.Validate()).To(Succeed())
				})
				It("should return an error if the encoder is nil", func() {
					cfg := fhttp.UnaryClientConfig{
						Decoder: fhttp.JSONCodec,
						Encoder: nil,
					}
					Expect(cfg.Validate()).To(MatchError(ContainSubstring("encoder")))
				})
				It("should return an error if the decoder is nil", func() {
					cfg := fhttp.UnaryClientConfig{
						Decoder: nil,
						Encoder: fhttp.JSONCodec,
					}
					Expect(cfg.Validate()).To(MatchError(ContainSubstring("decoder")))
				})
			})
			Describe("Override", func() {
				var originalCfg fhttp.UnaryClientConfig
				BeforeEach(func() {
					originalCfg = fhttp.UnaryClientConfig{
						Decoder: fhttp.JSONCodec,
						Encoder: fhttp.JSONCodec,
					}
				})
				It("should override the encoder and decoder if they are non-nil", func() {
					overrideCfg := fhttp.UnaryClientConfig{
						Decoder: fhttp.MsgPackCodec,
						Encoder: fhttp.MsgPackCodec,
					}
					cfg := originalCfg.Override(overrideCfg)
					Expect(cfg).To(Equal(overrideCfg))
				})
				It("shouldn't override if the encoder or decoder is nil", func() {
					overrideCfg := fhttp.UnaryClientConfig{}
					cfg := originalCfg.Override(overrideCfg)
					Expect(cfg).To(Equal(originalCfg))
				})
			})
		})
	})
	Describe("Server", func() {
		var (
			app    *fiber.App
			ctx    context.Context
			router *fhttp.Router
			req    *http.Request
		)
		BeforeEach(func() {
			app = fiber.New(fiber.Config{DisableStartupMessage: true})
			ctx = context.Background()
			router = MustSucceed(fhttp.NewRouter())
			req = httptest.NewRequest(fiber.MethodPost, "/", nil)
		})
		Describe("Normal Operation", func() {
			var (
				testReq  Request
				jsonData []byte
			)
			BeforeEach(func() {
				testReq = Request{ID: 1, Message: "Hello, World!"}
				jsonData = MustSucceed(binary.JSONCodec.Encode(ctx, testReq))
			})
			It("Should allow for different content types", func() {
				server := fhttp.NewUnaryServer[Request, Response](router, "/")
				server.BindHandler(func(_ context.Context, req Request) (Response, error) {
					return req, nil
				})
				router.BindTo(app)
				req.Header.Set(fiber.HeaderContentType, fhttp.MIMEApplicationJSON)
				req.Header.Set(fiber.HeaderAccept, fhttp.MIMEApplicationMsgPack)
				msgpackData := MustSucceed(binary.MsgPackCodec.Encode(ctx, testReq))
				req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
				req.ContentLength = int64(len(jsonData))
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(fiber.StatusOK))
				Expect(res.Header.Get(fiber.HeaderContentType)).
					To(Equal(fhttp.MIMEApplicationMsgPack))
				Expect(io.ReadAll(res.Body)).To(Equal(msgpackData))
			})
			It("should allow for adding response content type resolvers", func() {
				so := fhttp.WithResponseEncoders(map[string]binary.Encoder{
					"application/x-gob": binary.GobCodec,
				})
				server := fhttp.NewUnaryServer[Request, Response](router, "/", so)
				server.BindHandler(func(_ context.Context, req Request) (Response, error) {
					return req, nil
				})
				router.BindTo(app)
				req.Header.Set(fiber.HeaderContentType, fhttp.MIMEApplicationJSON)
				req.Header.Set(fiber.HeaderAccept, "application/x-gob")
				req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
				req.ContentLength = int64(len(jsonData))
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(fiber.StatusOK))
				Expect(res.Header.Get(fiber.HeaderContentType)).
					To(Equal("application/x-gob"))
				var decodedRes Response
				body := MustSucceed(io.ReadAll(res.Body))
				Expect(binary.GobCodec.Decode(ctx, body, &decodedRes)).To(Succeed())
				Expect(decodedRes).To(Equal(testReq))
			})
			It("should allow adding additional codecs for both request and response", func() {
				so := fhttp.WithResponseEncoders(map[string]binary.Encoder{
					"application/x-gob": binary.GobCodec,
				})
				server := fhttp.NewUnaryServer[Request, Response](router, "/", so)
				server.BindHandler(func(_ context.Context, req Request) (Response, error) {
					return req, nil
				})
				router.BindTo(app)
				req.Header.Set(fiber.HeaderContentType, "application/x-gob")
				req.Header.Set(fiber.HeaderAccept, "application/x-gob")
				testReq := Request{ID: 1, Message: "Hello, World!"}
				gobData := MustSucceed(binary.GobCodec.Encode(ctx, testReq))
				req.Body = io.NopCloser(bytes.NewBuffer(gobData))
				req.ContentLength = int64(len(gobData))
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(fiber.StatusOK))
				Expect(res.Header.Get(fiber.HeaderContentType)).
					To(Equal("application/x-gob"))
				var decodedRes Response
				body := MustSucceed(io.ReadAll(res.Body))
				Expect(binary.GobCodec.Decode(ctx, body, &decodedRes)).To(Succeed())
				Expect(decodedRes).To(Equal(testReq))
			})
		})
		Describe("Error Handling", func() {
			BeforeEach(func() {
				fhttp.NewUnaryServer[any, any](router, "/")
				router.BindTo(app)
			})
			It("Should return an error if the content type is not supported", func() {
				req.Header.Set(fiber.HeaderContentType, fiber.MIMETextPlain)
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(fiber.StatusUnsupportedMediaType))
			})
			It("Should return an error if decoding the request body fails", func() {
				req.Header.Set(fiber.HeaderContentType, fhttp.MIMEApplicationMsgPack)
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(fiber.StatusBadRequest))
				body := MustSucceed(io.ReadAll(res.Body))
				Expect(string(body)).To(ContainSubstring(binary.ErrDecode.Error()))
			})
			It("should return an error if the requested content type is not acceptable", func() {
				req.Header.Set(fiber.HeaderContentType, fhttp.MIMEApplicationMsgPack)
				req.Header.Set(fiber.HeaderAccept, fhttp.MIMETextPlain)
				res := MustSucceed(app.Test(req))
				Expect(res.StatusCode).To(Equal(fiber.StatusNotAcceptable))
			})
		})
	})
})
