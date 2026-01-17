// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/confluence"
	"go.uber.org/zap"
)

// Synchronizer is used to synchronize sequenced responses across multiple nodes.
// Synchronizer assumes that a request sent to multiple nodes contains a sequence number
// incremented with every request.
type synchronizer struct {
	alamos.Instrumentation
	confluence.LinearTransform[Response, Response]
	nodeCount int
	cycle     struct {
		counter int
		res     Response
	}
}

func newSynchronizer(nodeCount int, ins alamos.Instrumentation) confluence.Segment[Response, Response] {
	s := &synchronizer{}
	s.nodeCount = nodeCount
	s.Instrumentation = ins
	s.Transform = s.sync
	return s
}

func (s *synchronizer) sync(_ context.Context, res Response) (Response, bool, error) {
	if res.SeqNum == 0 {
		s.L.DPanic(
			"received response with zero sequence number",
			zap.Int("expected", s.cycle.res.SeqNum),
		)
		return res, false, nil
	}

	if s.cycle.counter == 0 {
		s.cycle.res = res
	} else if s.cycle.res.SeqNum != res.SeqNum {
		s.L.DPanic("unexpected sequence number",
			zap.Int("expected", s.cycle.res.SeqNum),
			zap.Int("actual", res.SeqNum),
		)
		return res, false, nil
	}
	s.cycle.counter++

	if !res.Authorized && s.cycle.res.Authorized {
		s.cycle.res.Authorized = false
	}
	if res.Command == CommandCommit && res.End > s.cycle.res.End {
		s.cycle.res.End = res.End
	}
	fulfilled := s.cycle.counter == s.nodeCount
	if fulfilled {
		s.cycle.counter = 0
	}
	return res, fulfilled, nil
}
