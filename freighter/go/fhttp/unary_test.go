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

var _ UnaryImplementation = &implementation{}

func (i *implementation) Start(host address.Address) (UnaryServer, UnaryClient) {
	i.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	router := MustSucceed(fhttp.NewRouter())
	server := fhttp.NewUnaryServer[Request, Response](router, "/")
	clientCfg := fhttp.ClientConfig{Codec: binary.JSONCodec}
	client := MustSucceed(fhttp.NewUnaryClient[Request, Response](clientCfg))
	router.BindTo(i.app)
	go func() {
		defer GinkgoRecover()
		Expect(i.app.Listen(host.PortString())).To(Succeed())
	}()
	return server, client
}

func (i *implementation) Stop() error { return i.app.Shutdown() }

type TestRequest struct {
	ID      int    `json:"id" msgpack:"id"`
	Message string `json:"message" msgpack:"message"`
}

var _ = Describe("Unary", func() {
	AssertUnary(&implementation{})
	Describe("Server", func() {
		It("Should return an error if the content type is not supported", func() {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			router := MustSucceed(fhttp.NewRouter())
			fhttp.NewUnaryServer[any, any](router, "/")
			router.BindTo(app)
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("Content-Type", "text/plain")
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusUnsupportedMediaType))
		})
		It("Should return an error if decoding the request body fails", func() {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			router := MustSucceed(fhttp.NewRouter())
			fhttp.NewUnaryServer[any, any](router, "/")
			router.BindTo(app)
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("Content-Type", "application/msgpack")
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusBadRequest))
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(string(body)).To(ContainSubstring(binary.ErrDecode.Error()))
		})
		It("should return an error if the requested content-type is not acceptable", func() {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			router := MustSucceed(fhttp.NewRouter())
			fhttp.NewUnaryServer[any, any](router, "/")
			router.BindTo(app)
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("Content-Type", "application/msgpack")
			req.Header.Set("Accept", "application/nonexistent")
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusNotAcceptable))
		})
		It("Should allow for different content types", func() {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			router := MustSucceed(fhttp.NewRouter())
			server := fhttp.NewUnaryServer[TestRequest, TestRequest](router, "/")
			server.BindHandler(func(_ context.Context, req TestRequest) (TestRequest, error) {
				return req, nil
			})
			router.BindTo(app)
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/msgpack")
			testReq := TestRequest{ID: 1, Message: "Hello, World!"}
			jsonData := MustSucceed(json.Marshal(testReq))
			msgpackData := MustSucceed(msgpack.Marshal(testReq))
			req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
			req.ContentLength = int64(len(jsonData))
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusOK))
			Expect(res.Header.Get("Content-Type")).To(Equal("application/msgpack"))
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(body).To(Equal(msgpackData))
		})
		It("should allow for adding response content type resolvers", func() {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			router := MustSucceed(fhttp.NewRouter())
			server := fhttp.NewUnaryServer[TestRequest, TestRequest](router, "/", fhttp.WithResponseEncoders(map[string]binary.Encoder{
				"application/x-gob": binary.GobCodec,
			}))
			server.BindHandler(func(_ context.Context, req TestRequest) (TestRequest, error) {
				return req, nil
			})
			router.BindTo(app)
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/x-gob")
			testReq := TestRequest{ID: 1, Message: "Hello, World!"}
			jsonData := MustSucceed(json.Marshal(testReq))
			req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
			req.ContentLength = int64(len(jsonData))
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusOK))
			Expect(res.Header.Get("Content-Type")).To(Equal("application/x-gob"))
			var decodedRes TestRequest
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(binary.GobCodec.Decode(context.Background(), body, &decodedRes)).To(Succeed())
			Expect(decodedRes).To(Equal(testReq))
		})
		It("should allow sending direct io.Reader as response", func() {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			router := MustSucceed(fhttp.NewRouter())
			server := fhttp.NewUnaryServer[TestRequest, io.Reader](router, "/")
			server.BindHandler(func(_ context.Context, req TestRequest) (io.Reader, error) {
				return bytes.NewBufferString(req.Message), nil
			})
			router.BindTo(app)
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("Content-Type", "application/json")
			jsonData := MustSucceed(json.Marshal(TestRequest{ID: 1, Message: "Hello, World!"}))
			req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
			req.ContentLength = int64(len(jsonData))
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(fiber.StatusOK))
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(string(body)).To(Equal("Hello, World!"))
		})
	})
})
