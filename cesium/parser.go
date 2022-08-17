package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/kv"
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/confluence"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"sync"
)

type createParser struct {
	ctx       context.Context
	logger    *zap.Logger
	metrics   createMetrics
	wg        *sync.WaitGroup
	responses confluence.AbstractUnarySource[CreateResponse]
	channels  map[channel.Key]channel.Channel
	header    *kv.Header
}

func (c *createParser) parse(segments []Segment) ([]createOperation, error) {
	var ops []createOperation
	for _, seg := range segments {
		ch, ok := c.channels[seg.ChannelKey]
		if !ok {
			return ops, errors.AssertionFailedf("invalid channel key")
		}
		op := createOperationUnary{
			ctx:       c.ctx,
			seg:       seg.Sugar(ch),
			logger:    c.logger,
			kv:        c.header,
			metrics:   c.metrics,
			wg:        c.wg,
			responses: c.responses,
		}
		c.metrics.segSize.Record(int(op.seg.UnboundedSize()))
		ops = append(ops, op)
	}
	return ops, nil
}

type retrieveParser struct {
	responses *confluence.AbstractUnarySource[RetrieveResponse]
	logger    *zap.Logger
	metrics   retrieveMetrics
	wg        *sync.WaitGroup
	errC      chan<- error
}

func (r *retrieveParser) parse(ranges []*segment.Range) []retrieveOperation {
	var ops []retrieveOperation
	for _, rng := range ranges {
		for _, header := range rng.Headers {
			seg := header.Sugar(rng.Channel)
			seg.SetBounds(rng.Bounds)
			ops = append(ops, retrieveOperationUnary{
				ctx:       context.Background(),
				errC:      r.errC,
				seg:       seg,
				dataRead:  r.metrics.dataRead,
				wg:        r.wg,
				logger:    r.logger,
				responses: r.responses,
			})
		}
	}
	return ops
}
