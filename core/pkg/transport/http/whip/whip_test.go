// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package whip_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
	"uuid"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/recovery"
	apiwhip "github.com/synnaxlabs/synnax/pkg/api/whip"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/transport/http/whip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func send(
	app *fiber.App,
	method, path, token, contentType, body string,
) (*http.Response, string) {
	GinkgoHelper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set(fiber.HeaderContentType, contentType)
	}
	res := MustSucceed(app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second}))
	defer func() { Expect(res.Body.Close()).To(Succeed()) }()
	return res, string(MustSucceed(io.ReadAll(res.Body)))
}

var _ = Describe("Transport", func() {
	var (
		app         *fiber.App
		tr          *whip.Transport
		task        uuid.UUID
		session     uuid.UUID
		path        string
		publishes   []apiwhip.PublishRequest
		unpublishes []apiwhip.UnpublishRequest
		publishErr  error
	)
	publish := func() (*http.Response, string) {
		return send(app, http.MethodPost, path, "key", "application/sdp", "o")
	}
	BeforeEach(func() {
		tr = whip.New()
		app = fiber.New()
		tr.BindTo(app)
		task, session = uuid.New(), uuid.New()
		path = "/api/v1/whip/" + task.String()
		publishes, unpublishes, publishErr = nil, nil, nil
		tr.Publish().BindHandler(func(
			_ context.Context,
			req apiwhip.PublishRequest,
		) (apiwhip.PublishResponse, error) {
			publishes = append(publishes, req)
			res := apiwhip.PublishResponse{Session: session, Answer: "answer"}
			return res, publishErr
		})
		tr.Unpublish().BindHandler(func(
			_ context.Context,
			req apiwhip.UnpublishRequest,
		) (struct{}, error) {
			unpublishes = append(unpublishes, req)
			return struct{}{}, nil
		})
	})

	Describe("Publish", func() {
		It("Should answer with the SDP answer and the URL of the session", func() {
			res, body := publish()
			Expect(res.StatusCode).To(Equal(http.StatusCreated))
			Expect(res.Header.Get(fiber.HeaderContentType)).To(Equal("application/sdp"))
			Expect(res.Header.Get(fiber.HeaderLocation)).
				To(Equal(path + "/" + session.String()))
			Expect(body).To(Equal("answer"))
			Expect(publishes).To(Equal([]apiwhip.PublishRequest{
				{Task: task, Key: "key", Offer: "o"},
			}))
		})

		It("Should reject a request with no bearer token", func() {
			res, body := send(app, http.MethodPost, path, "", "application/sdp", "o")
			Expect(res.StatusCode).To(Equal(http.StatusUnauthorized))
			Expect(body).
				To(HavePrefix("the request needs an API key as its bearer token"))
			Expect(publishes).To(BeEmpty())
		})

		It("Should reject a body that is not SDP", func() {
			res, body := send(
				app,
				http.MethodPost,
				path,
				"key",
				"application/json",
				"{}",
			)
			Expect(res.StatusCode).To(Equal(http.StatusUnsupportedMediaType))
			Expect(body).
				To(Equal("the body must be an SDP offer with type application/sdp"))
			Expect(publishes).To(BeEmpty())
		})

		It("Should reject a task key that is not a UUID", func() {
			res, body := send(
				app,
				http.MethodPost,
				"/api/v1/whip/cat",
				"key",
				"application/sdp",
				"o",
			)
			Expect(res.StatusCode).To(Equal(http.StatusNotFound))
			Expect(body).To(HavePrefix(`task "cat" does not exist`))
		})

		DescribeTable(
			"Should answer a failed publish with the status code of its error",
			func(err error, code int) {
				publishErr = err
				res, body := publish()
				Expect(res.StatusCode).To(Equal(code))
				Expect(body).To(Equal(err.Error()))
			},
			Entry("access denied", auth.ErrAccessDenied, http.StatusForbidden),
			Entry("no license", license.ErrMissing, http.StatusForbidden),
			Entry(
				"invalid credentials",
				auth.ErrInvalidCredentials,
				http.StatusUnauthorized,
			),
			Entry("not found", query.ErrNotFound, http.StatusNotFound),
			Entry("validation", validate.ErrValidation, http.StatusBadRequest),
			Entry("unknown", errors.New("cat"), http.StatusInternalServerError),
		)
	})

	Describe("Unpublish", func() {
		It("Should end a session", func() {
			res, _ := send(
				app,
				http.MethodDelete,
				path+"/"+session.String(),
				"key",
				"",
				"",
			)
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(unpublishes).To(Equal([]apiwhip.UnpublishRequest{
				{Task: task, Session: session, Key: "key"},
			}))
		})

		It("Should reject a session key that is not a UUID", func() {
			res, body := send(app, http.MethodDelete, path+"/cat", "key", "", "")
			Expect(res.StatusCode).To(Equal(http.StatusNotFound))
			Expect(body).To(HavePrefix(`session "cat" does not exist`))
			Expect(unpublishes).To(BeEmpty())
		})
	})

	Describe("Use", func() {
		It("Should run the middleware of both servers before their handlers", func() {
			tr.Use(freighter.MiddlewareFunc(func(
				ctx freighter.Context,
				_ freighter.Next,
			) (freighter.Context, error) {
				return ctx, license.ErrMissing
			}))
			res, body := publish()
			Expect(res.StatusCode).To(Equal(http.StatusForbidden))
			Expect(body).To(Equal(license.ErrMissing.Error()))
			res, _ = send(
				app,
				http.MethodDelete,
				path+"/"+session.String(),
				"key",
				"",
				"",
			)
			Expect(res.StatusCode).To(Equal(http.StatusForbidden))
			Expect(publishes).To(BeEmpty())
			Expect(unpublishes).To(BeEmpty())
		})

		It("Should answer a panic in a handler as an internal error", func() {
			tr.Use(recovery.Middleware(alamos.Instrumentation{}))
			tr.Publish().BindHandler(func(
				context.Context,
				apiwhip.PublishRequest,
			) (apiwhip.PublishResponse, error) {
				panic("cat")
			})
			res, body := publish()
			Expect(res.StatusCode).To(Equal(http.StatusInternalServerError))
			Expect(body).To(Equal(recovery.ErrPanic.Error()))
		})
	})

	It("Should refuse trickle ICE", func() {
		res, _ := send(app, http.MethodPatch, path+"/"+session.String(), "key", "", "")
		Expect(res.StatusCode).To(Equal(http.StatusMethodNotAllowed))
	})

	It("Should report the WHIP protocol", func() {
		Expect(tr.Report()).To(HaveKeyWithValue("protocol", "whip"))
		Expect(tr.Publish().Report()).To(HaveKeyWithValue("protocol", "whip"))
	})
})
