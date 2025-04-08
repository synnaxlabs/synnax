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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type synchronizer struct {
	internal core.Synchronizer
	confluence.LinearTransform[Response, Response]
	bulkheadSignal chan bool
}

func newSynchronizer(nodeCount int, bulkheadSig chan bool, ins alamos.Instrumentation) confluence.Segment[Response, Response] {
	s := &synchronizer{}
	s.internal.NodeCount = nodeCount
	s.internal.Instrumentation = ins
	s.Transform = s.sync
	s.bulkheadSignal = bulkheadSig
	return s
}

func (a *synchronizer) sync(ctx context.Context, res Response) (Response, bool, error) {
	// If we receive a negative ack from a data write on any node, close the validator
	// to prevent more writes from being processed.
	if res.Command == Data && !res.Ack {
		return res, true, signal.SendUnderContext(ctx, a.bulkheadSignal, true)
	}
	ack, seqNum, fulfilled := a.internal.Sync(res.SeqNum, res.Ack)
	if fulfilled {
		return Response{
			Command: res.Command,
			Ack:     ack,
			SeqNum:  seqNum,
			Error:   res.Error,
			End:     res.End,
		}, true, nil
	}
	return Response{}, false, nil
}
