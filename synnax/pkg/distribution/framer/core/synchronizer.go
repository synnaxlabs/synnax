// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/alamos"
	"go.uber.org/zap"
)

// Synchronizer is used to synchronized sequenced responses across multiple nodes.
// Synchronizer assumes that a request sent to multiple nodes contains a sequence
// number that is incremented with every request.
type Synchronizer struct {
	alamos.Instrumentation
	NodeCount int
	cycle     struct {
		counter int
		err     error
		seqNum  int
	}
}

func (s *Synchronizer) Sync(nodeSeqNum int, nodeErr error) (err error, seqNum int, fulfilled bool) {
	if s.cycle.seqNum != nodeSeqNum {
		s.L.Warn("unexpected sequence number", zap.Int("expected", s.cycle.seqNum), zap.Int("actual", nodeSeqNum))
		return s.cycle.err, 0, false
	}
	if s.cycle.counter == 0 {
		s.cycle.err = nil
		s.cycle.seqNum = nodeSeqNum
	}
	s.cycle.counter++
	// If we have a bad ack for any response, set the ack for the
	if nodeErr != nil {
		s.cycle.err = errors.CombineErrors(s.cycle.err, nodeErr)
	}
	if s.cycle.counter == s.NodeCount {
		s.cycle.counter = 0
		return s.cycle.err, seqNum, true
	}
	return s.cycle.err, seqNum, false
}
