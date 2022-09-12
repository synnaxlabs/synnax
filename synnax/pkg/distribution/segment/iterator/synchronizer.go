package iterator

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/confluence"
)

type synchronizer struct {
	counter          int
	nodeIDs          []core.NodeID
	acknowledgements []Response
	confluence.LinearTransform[Response, Response]
}

func newSynchronizer(nodeIDs []core.NodeID) confluence.Segment[Response, Response] {
	s := &synchronizer{nodeIDs: nodeIDs, counter: 1}
	s.LinearTransform.Transform = s.sync
	return s
}

func (a *synchronizer) sync(_ context.Context, res Response) (Response, bool, error) {
	if res.Counter != a.counter {
		panic("[distribution.iterator] - received out of order response")
	}
	a.acknowledgements = append(a.acknowledgements, res)

	if len(a.acknowledgements) == len(a.nodeIDs) {
		ack := a.buildAck()
		a.acknowledgements = nil
		a.counter++
		return ack, true, nil
	}

	return res, false, nil
}

func (a *synchronizer) buildAck() Response {
	return Response{
		Variant: AckResponse,
		Command: a.acknowledgements[0].Command,
		Ack:     !lo.Contains(lo.Map(a.acknowledgements, func(res Response, _ int) bool { return res.Ack }), false),
		Counter: a.counter,
	}
}
