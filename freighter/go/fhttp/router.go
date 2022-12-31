// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp

import (
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/httputil"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type route struct {
	path       string
	handler    fiber.Handler
	transport  freighter.Transport
	httpMethod string
}

type RouterConfig struct {
	Logger *zap.SugaredLogger
}

var _ config.Config[RouterConfig] = RouterConfig{}

func (r RouterConfig) Validate() error {
	v := validate.New("[fhttp.Router]")
	validate.NotNil(v, "Logger", r.Logger)
	return v.Error()
}

func (r RouterConfig) Override(other RouterConfig) RouterConfig {
	r.Logger = override.Nil(r.Logger, other.Logger)
	return r
}

func NewRouter(configs ...RouterConfig) *Router {
	cfg, err := config.OverrideAndValidate(RouterConfig{}, configs...)
	if err != nil {
		panic(err)
	}
	return &Router{RouterConfig: cfg}
}

type Router struct {
	RouterConfig
	routes []route
}

func (r *Router) BindTo(app *fiber.App) {
	for _, route := range r.routes {
		if route.httpMethod == "GET" {
			app.Get(route.path, route.handler)
		} else {
			app.Post(route.path, route.handler)
		}
	}
}

func (r *Router) Report() alamos.Report {
	return alamos.Report{}
}

func (r *Router) register(
	path string,
	httpMethod string,
	t freighter.Transport,
	h fiber.Handler,
) {
	r.routes = append(r.routes, route{
		httpMethod: httpMethod,
		path:       path,
		handler:    h,
		transport:  t,
	})
}

func StreamServer[RQ, RS freighter.Payload](r *Router, path string) freighter.StreamServer[RQ, RS] {
	s := &streamServer[RQ, RS]{
		Reporter: streamReporter,
		path:     path,
		logger:   r.Logger,
	}
	r.register(path, "GET", s, s.fiberHandler)
	return s
}

func UnaryGetServer[RQ, RS freighter.Payload](r *Router, path string) freighter.UnaryServer[RQ, RS] {
	us := &unaryServer[RQ, RS]{
		Reporter: unaryReporter,
		path:     path,
		requestParser: func(c *fiber.Ctx, ecd httputil.EncoderDecoder) (req RQ, _ error) {
			return req, parseQueryParams(c, &req)
		},
	}
	r.register(path, "GET", us, us.fiberHandler)
	return us
}

func UnaryPostServer[RQ, RS freighter.Payload](r *Router, path string) freighter.UnaryServer[RQ, RS] {
	us := &unaryServer[RQ, RS]{
		Reporter: unaryReporter,
		path:     path,
		requestParser: func(c *fiber.Ctx, ecd httputil.EncoderDecoder) (req RQ, _ error) {
			return req, c.BodyParser(&req)
		},
	}
	r.register(path, "POST", us, us.fiberHandler)
	return us
}
