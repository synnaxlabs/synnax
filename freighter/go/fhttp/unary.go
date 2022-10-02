package fhttp

import (
	"bytes"
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/httputil"
	"net/http"
	"strings"
)

type unaryServer[RQ, RS freighter.Payload] struct {
	freighter.Reporter
	freighter.MiddlewareCollector
	requestParser func(c *fiber.Ctx, ecd httputil.EncoderDecoder) (RQ, error)
	path          string
	handle        func(ctx context.Context, rq RQ) (RS, error)
}

func (s *unaryServer[RQ, RS]) BindHandler(handle func(ctx context.Context, rq RQ) (RS, error)) {
	s.handle = handle
}

func (s *unaryServer[RQ, RS]) fiberHandler(c *fiber.Ctx) error {
	c.Accepts(httputil.SupportedContentTypes()...)
	ecd, err := httputil.DetermineEncoderDecoder(c.Get(fiber.HeaderContentType))
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, ecd.ContentType())
	req, err := s.requestParser(c, ecd)
	if err != nil {
		return err
	}
	var res RS
	err = s.MiddlewareCollector.Exec(
		c.Context(),
		parseRequestParams(c, address.Address(c.Path())),
		freighter.FinalizerFunc(func(ctx context.Context, _ freighter.MD) (err error) {
			res, err = s.handle(ctx, req)
			return err
		}),
	)
	fErr := ferrors.Encode(err)
	if fErr.Type == ferrors.Nil {
		return encodeAndWrite(c, ecd, res)
	}
	c.Status(fiber.StatusBadRequest)
	return encodeAndWrite(c, ecd, fErr)
}

type unaryClient[RQ, RS freighter.Payload] struct {
	freighter.Reporter
	freighter.MiddlewareCollector
	ecd httputil.EncoderDecoder
}

func (u *unaryClient[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (res RS, err error) {
	return res, u.MiddlewareCollector.Exec(
		ctx,
		freighter.MD{Protocol: unaryReporter.Protocol, Target: target},
		freighter.FinalizerFunc(func(ctx context.Context, md freighter.MD) error {
			b, err := u.ecd.Encode(req)
			if err != nil {
				return err
			}
			httpReq, err := http.NewRequestWithContext(
				ctx,
				"POST",
				"http://"+target.String(),
				bytes.NewReader(b),
			)
			if err != nil {
				return err
			}
			httpReq.Header.Set(fiber.HeaderContentType, u.ecd.ContentType())
			for k, v := range md.Params {
				if vStr, ok := v.(string); ok {
					httpReq.Header.Set(k, vStr)
				}
			}
			httpRes, err := (&http.Client{}).Do(httpReq)
			if err != nil {
				return err
			}
			if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
				var pld ferrors.Payload
				if err := u.ecd.DecodeStream(httpRes.Body, &pld); err != nil {
					return err
				}
				return ferrors.Decode(pld)
			}
			return u.ecd.DecodeStream(httpRes.Body, &res)
		}),
	)
}

func encodeAndWrite(c *fiber.Ctx, ecd httputil.EncoderDecoder, v interface{}) error {
	b, err := ecd.Encode(v)
	if err != nil {
		return err
	}
	_, err = c.Write(b)
	return err
}

func parseQueryParams[V any](c *fiber.Ctx, v *V) error {
	return c.QueryParser(v)
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
