package core

type Synchronizer struct {
	NodeCount int
	Counter   int
	SeqNum    int
	ack       bool
}

func (s *Synchronizer) Sync(seqNum int, ack bool) (_ack bool, _seqNum int, fulfilled bool) {
	if seqNum != s.SeqNum {
		panic("[distribution.exec] - received out of order response")
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
