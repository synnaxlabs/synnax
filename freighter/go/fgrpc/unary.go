package fgrpc

import (
	"context"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	roacherrors "github.com/cockroachdb/errors"
	"google.golang.org/grpc"
)

type UnaryTransport[RQ, RQT, RS, RST freighter.Payload] struct {
	RequestTranslator  Translator[RQ, RQT]
	ResponseTranslator Translator[RS, RST]
	Pool               *Pool
	Client             func(context.Context, grpc.ClientConnInterface, RQT) (RST, error)
	Handler            func(context.Context, RQ) (RS, error)
	ServiceDesc        *grpc.ServiceDesc
}

func (u *UnaryTransport[RQ, RQT, RS, RST]) Report() alamos.Report {
	return reporter.Report()
}

func (u *UnaryTransport[RQ, RQT, RS, RST]) BindTo(reg grpc.ServiceRegistrar) {
	reg.RegisterService(u.ServiceDesc, u)
}

func (u *UnaryTransport[RQ, RQT, RS, RST]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (res RS, err error) {
	conn, err := u.Pool.Acquire(target)
	if err != nil {
		return res, err
	}
	tReq, err := u.RequestTranslator.Forward(req)
	if err != nil {
		return res, err
	}
	tRes, err := u.Client(ctx, conn.ClientConn, tReq)
	if err != nil {
		return res, err
	}
	return u.ResponseTranslator.Backward(tRes)
}

func (u *UnaryTransport[RQ, RQT, RS, RST]) Exec(ctx context.Context, tReq RQT) (tRes RST, err error) {
	req, err := u.RequestTranslator.Backward(tReq)
	if err != nil {
		return tRes, err
	}
	if u.Handler == nil {
		return tRes, roacherrors.New("[freighter]- no handler registered")
	}
	res, err := u.Handler(ctx, req)
	if err != nil {
		return tRes, err
	}
	return u.ResponseTranslator.Forward(res)
}

func (u *UnaryTransport[RQ, RQT, RS, RST]) BindHandler(
	handler func(context.Context, RQ) (RS, error),
) {
	u.Handler = handler
}
