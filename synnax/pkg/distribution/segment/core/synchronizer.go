package exec

type Synchronizer struct {
	NodeCount int
	counter   int
	seqNum    int
	ack       bool
}

func (s *Synchronizer) Sync(seqNum int, ack bool) (_ack bool, _seqNum int, fulfilled bool) {
	if seqNum != s.seqNum {
		panic("[distribution.exec] - received out of order response")
	}
	s.counter++
	if s.counter == s.NodeCount {
		s.counter = 0
		_seqNum = s.seqNum
		s.seqNum++
		return true, _seqNum, s.ack
	}
	return ack, s.seqNum, false
}
