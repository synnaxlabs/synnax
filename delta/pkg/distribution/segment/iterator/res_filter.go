package iterator

import (
	"context"
	"github.com/arya-analytics/x/confluence"
)

type ackFilter struct {
	confluence.Filter[Response]
}

func newAckRouter(ackMessages confluence.Inlet[Response]) *ackFilter {
	rs := &ackFilter{}
	rs.Filter.Rejects = ackMessages
	rs.Filter.Apply = rs.filter
	return rs
}

func (rs *ackFilter) filter(ctx context.Context, res Response) (bool, error) {
	return res.Variant == DataResponse, nil
}
