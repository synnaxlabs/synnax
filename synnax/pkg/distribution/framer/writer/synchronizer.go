package writer

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/confluence"
)

type synchronizer struct {
	internal core.Synchronizer
	confluence.LinearTransform[Response, Response]
	artificialSeqNumInc int
	bulkheadSignal      chan bool
}

func newSynchronizer(nodeCount int, bulkheadSig chan bool) confluence.Segment[Response, Response] {
	s := &synchronizer{}
	s.internal.NodeCount = nodeCount
	s.internal.SeqNum = 1
	s.LinearTransform.Transform = s.sync
	s.bulkheadSignal = bulkheadSig
	return s
}

func (a *synchronizer) sync(_ context.Context, res Response) (Response, bool, error) {
	// If we receive a negative ack from a data write on any node, close the bulkhead
	// to prevent more writes from being processed.
	if res.Command == Data && !res.Ack {
		a.bulkheadSignal <- true
		return res, true, nil
	}

	// If the SeqNum is -1, it means the responses is coming from transient errors in
	// the gateway execution pipeline. In this case, we artificially increment the
	// sequence number to ensure the caller receives the correct sequence numbers for
	// future commands.
	if res.SeqNum == -1 {
		res.SeqNum = a.internal.SeqNum
		a.artificialSeqNumInc++
		return res, true, nil
	}

	ack, seqNum, fulfilled := a.internal.Sync(res.SeqNum, res.Ack)
	if fulfilled {
		// If the caller has acknowledged the error by sending an error info request,
		// we're free to open the bulkhead again and allow writes.
		if res.Command == Error {
			a.bulkheadSignal <- false
		}
		return Response{
			Command: res.Command,
			Ack:     ack,
			SeqNum:  seqNum + a.artificialSeqNumInc,
		}, true, nil
	}
	return Response{}, false, nil
}
