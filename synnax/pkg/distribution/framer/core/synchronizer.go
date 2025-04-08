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
	"github.com/synnaxlabs/alamos"
	"go.uber.org/zap"
)

// Synchronizer is used to synchronized sequenced responses across multiple nodes.
// Synchronizer assumes that a request sent to multiple nodes contains a sequence
// number that is incremented with every request.
type Synchronizer struct {
	alamos.Instrumentation
	NodeCount int
	counter   int
	ack       bool
	seqNum    int
}

func (s *Synchronizer) Sync(nodeSeqNum int, nodeAck bool) (ack bool, seqNum int, fulfilled bool) {
	if s.seqNum != nodeSeqNum {
		s.L.Warn("unexpected sequence number", zap.Int("expected", s.seqNum), zap.Int("actual", nodeSeqNum))
		return false, 0, false
	}
	if s.counter == 0 {
		s.ack = true
		s.seqNum = nodeSeqNum
	}
	s.counter++
	// If we have a bad ack for any response, set the ack for the
	if !nodeAck {
		s.ack = false
	}
	if s.counter == s.NodeCount {
		s.counter = 0
		ack = s.ack
		s.ack = true
		return ack, seqNum, true
	}
	return ack, seqNum, false
}
