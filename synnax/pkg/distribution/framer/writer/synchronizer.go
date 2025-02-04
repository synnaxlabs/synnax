// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
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
	s.Transform = s.sync
	s.bulkheadSignal = bulkheadSig
	return s
}

func (a *synchronizer) sync(ctx context.Context, res Response) (Response, bool, error) {
	if res.Variant == Control {
		return res, true, nil
	}

	// If the SeqNum is -1, it means the responses is coming from transient errors in
	// the gateway execution pipeline. In this case, we artificially increment the
	// sequence number to ensure the caller receives the correct sequence numbers for
	// future commands.
	if res.SeqNum == -1 {
		res.SeqNum = a.internal.SeqNum
		a.artificialSeqNumInc++
		return res, true, signal.SendUnderContext(ctx, a.bulkheadSignal, true)
	}

	// If we receive a negative ack from a data write on any node, close the validator
	// to prevent more writes from being processed.
	if res.Command == Data && !res.Ack {
		return res, true, signal.SendUnderContext(ctx, a.bulkheadSignal, true)
	}

	ack, seqNum, fulfilled := a.internal.Sync(res.SeqNum, res.Ack)
	if fulfilled {
		// If the caller has acknowledged the error by sending an error info request,
		// we're free to open the validator again and allow writes.
		if res.Command == Error {
			if err := signal.SendUnderContext(ctx, a.bulkheadSignal, false); err != nil {
				return res, true, err
			}
		}
		return Response{
			Command: res.Command,
			Ack:     ack,
			SeqNum:  seqNum + a.artificialSeqNumInc,
			End:     res.End,
		}, true, nil
	}
	return Response{}, false, nil
}
