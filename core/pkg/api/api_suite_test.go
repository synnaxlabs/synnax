// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/batch"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	thttp "github.com/synnaxlabs/synnax/pkg/transport/http"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	apiLayer *api.Layer
	svc      *service.Layer
	db       *gorp.DB
	node     testutil.Node
	// exec runs a batch as the API layer binds it.
	exec freighter.UnaryHandler[batch.ExecRequest, batch.ExecResponse]
	// app serves the API layer over HTTP.
	app *fiber.App
	// rootToken is the session token of the root user.
	rootToken string
)

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite", SuiteConfig())
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = testutil.NewNode(ctx)
	db = node.DB
	rootCreds := auth.Credentials{Username: "root", Password: "root"}
	svc = MustOpen(svctestutil.OpenLayer(
		ctx,
		node,
		service.LayerConfig{RootCredentials: rootCreds},
	))
	errs := api.NewErrorCodec()
	apiLayer = MustSucceed(api.NewLayer(api.LayerConfig{
		Service:      svc,
		Distribution: node.Layer,
		Errors:       errs,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: svc.RBAC,
		Held: apiauth.HeldTables(),
	}))
	exec = bind.Write(binder, apiLayer.Batch.Exec)
	router := MustSucceed(
		fhttp.NewRouter(context.Background(), fhttp.RouterConfig{Errors: errs}),
	)
	thttp.Bind(apiLayer, router)
	app = fiber.New()
	router.BindTo(app)
	DeferCleanup(func() { Expect(app.Shutdown()).To(Succeed()) })
	var login apiauth.LoginResponse
	Expect(post("/api/v1/auth/login", rootCreds, &login)).To(Succeed())
	rootToken = login.Token
})

var _ = ShouldNotLeakGoroutinesPerSpec()

// post sends req to path over HTTP as the root user and decodes the response into res
// unless res is nil. It returns the error that the response holds.
func post(path string, req, res any) error {
	GinkgoHelper()
	httpReq := httptest.NewRequest(
		http.MethodPost,
		path,
		bytes.NewReader(MustSucceed(json.Codec.Encode(req))),
	)
	httpReq.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	httpReq.Header.Set(fiber.HeaderAccept, fiber.MIMEApplicationJSON)
	httpReq.Header.Set(fiber.HeaderAuthorization, "Bearer "+rootToken)
	httpRes := send(httpReq)
	defer func() { Expect(httpRes.Body.Close()).To(Succeed()) }()
	body := MustSucceed(io.ReadAll(httpRes.Body))
	if httpRes.StatusCode != http.StatusOK {
		var pld errors.Payload
		Expect(json.Codec.Decode(body, &pld)).To(Succeed(), string(body))
		return api.NewErrorCodec().Decode(pld)
	}
	if res != nil {
		Expect(json.Codec.Decode(body, res)).To(Succeed())
	}
	return nil
}

// send sends req to app. The caller closes the body of the response.
func send(req *http.Request) *http.Response {
	GinkgoHelper()
	return MustSucceed(app.Test(req, fiber.TestConfig{
		Timeout:       10 * time.Second,
		FailOnTimeout: true,
	}))
}
