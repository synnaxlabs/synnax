package bits

import "github.com/synnaxlabs/x/types"

type Pos int

func (f Pos) Get(b byte) bool {
	return ((b >> f) & 1) == 1
}

func (f Pos) Set(b byte, value bool) byte {
	v := types.BoolToUint8(value) << f
	return b | v
}
