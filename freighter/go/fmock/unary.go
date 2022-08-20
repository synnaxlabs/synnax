package fmock

import (
	"context"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
)

// Unary is a mock, synchronous implementation of the freighter.Unary interface.
type Unary[RQ, RS freighter.Payload] struct {
	Address address.Address
	Network *Network[RQ, RS]
	Handler func(context.Context, RQ) (RS, error)
}

func (t *Unary[RQ, RS]) Report() alamos.Report { return reporter.Report() }

// Send implements the freighter.Unary interface.
func (t *Unary[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (res RS, err error) {
	route, ok := t.Network.resolveUnary(target)
	if !ok || route.Handler == nil {
		return res, address.TargetNotFound(target)
	}
	res, err = route.Handler(ctx, req)
	t.Network.appendEntry(t.Address, target, req, res, err)
	return res, err
}

// BindHandler implements the freighter.Unary interface.
func (t *Unary[RQ, RS]) BindHandler(handler func(context.Context, RQ) (RS, error)) {
	t.Handler = handler
}
