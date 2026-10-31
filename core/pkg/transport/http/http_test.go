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
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	apilicense "github.com/synnaxlabs/synnax/pkg/api/license"
	apirack "github.com/synnaxlabs/synnax/pkg/api/rack"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	thttp "github.com/synnaxlabs/synnax/pkg/transport/http"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

func openLicensed(ctx SpecContext, creds auth.Credentials) *fiber.App {
	GinkgoHelper()
	node := testutil.NewNode(ctx)
	signer := svctestutil.NewSigner()
	lic := svctestutil.NewLicense()
	key := append([]byte("license/"), lic.Jti.String()...)
	Expect(node.Layer.DB.KV().Set(ctx, key, []byte(signer.Sign(lic)))).To(Succeed())
	sec := MustSucceed(security.NewProvider(security.ProviderConfig{
		Insecure: new(true),
	}))
	svc := MustOpen(service.OpenLayer(ctx, service.LayerConfig{
		Distribution:    node.Layer,
		Security:        sec,
		Storage:         node.Storage,
		RootCredentials: creds,
		License:         license.ServiceConfig{Anchors: signer.Anchors},
	}))
	layer := MustSucceed(api.NewLayer(api.LayerConfig{
		Service:      svc,
		Distribution: node.Layer,
	}))
	router := MustSucceed(fhttp.NewRouter())
	thttp.Bind(layer, router)
	app := fiber.New()
	router.BindTo(app)
	DeferCleanup(func() { Expect(app.Shutdown()).To(Succeed()) })
	return app
}

func postJSON(
	app *fiber.App,
	path string,
	headers map[string]string,
	body any,
) *http.Response {
	GinkgoHelper()
	req := httptest.NewRequest(
		http.MethodPost,
		path,
		bytes.NewReader(MustSucceed(json.Marshal(body))),
	)
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	req.Header.Set(fiber.HeaderAccept, fiber.MIMEApplicationJSON)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if host, ok := headers[fiber.HeaderHost]; ok {
		req.Host = host
	}
	return MustSucceed(app.Test(req, fiber.TestConfig{
		Timeout:       10 * time.Second,
		FailOnTimeout: true,
	}))
}

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
			Entry("unary frame read", http.MethodPost, "/api/v1/frame/read"),
			Entry("stream frame writer", http.MethodGet, "/api/v1/frame/write"),
			Entry("unary task rename", http.MethodPost, "/api/v1/task/rename"),
			Entry("unary device rename", http.MethodPost, "/api/v1/device/rename"),
			Entry("unary rack rename", http.MethodPost, "/api/v1/rack/rename"),
			Entry("unary status rename", http.MethodPost, "/api/v1/status/rename"),
			Entry("unary view rename", http.MethodPost, "/api/v1/view/rename"),
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
			app    *fiber.App
			signer svctestutil.Signer
		)
		creds := auth.Credentials{Username: "root", Password: "root"}
		BeforeEach(func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			signer = svctestutil.NewSigner()
			sec := MustSucceed(security.NewProvider(security.ProviderConfig{
				Insecure: new(true),
			}))
			svc := MustOpen(service.OpenLayer(ctx, service.LayerConfig{
				Distribution:    node.Layer,
				Security:        sec,
				Storage:         node.Storage,
				RootCredentials: creds,
				License:         license.ServiceConfig{Anchors: signer.Anchors},
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

		It("Should log in with a MessagePack session request", func() {
			body := MustSucceed(
				msgpack.Codec.Encode(context.Background(), map[string]any{
					"proof": map[string]string{
						"method":   "password",
						"username": creds.Username,
						"password": creds.Password,
					},
				}),
			)
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/auth/session/create",
				bytes.NewReader(body),
			)
			req.Header.Set(fiber.HeaderContentType, msgpack.Codec.ContentType())
			req.Header.Set(fiber.HeaderAccept, msgpack.Codec.ContentType())
			res := MustSucceed(app.Test(req, fiber.TestConfig{
				Timeout:       10 * time.Second,
				FailOnTimeout: true,
			}))
			raw := MustSucceed(io.ReadAll(res.Body))
			Expect(res.StatusCode).To(Equal(http.StatusOK), string(raw))
			var created apiauth.SessionCreateResponse
			Expect(msgpack.Codec.Decode(context.Background(), raw, &created)).
				To(Succeed())
			Expect(created.Token).To(HavePrefix("sys_"))
		})

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
						Key: signer.Sign(svctestutil.NewLicense()),
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

		It("Should keep a cookie's token out of a refused response", func() {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/auth/session/create",
				bytes.NewReader(MustSucceed(json.Marshal(apiauth.SessionCreateRequest{
					Proof:  auth.Proof{Variant: auth.PasswordProof{Credentials: creds}},
					Cookie: true,
				}))),
			)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			req.Header.Set(fiber.HeaderAccept, fiber.MIMEApplicationJSON)
			created := MustSucceed(app.Test(req))
			Expect(created.StatusCode).To(Equal(http.StatusOK))
			cookies := created.Cookies()
			Expect(cookies).To(HaveLen(1))
			req = httptest.NewRequest(
				http.MethodPost,
				"/api/v1/channel/retrieve",
				bytes.NewReader(MustSucceed(json.Marshal(channel.RetrieveRequest{}))),
			)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			req.Header.Set(fiber.HeaderAccept, fiber.MIMEApplicationJSON)
			req.Header.Set(fiber.HeaderCookie, cookies[0].Name+"="+cookies[0].Value)
			res := MustSucceed(app.Test(req))
			Expect(res.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(res.Header.Get(fiber.HeaderAuthorization)).To(BeEmpty())
		})
	})
	Describe("Session cookie", func() {
		var app *fiber.App
		creds := auth.Credentials{Username: "root", Password: "root"}
		BeforeEach(func(ctx SpecContext) { app = openLicensed(ctx, creds) })
		post := func(path string, headers map[string]string, body any) *http.Response {
			GinkgoHelper()
			return postJSON(app, path, headers, body)
		}
		loginWithCookie := func(host string) *http.Cookie {
			GinkgoHelper()
			res := post(
				"/api/v1/auth/session/create",
				map[string]string{fiber.HeaderHost: host},
				apiauth.SessionCreateRequest{
					Proof:  auth.Proof{Variant: auth.PasswordProof{Credentials: creds}},
					Cookie: true,
				},
			)
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(res.StatusCode).To(Equal(http.StatusOK), string(body))
			var parsed apiauth.SessionCreateResponse
			Expect(json.Unmarshal(body, &parsed)).To(Succeed())
			Expect(parsed.Token).To(BeEmpty())
			cookies := res.Cookies()
			Expect(cookies).To(HaveLen(1))
			_, port := MustSucceed2(net.SplitHostPort(host))
			Expect(cookies[0].Name).To(Equal("synnax-session-" + port))
			Expect(cookies[0].HttpOnly).To(BeTrue())
			Expect(cookies[0].Value).To(HavePrefix("sys_"))
			Expect(
				cookies[0].MaxAge,
			).To(Equal(int(auth.DefaultIdleTimeout / time.Second)))
			return cookies[0]
		}
		// a browser sends back the name and value only, for every Core on the host
		withCookie := func(host string, cs ...*http.Cookie) map[string]string {
			pairs := lo.Map(cs, func(c *http.Cookie, _ int) string {
				return c.Name + "=" + c.Value
			})
			return map[string]string{
				fiber.HeaderHost:   host,
				fiber.HeaderCookie: strings.Join(pairs, "; "),
			}
		}
		It("Should authenticate a request that carries only the cookie", func() {
			cookie := loginWithCookie("localhost:9090")
			res := post(
				"/api/v1/auth/session/renew",
				withCookie("localhost:9090", cookie),
				struct{}{},
			)
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(res.StatusCode).To(Equal(http.StatusOK), string(body))
			var parsed apiauth.SessionRenewResponse
			Expect(json.Unmarshal(body, &parsed)).To(Succeed())
			Expect(parsed.User).ToNot(BeNil())
			Expect(parsed.User.Name).To(Equal(creds.Username))
		})
		It("Should give the cookie a fresh lifetime on any request", func() {
			cookie := loginWithCookie("localhost:9090")
			res := post(
				"/api/v1/auth/session/retrieve",
				withCookie("localhost:9090", cookie),
				apiauth.SessionRetrieveRequest{},
			)
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			refreshed := res.Cookies()
			Expect(refreshed).To(HaveLen(1))
			Expect(refreshed[0].Value).To(Equal(cookie.Value))
			Expect(refreshed[0].MaxAge).
				To(Equal(int(auth.DefaultIdleTimeout / time.Second)))
		})
		It("Should not set a cookie when a header client renews", func() {
			cookie := loginWithCookie("localhost:9090")
			res := post(
				"/api/v1/auth/session/renew",
				map[string]string{
					fiber.HeaderHost:          "localhost:9090",
					fiber.HeaderAuthorization: "Bearer " + cookie.Value,
				},
				struct{}{},
			)
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(res.Cookies()).To(BeEmpty())
		})
		It("Should keep the sessions of Cores on different ports apart", func() {
			first := loginWithCookie("localhost:9090")
			second := loginWithCookie("localhost:9091")
			Expect(first.Name).ToNot(Equal(second.Name))
			renew := func(host string) apiauth.SessionRenewResponse {
				GinkgoHelper()
				res := post(
					"/api/v1/auth/session/renew",
					withCookie(host, first, second),
					struct{}{},
				)
				Expect(res.StatusCode).To(Equal(http.StatusOK))
				var parsed apiauth.SessionRenewResponse
				Expect(json.NewDecoder(res.Body).Decode(&parsed)).To(Succeed())
				return parsed
			}
			Expect(renew("localhost:9090").Session.Key).
				ToNot(Equal(renew("localhost:9091").Session.Key))
		})
		It("Should authenticate a frame read that carries only the cookie", func() {
			cookie := loginWithCookie("localhost:9090")
			headers := withCookie("localhost:9090", cookie)
			headers[fiber.HeaderAccept] = "text/csv"
			res := post("/api/v1/frame/read", headers, apiframer.ReadRequest{})
			body := MustSucceed(io.ReadAll(res.Body))
			Expect(res.StatusCode).To(Equal(http.StatusOK), string(body))
		})
		DescribeTable(
			"Should ignore a client-set session cookie param on a public route",
			func(path string, headers map[string]string) {
				res := post(path, headers, struct{}{})
				Expect(res.StatusCode).To(Equal(http.StatusOK))
				Expect(res.Cookies()).To(BeEmpty())
			},
			Entry(
				"header",
				"/api/v1/connectivity/check",
				map[string]string{apiauth.SessionCookieParam: "x"},
			),
			Entry(
				"query parameter",
				"/api/v1/connectivity/check?freighterctx"+apiauth.SessionCookieParam+"=x",
				nil,
			),
		)
		It("Should refuse a request with neither header nor cookie", func() {
			res := post("/api/v1/auth/session/renew", nil, struct{}{})
			Expect(res.StatusCode).To(Equal(http.StatusBadRequest))
		})
		It("Should clear the cookie when the session logs out", func() {
			cookie := loginWithCookie("localhost:9090")
			renew := post(
				"/api/v1/auth/session/renew",
				withCookie("localhost:9090", cookie),
				struct{}{},
			)
			var parsed apiauth.SessionRenewResponse
			Expect(json.NewDecoder(renew.Body).Decode(&parsed)).To(Succeed())
			res := post(
				"/api/v1/auth/session/delete",
				withCookie("localhost:9090", cookie),
				apiauth.SessionDeleteRequest{
					Keys: []auth.SessionKey{parsed.Session.Key},
				},
			)
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			cleared := res.Cookies()
			Expect(cleared).To(HaveLen(1))
			Expect(cleared[0].Value).To(BeEmpty())
			Expect(cleared[0].MaxAge).To(BeNumerically("<", 0))
			again := post(
				"/api/v1/auth/session/renew",
				withCookie("localhost:9090", cookie),
				struct{}{},
			)
			Expect(again.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})
	Describe("Credentials", func() {
		var app *fiber.App
		creds := auth.Credentials{Username: "root", Password: "root"}
		BeforeEach(func(ctx SpecContext) { app = openLicensed(ctx, creds) })
		call := func(path, token string, body, res any) int {
			GinkgoHelper()
			r := postJSON(
				app,
				path,
				map[string]string{fiber.HeaderAuthorization: "Bearer " + token},
				body,
			)
			raw := MustSucceed(io.ReadAll(r.Body))
			if r.StatusCode == http.StatusOK && res != nil {
				Expect(json.Unmarshal(raw, res)).To(Succeed(), string(raw))
			}
			return r.StatusCode
		}
		login := func(proof auth.Proof) string {
			GinkgoHelper()
			var res apiauth.SessionCreateResponse
			Expect(call(
				"/api/v1/auth/session/create",
				"",
				apiauth.SessionCreateRequest{Proof: proof},
				&res,
			)).To(Equal(http.StatusOK))
			return res.Token
		}
		It("Should create, list, log in with, and delete a key", func() {
			rootToken := login(
				auth.Proof{Variant: auth.PasswordProof{Credentials: creds}},
			)
			var created apiauth.KeyCreateResponse
			Expect(call(
				"/api/v1/auth/key/create",
				rootToken,
				apiauth.KeyCreateRequest{Name: "ci"},
				&created,
			)).To(Equal(http.StatusOK))
			Expect(created.Token).To(HavePrefix("syk_"))
			var listed apiauth.KeyRetrieveResponse
			Expect(call(
				"/api/v1/auth/key/retrieve",
				rootToken,
				apiauth.KeyRetrieveRequest{},
				&listed,
			)).To(Equal(http.StatusOK))
			Expect(listed.Keys).To(ConsistOf(created.Key))
			keyToken := login(auth.Proof{Variant: auth.APIKeyProof{
				APIKeyCredentials: auth.APIKeyCredentials{Key: created.Token},
			}})
			Expect(call("/api/v1/auth/session/renew", keyToken, struct{}{}, nil)).
				To(Equal(http.StatusOK))
			Expect(call(
				"/api/v1/auth/key/delete",
				rootToken,
				apiauth.KeyDeleteRequest{Keys: []apikey.Key{created.Key.Key}},
				nil,
			)).To(Equal(http.StatusOK))
			Expect(call("/api/v1/auth/session/renew", keyToken, struct{}{}, nil)).
				To(Equal(http.StatusBadRequest))
		})
		It("Should set the caller's own password", func() {
			token := login(auth.Proof{Variant: auth.PasswordProof{Credentials: creds}})
			Expect(call(
				"/api/v1/auth/password/set",
				token,
				apiauth.PasswordSetRequest{Password: "new", CurrentPassword: "wrong"},
				nil,
			)).To(Equal(http.StatusBadRequest))
			var set apiauth.PasswordSetResponse
			Expect(call(
				"/api/v1/auth/password/set",
				token,
				apiauth.PasswordSetRequest{
					Password:        "new",
					CurrentPassword: creds.Password,
				},
				&set,
			)).To(Equal(http.StatusOK))
			Expect(call("/api/v1/auth/session/renew", token, struct{}{}, nil)).
				To(Equal(http.StatusBadRequest))
			Expect(call("/api/v1/auth/session/renew", set.Token, struct{}{}, nil)).
				To(Equal(http.StatusOK))
			login(auth.Proof{Variant: auth.PasswordProof{Credentials: auth.Credentials{
				Username: creds.Username,
				Password: "new",
			}}})
		})
		It("Should delete a rack's keys and end their sessions with the rack", func() {
			rootToken := login(
				auth.Proof{Variant: auth.PasswordProof{Credentials: creds}},
			)
			var racks apirack.CreateResponse
			Expect(call(
				"/api/v1/rack/create",
				rootToken,
				apirack.CreateRequest{
					Racks: []rack.Rack{{Name: "gone"}, {Name: "kept"}},
				},
				&racks,
			)).To(Equal(http.StatusOK))
			keyFor := func(r rack.Rack) string {
				GinkgoHelper()
				var created apiauth.KeyCreateResponse
				Expect(call(
					"/api/v1/auth/key/create",
					rootToken,
					apiauth.KeyCreateRequest{
						Subject: rack.OntologyIDs([]rack.Key{r.Key})[0],
					},
					&created,
				)).To(Equal(http.StatusOK))
				return created.Token
			}
			keyProof := func(key string) auth.Proof {
				return auth.Proof{Variant: auth.APIKeyProof{
					APIKeyCredentials: auth.APIKeyCredentials{Key: key},
				}}
			}
			goneKey, keptKey := keyFor(racks.Racks[0]), keyFor(racks.Racks[1])
			goneSession := login(keyProof(goneKey))
			keptSession := login(keyProof(keptKey))
			Expect(call(
				"/api/v1/rack/delete",
				rootToken,
				apirack.DeleteRequest{Keys: []rack.Key{racks.Racks[0].Key}},
				nil,
			)).To(Equal(http.StatusOK))
			Expect(call("/api/v1/auth/session/renew", goneSession, struct{}{}, nil)).
				To(Equal(http.StatusBadRequest))
			Expect(call(
				"/api/v1/auth/session/create",
				"",
				apiauth.SessionCreateRequest{Proof: keyProof(goneKey)},
				nil,
			)).To(Equal(http.StatusBadRequest))
			Expect(call("/api/v1/auth/session/renew", keptSession, struct{}{}, nil)).
				To(Equal(http.StatusOK))
			login(keyProof(keptKey))
		})
	})
})
