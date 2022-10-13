package seeker

import "github.com/synnaxlabs/x/telem"

type Seeker interface {
	Seek(position int) (root int, err error)
}

type StaticSeeker struct {
	DataRate telem.Rate
}

func (s StaticSeeker) Seek(position int) (root int, err error) {
	return int(telem.TimeSpan(position) / s.DataRate.Period()), nil
}

//
//type alignment struct {
//	position int
//	root     int
//}
//
//type IndexSeeker struct {
//	rootIndexSize int
//	indexSize     int
//	index         []alignment
//}
//
//func (s IndexSeeker) Seek(position int) (root int, err error) {
//
//}
//
//func (s IndexSeeker) SeekLT(position int) (rootGE int, err erro) {
//	for i, v := range s.index {
//		if v.position < position {
//			if i == 0 {
//				return 0, nil
//			}
//			return s.index[i-1].root, nil
//		}
//		if i == len(s.index)-1 {
//			return v.root, nil
//		}
//	}
//	return.index[0].root, nil
//}
