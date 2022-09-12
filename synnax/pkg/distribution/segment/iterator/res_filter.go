package iterator

import (
	"github.com/synnaxlabs/x/confluence"
)

import (
	"context"
)

type ackFilter struct {
	confluence.Filter[Response]
}

func newAckFilter() *ackFilter {
	rs := &ackFilter{}
	rs.Filter.Apply = rs.filter
	return rs
}

func (rs *ackFilter) filter(ctx context.Context, res Response) (bool, error) {
	return res.Variant == AckResponse, nil
}
