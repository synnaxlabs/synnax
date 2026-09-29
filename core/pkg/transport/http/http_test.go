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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	apilicense "github.com/synnaxlabs/synnax/pkg/api/license"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/security"
	secmock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	svcmock "github.com/synnaxlabs/synnax/pkg/service/mock"
	thttp "github.com/synnaxlabs/synnax/pkg/transport/http"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("HTTP", func() {
	Describe("Bind", func() {
		var app *fiber.App
		BeforeEach(func() {
			router := MustSucceed(fhttp.NewRouter())
			thttp.Bind(apiLayer, router)
			app = fiber.New()
			router.BindTo(app)
		})
		AfterEach(func() { Expect(app.Shutdown()).To(Succeed()) })

		DescribeTable("Should register a route for each API endpoint",
			func(method, path string) {
				res := MustSucceed(app.Test(httptest.NewRequest(method, path, nil)))
				Expect(res.StatusCode).ToNot(Equal(http.StatusNotFound))
			},
			Entry("unary auth login", http.MethodPost, "/api/v1/auth/login"),
			Entry("unary channel create", http.MethodPost, "/api/v1/channel/create"),
			Entry("unary frame delete", http.MethodPost, "/api/v1/frame/delete"),
			Entry("stream frame writer", http.MethodGet, "/api/v1/frame/write"),
		)

		It("Should not register a route for an unknown path", func() {
			res := MustSucceed(app.Test(
				httptest.NewRequest(http.MethodPost, "/api/v1/does-not-exist", nil),
			))
			Expect(res.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("License gate", func() {
		var (
			app  *fiber.App
			keys svcmock.Keys
		)
		creds := auth.Credentials{Username: "root", Password: "root"}
		BeforeEach(func(ctx SpecContext) {
			node := mock.NewNode(ctx)
			keys = svcmock.NewKeys()
			sec := MustSucceed(security.NewProvider(security.ProviderConfig{
				Insecure: new(true),
				KeySize:  secmock.SmallKeySize,
			}))
			svc := MustOpen(service.OpenLayer(ctx, service.LayerConfig{
				Distribution:    node.Layer,
				Security:        sec,
				Storage:         node.Storage,
				RootCredentials: creds,
				License:         license.ServiceConfig{Anchors: keys.Anchors},
			}))
			layer := MustSucceed(api.NewLayer(api.LayerConfig{
				Service:      svc,
				Distribution: node.Layer,
			}))
			router := MustSucceed(fhttp.NewRouter())
			thttp.Bind(layer, router)
			app = fiber.New()
			router.BindTo(app)
			DeferCleanup(func() { Expect(app.Shutdown()).To(Succeed()) })
		})
		post := func(path, token string, body any) (int, []byte) {
			GinkgoHelper()
			req := httptest.NewRequest(
				http.MethodPost,
				path,
				bytes.NewReader(MustSucceed(json.Marshal(body))),
			)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			req.Header.Set(fiber.HeaderAccept, fiber.MIMEApplicationJSON)
			req.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)
			res := MustSucceed(app.Test(req, fiber.TestConfig{
				Timeout:       10 * time.Second,
				FailOnTimeout: true,
			}))
			return res.StatusCode, MustSucceed(io.ReadAll(res.Body))
		}
		login := func() string {
			GinkgoHelper()
			status, body := post("/api/v1/auth/login", "", creds)
			Expect(status).To(Equal(http.StatusOK), string(body))
			var res apiauth.LoginResponse
			Expect(json.Unmarshal(body, &res)).To(Succeed())
			return res.Token
		}

		DescribeTable(
			"Should reach an ungated endpoint while no license is active",
			func(path string, body func() any) {
				status, res := post(path, login(), body())
				Expect(status).To(Equal(http.StatusOK), string(res))
			},
			Entry(
				"connectivity check",
				"/api/v1/connectivity/check",
				func() any { return struct{}{} },
			),
			Entry(
				"login",
				"/api/v1/auth/login",
				func() any { return creds },
			),
			Entry(
				"license retrieve",
				"/api/v1/license/retrieve",
				func() any { return apilicense.RetrieveRequest{} },
			),
			Entry(
				"license activate",
				"/api/v1/license/activate",
				func() any {
					return apilicense.ActivateRequest{
						Token: keys.Sign(svcmock.NewLicense()),
					}
				},
			),
		)

		It("Should refuse a gated endpoint while no license is active", func(
			ctx SpecContext,
		) {
			status, body := post(
				"/api/v1/channel/retrieve",
				login(),
				channel.RetrieveRequest{},
			)
			Expect(status).To(Equal(http.StatusBadRequest))
			var pld errors.Payload
			Expect(json.Unmarshal(body, &pld)).To(Succeed())
			Expect(errors.Decode(ctx, pld)).To(MatchError(license.ErrMissing))
		})
	})
})
