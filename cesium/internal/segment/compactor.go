package segment

import (
	"github.com/synnaxlabs/x/telem"
)

type Compactor struct {
	Density telem.Density
}

func (c Compactor) Compact(sa, sb MD) (MD, bool) {
	s1, s2 := sa, sb
	if sb.Alignment.Before(sa.Alignment) {
		s1, s2 = sb, sa
	}
	if s1.End(c.Density) != s2.Alignment || s1.EndOffset() != s2.Offset || s1.FileKey != s2.FileKey {
		return MD{}, false
	}
	return MD{
		ChannelKey: s1.ChannelKey,
		Alignment:  s1.Alignment,
		Offset:     s1.Offset,
		Size:       s1.Size + s2.Size,
		FileKey:    s1.FileKey,
	}, true
}
