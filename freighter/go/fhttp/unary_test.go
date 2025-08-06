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
	"encoding/json"
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
	"github.com/vmihailenco/msgpack/v5"
)

type implementation struct{ app *fiber.App }

var _ UnaryImplementation = (*implementation)(nil)

func (i *implementation) Start(host address.Address) (UnaryServer, UnaryClient) {
	i.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	router := MustSucceed(fhttp.NewRouter())
	server := fhttp.NewUnaryServer[Request, Response](router, "/")
	client := MustSucceed(fhttp.NewUnaryClient[Request, Response]())
	router.BindTo(i.app)
	go func() {
		defer GinkgoRecover()
		Expect(i.app.Listen(host.PortString())).To(Succeed())
	}()
	return server, client
}

func (i *implementation) Stop() error { return i.app.Shutdown() }

var _ = Describe("Unary", func() {
	AssertUnary(&implementation{})
	Describe("Server", func() {
		var (
			app    *fiber.App
			router *fhttp.Router
			req    *http.Request
		)
		BeforeEach(func() {
			app = fiber.New(fiber.Config{DisableStartupMessage: true})
			router = MustSucceed(fhttp.NewRouter())
			req = httptest.NewRequest(fiber.MethodPost, "/", nil)
		})
		It("Should return an error if the content type is not supported", func() {
			fhttp.NewUnaryServer[any, any](router, "/")
			router.BindTo(app)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMETextPlain)
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusUnsupportedMediaType))
		})
		It("Should return an error if decoding the request body fails", func() {
			fhttp.NewUnaryServer[any, any](router, "/")
			router.BindTo(app)
			req.Header.Set(fiber.HeaderContentType, fhttp.MIMEApplicationMsgPack)
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusBadRequest))
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(string(body)).To(ContainSubstring(binary.ErrDecode.Error()))
		})
		It("should return an error if the requested content-type is not acceptable", func() {
			fhttp.NewUnaryServer[any, any](router, "/")
			router.BindTo(app)
			req.Header.Set(fiber.HeaderContentType, fhttp.MIMEApplicationMsgPack)
			req.Header.Set(fiber.HeaderAccept, "application/nonexistent")
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusNotAcceptable))
		})
		It("Should allow for different content types", func() {
			server := fhttp.NewUnaryServer[Request, Response](router, "/")
			server.BindHandler(func(_ context.Context, req Request) (Response, error) {
				return req, nil
			})
			router.BindTo(app)
			req.Header.Set(fiber.HeaderContentType, fhttp.MIMEApplicationJSON)
			req.Header.Set(fiber.HeaderAccept, fhttp.MIMEApplicationMsgPack)
			testReq := Request{ID: 1, Message: "Hello, World!"}
			jsonData := MustSucceed(json.Marshal(testReq))
			msgpackData := MustSucceed(msgpack.Marshal(testReq))
			req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
			req.ContentLength = int64(len(jsonData))
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusOK))
			Expect(res.Header.Get(fiber.HeaderContentType)).
				To(Equal(fhttp.MIMEApplicationMsgPack))
			Expect(io.ReadAll(res.Body)).To(Equal(msgpackData))
		})
		It("should allow for adding response content type resolvers", func() {
			so := fhttp.WithResponseEncoders(map[string]func() binary.Encoder{
				"application/x-gob": func() binary.Encoder { return binary.GobCodec },
			})
			server := fhttp.NewUnaryServer[Request, Response](router, "/", so)
			server.BindHandler(func(_ context.Context, req Request) (Response, error) {
				return req, nil
			})
			router.BindTo(app)
			req.Header.Set(fiber.HeaderContentType, fhttp.MIMEApplicationJSON)
			req.Header.Set(fiber.HeaderAccept, "application/x-gob")
			testReq := Request{ID: 1, Message: "Hello, World!"}
			jsonData := MustSucceed(json.Marshal(testReq))
			req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
			req.ContentLength = int64(len(jsonData))
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusOK))
			Expect(res.Header.Get(fiber.HeaderContentType)).
				To(Equal("application/x-gob"))
			var decodedRes Response
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(binary.GobCodec.Decode(context.Background(), body, &decodedRes)).
				To(Succeed())
			Expect(decodedRes).To(Equal(testReq))
		})
		It("should allow sending direct io.Reader as response", func() {
			server := fhttp.NewUnaryServer[Request, io.Reader](router, "/")
			server.BindHandler(func(_ context.Context, req Request) (io.Reader, error) {
				return bytes.NewBufferString(req.Message), nil
			})
			router.BindTo(app)
			req.Header.Set(fiber.HeaderContentType, fhttp.MIMEApplicationJSON)
			jsonData := MustSucceed(
				json.Marshal(Request{ID: 1, Message: "Hello, World!"}),
			)
			req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
			req.ContentLength = int64(len(jsonData))
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusOK))
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(string(body)).To(Equal("Hello, World!"))
		})
		It("should allow adding additional codecs for both request and response", func() {
			so := fhttp.WithAdditionalCodecs(map[string]func() binary.Codec{
				"application/x-gob": func() binary.Codec { return binary.GobCodec },
			})
			server := fhttp.NewUnaryServer[Request, Response](router, "/", so)
			server.BindHandler(func(_ context.Context, req Request) (Response, error) {
				return req, nil
			})
			router.BindTo(app)
			req.Header.Set(fiber.HeaderContentType, "application/x-gob")
			req.Header.Set(fiber.HeaderAccept, "application/x-gob")
			testReq := Request{ID: 1, Message: "Hello, World!"}
			gobData := MustSucceed(
				binary.GobCodec.Encode(context.Background(), testReq),
			)
			req.Body = io.NopCloser(bytes.NewBuffer(gobData))
			req.ContentLength = int64(len(gobData))
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusOK))
			Expect(res.Header.Get(fiber.HeaderContentType)).
				To(Equal("application/x-gob"))
			var decodedRes Response
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(binary.GobCodec.Decode(context.Background(), body, &decodedRes)).
				To(Succeed())
			Expect(decodedRes).To(Equal(testReq))
		})
	})
})
