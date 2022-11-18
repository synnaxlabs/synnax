package writer

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/core"
	"github.com/synnaxlabs/x/confluence"
)

type synchronizer struct {
	internal core.Synchronizer
	confluence.LinearTransform[Response, Response]
	artificialSeqNumInc int
}

func newSynchronizer(nodeCount int) confluence.Segment[Response, Response] {
	s := &synchronizer{}
	s.internal.NodeCount = nodeCount
	s.internal.SeqNum = 1
	s.LinearTransform.Transform = s.sync
	return s
}

func (a *synchronizer) sync(_ context.Context, res Response) (Response, bool, error) {
	if res.Command == Data {
		return res, true, nil
	}
	// If the SeqNum is -1, it means the response is coming from transient errors in
	// the gateway execution pipeline. In this case, we artificially increment the
	// sequence number to ensure the caller receives the correct sequence numbers for
	// future commands.
	logrus.Info(res)
	if res.SeqNum == -1 {
		res.SeqNum = a.internal.SeqNum
		a.artificialSeqNumInc++
		return res, true, nil
	}
	ack, seqNum, fulfilled := a.internal.Sync(res.SeqNum, res.Ack)
	if fulfilled {
		return Response{
			Command: res.Command,
			Ack:     ack,
			SeqNum:  seqNum + a.artificialSeqNumInc,
		}, true, nil
	}
	return Response{}, false, nil

}
