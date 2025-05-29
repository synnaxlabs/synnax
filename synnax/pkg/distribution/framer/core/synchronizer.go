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
	"fmt"
)

type Synchronizer struct {
	NodeCount int
	Counter   int
	SeqNum    int
	ack       bool
}

func (s *Synchronizer) Sync(seqNum int, ack bool) (_ack bool, _seqNum int, fulfilled bool) {
	if seqNum != s.SeqNum {
		fmt.Printf("[distribution.framer.core] - received out of order response: %d, expected: %d", seqNum, s.SeqNum)
	}
	if s.Counter == 0 {
		s.ack = true
	}
	s.Counter++
	if !ack {
		s.ack = false
	}
	if s.Counter == s.NodeCount {
		s.Counter = 0
		_seqNum = s.SeqNum
		s.SeqNum++
		_ack = s.ack
		s.ack = true
		return _ack, _seqNum, true
	}
	return ack, s.SeqNum, false
}
