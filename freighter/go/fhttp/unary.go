package fhttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/httputil"
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
	oMD, err := s.MiddlewareCollector.Exec(
		c.Context(),
		parseRequestMD(c, address.Address(c.Path())),
		freighter.FinalizerFunc(func(ctx context.Context, md freighter.MD) (oMD freighter.MD, err error) {
			res, err = s.handle(ctx, req)
			return freighter.MD{Protocol: md.Protocol, Params: make(freighter.Params)}, err
		}),
	)
	setResponseMD(c, oMD)
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
	_, err = u.MiddlewareCollector.Exec(
		ctx,
		freighter.MD{Protocol: unaryReporter.Protocol, Target: target},
		freighter.FinalizerFunc(func(ctx context.Context, iMD freighter.MD) (oMD freighter.MD, err error) {
			b, err := u.ecd.Encode(req)
			if err != nil {
				return oMD, err
			}
			httpReq, err := http.NewRequestWithContext(
				ctx,
				"POST",
				"http://"+target.String(),
				bytes.NewReader(b),
			)
			if err != nil {
				return oMD, err
			}
			setRequestMD(httpReq, iMD)
			httpReq.Header.Set(fiber.HeaderContentType, u.ecd.ContentType())

			httpRes, err := (&http.Client{}).Do(httpReq)
			oMD = parseResponseMD(httpRes, target)
			if err != nil {
				return oMD, err
			}

			if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
				var pld ferrors.Payload
				if err := u.ecd.DecodeStream(httpRes.Body, &pld); err != nil {
					return oMD, err
				}
				return oMD, ferrors.Decode(pld)
			}
			return oMD, u.ecd.DecodeStream(httpRes.Body, &res)
		}),
	)
	return res, err
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

func parseRequestMD(c *fiber.Ctx, target address.Address) freighter.MD {
	md := freighter.MD{
		Protocol: unaryReporter.Protocol,
		Target:   target,
		Sec:      parseSecurityInfo(c),
	}
	headers := c.GetReqHeaders()
	md.Params = make(freighter.Params, len(headers))
	for k, v := range c.GetReqHeaders() {
		md.Params[k] = v
	}
	for k, v := range parseQueryString(c) {
		if isFreighterQueryStringParam(k) {
			md.Params[strings.TrimPrefix(k, freighterMDPrefix)] = v
		}
	}
	return md
}

func parseSecurityInfo(c *fiber.Ctx) (info freighter.SecurityInfo) {
	if c.Context().IsTLS() {
		info.TLS.Used = true
		info.TLS.ConnectionState = c.Context().Conn().(*tls.Conn).ConnectionState()
	}
	return info
}

func setRequestMD(c *http.Request, md freighter.MD) {
	for k, v := range md.Params {
		if vStr, ok := v.(string); ok {
			c.Header.Set(freighterMDPrefix+k, vStr)
		}
	}
}

func setResponseMD(c *fiber.Ctx, md freighter.MD) {
	for k, v := range md.Params {
		if vStr, ok := v.(string); ok {
			c.Set(freighterMDPrefix+k, vStr)
		}
	}
}

func parseResponseMD(c *http.Response, target address.Address) freighter.MD {
	md := freighter.MD{Protocol: unaryReporter.Protocol, Target: target, Params: make(freighter.Params, len(c.Header))}
	for k, v := range c.Header {
		md.Params[k] = v[0]
	}
	return md
}

func parseQueryString(c *fiber.Ctx) map[string]string {
	data := make(map[string]string)
	c.Context().QueryArgs().VisitAll(func(key, val []byte) {
		k := utils.UnsafeString(key)
		v := utils.UnsafeString(val)
		data[k] = v
	})
	return data
}

const freighterMDPrefix = "freightermd"

func isFreighterQueryStringParam(k string) bool {
	// check if the key has the md prefix
	return strings.HasPrefix(k, freighterMDPrefix)
}
