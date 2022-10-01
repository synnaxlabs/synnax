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

type RouterConfig struct {
	App    *fiber.App
	Logger *zap.SugaredLogger
}

var _ config.Config[RouterConfig] = RouterConfig{}

func (r RouterConfig) Validate() error {
	v := validate.New("[fhttp.Router]")
	validate.NotNil(v, "App", r.App)
	validate.NotNil(v, "Logger", r.Logger)
	return v.Error()
}

func (r RouterConfig) Override(other RouterConfig) RouterConfig {
	r.App = override.Nil(r.App, other.App)
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
	transports []freighter.Transport
}

func (r *Router) Report() alamos.Report {
	return alamos.Report{}
}

func (r *Router) register(t freighter.Transport) {
	r.transports = append(r.transports, t)
}

func StreamServer[RQ, RS freighter.Payload](r *Router, path string) freighter.StreamServer[RQ, RS] {
	s := &streamServer[RQ, RS]{
		Reporter: streamReporter,
		path:     path,
		logger:   r.Logger,
	}
	r.App.Get(path, s.fiberHandler)
	r.register(s)
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
	r.App.Get(path, us.fiberHandler)
	r.register(us)
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
	r.App.Post(path, us.fiberHandler)
	r.register(us)
	return us
}
