package kv

import (
	"context"
	"github.com/synnaxlabs/x/confluence"
	kvx "github.com/synnaxlabs/x/kv"
)

type persist struct {
	bw kvx.BatchWriter
	confluence.LinearTransform[BatchRequest, BatchRequest]
}

func newPersist(bw kvx.BatchWriter) segment {
	ps := &persist{bw: bw}
	ps.LinearTransform.Transform = ps.persist
	return ps
}

func (ps *persist) persist(ctx context.Context, bd BatchRequest) (BatchRequest, bool, error) {
	err := bd.commitTo(ps.bw)
	return bd, err == nil, nil
}
