// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/confluence"
	"go.uber.org/zap"
)

type synchronizer struct {
	confluence.LinearTransform[Response, Response]
	ins   alamos.Instrumentation
	cycle struct {
		res     Response
		counter int
	}
	nodeCount int
}

func newSynchronizer(nodeCount int, ins alamos.Instrumentation) confluence.Segment[Response, Response] {
	s := &synchronizer{nodeCount: nodeCount, ins: ins}
	s.nodeCount = nodeCount
	s.Transform = s.sync
	return s
}

func (s *synchronizer) sync(_ context.Context, res Response) (Response, bool, error) {
	if res.SeqNum == 0 {
		s.ins.L.DPanic(
			"received response with zero sequence number",
			zap.Int("expected", s.cycle.res.SeqNum),
		)
		return res, false, nil
	}

	if res.Variant == ResponseVariantData {
		return res, true, nil
	}

	if s.cycle.counter == 0 {
		s.cycle.res = res
	}

	if res.SeqNum != s.cycle.res.SeqNum {
		s.ins.L.DPanic(
			"received out of order response", zap.Int("expected", s.cycle.res.SeqNum), zap.Int("actual", res.SeqNum),
		)
		return res, false, nil
	}

	s.cycle.counter++

	if !res.Ack {
		s.cycle.res.Ack = false
	}

	fulfilled := s.cycle.counter == s.nodeCount
	if fulfilled {
		s.cycle.counter = 0
	}

	return res, fulfilled, nil
}
