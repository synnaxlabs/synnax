package fhttp

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/httputil"
	"strings"
)

type unaryServer[RQ, RS freighter.Payload] struct {
	freighter.Reporter
	requestParser func(c *fiber.Ctx, ecd httputil.EncoderDecoder) (RQ, error)
	path          string
	handle        func(ctx context.Context, rq RQ) (RS, error)
	freighter.MiddlewareCollector
}

func (s unaryServer[RQ, RS]) BindHandler(handle func(ctx context.Context, rq RQ) (RS, error)) {
	s.handle = handle
}

func (s unaryServer[RQ, RS]) fiberHandler(c *fiber.Ctx) error {
	return s.MiddlewareCollector.Exec(
		c.Context(),
		parseRequestParams(c, address.Address(c.Path())),
		freighter.FinalizerFunc(func(ctx context.Context, _ freighter.MD) error {
			c.Accepts(httputil.SupportedContentTypes()...)
			ecd, err := httputil.DetermineEncoderDecoder(c.Get(fiber.HeaderContentType))
			if err != nil {
				return err
			}
			req, err := s.requestParser(c, ecd)
			if err != nil {
				return err
			}
			res, err := s.handle(ctx, req)
			b, err := ecd.Encode(res)
			if err != nil {
				return err
			}
			_, err = c.Write(b)
			return err
		}),
	)
}

func parseQueryParams[V any](c *fiber.Ctx, v *V) error {
	return c.QueryParser(v)
}

var unaryReporter = freighter.Reporter{
	Protocol:  "http",
	Encodings: httputil.SupportedContentTypes(),
}

func parseRequestParams(c *fiber.Ctx, target address.Address) freighter.MD {
	md := freighter.MD{Protocol: unaryReporter.Protocol, Target: target}
	headers := c.GetReqHeaders()
	queryParams := c.AllParams()
	md.Params = make(freighter.Params, len(headers)+len(queryParams))
	for k, v := range c.GetReqHeaders() {
		md.Params[k] = v
	}
	for k, v := range c.AllParams() {
		if isFreighterMDParam(k) {
			md.Params[k] = v
		}
	}
	return md
}

const freighterMDPrefix = "freighter.md."

func isFreighterMDParam(k string) bool {
	// check if the key has the md prefix
	return strings.HasPrefix(k, freighterMDPrefix)
}
