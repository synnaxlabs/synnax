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

// synchronizer merges the responses of the writer's storage and free branches into one
// response per request. It assumes that each request carries a sequence number
// incremented with every request.
type synchronizer struct {
	confluence.LinearTransform[Response, Response]
	alamos.Instrumentation
	cycle struct {
		res     Response
		counter int
	}
	branchCount int
}

func newSynchronizer(
	branchCount int,
	ins alamos.Instrumentation,
) confluence.Segment[Response, Response] {
	s := &synchronizer{}
	s.branchCount = branchCount
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
	if s.cycle.res.Err == nil {
		s.cycle.res.Err = res.Err
	}
	// A commit that fails on one branch has no end, even if the other applied it.
	if s.cycle.res.Err != nil {
		s.cycle.res.End = 0
	}
	fulfilled := s.cycle.counter == s.branchCount
	if fulfilled {
		s.cycle.counter = 0
	}
	return s.cycle.res, fulfilled, nil
}
