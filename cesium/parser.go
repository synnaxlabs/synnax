package cesium

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/kv"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/confluence"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"sync"
)

type writeParser struct {
	logger    *zap.Logger
	metrics   createMetrics
	wg        *sync.WaitGroup
	responses confluence.AbstractUnarySource[WriteResponse]
	channels  map[channel.Key]channel.Channel
	header    *kv.Header
}

func (c *writeParser) parse(ctx context.Context, segments []Segment) ([]writeOperation, error) {
	var ops []writeOperation
	for _, seg := range segments {
		ch, ok := c.channels[seg.ChannelKey]
		if !ok {
			return ops, errors.New("invalid channel key")
		}
		op := writeOperation{
			ctx:       ctx,
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

type readParser struct {
	responses *confluence.AbstractUnarySource[IteratorResponse]
	logger    *zap.Logger
	metrics   retrieveMetrics
	wg        *sync.WaitGroup
	errC      chan<- error
}

func (r *readParser) parse(ctx context.Context, ranges []*segment.Range) []readOperation {
	var ops []readOperation
	for _, rng := range ranges {
		for _, header := range rng.Headers {
			seg := header.Sugar(rng.Channel)
			seg.SetBounds(rng.Bounds)
			ops = append(ops, readOperation{
				ctx:       ctx,
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
