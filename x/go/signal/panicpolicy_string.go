// Code generated by "stringer -type=PanicPolicy"; DO NOT EDIT.

package signal

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[PropagatePanic-0]
	_ = x[RecoverNoErr-1]
	_ = x[RecoverErr-2]
	_ = x[Restart-3]
}

const _PanicPolicy_name = "PropagatePanicRecoverNoErrRecoverErrRestart"

var _PanicPolicy_index = [...]uint8{0, 14, 26, 36, 43}

func (i PanicPolicy) String() string {
	if i >= PanicPolicy(len(_PanicPolicy_index)-1) {
		return "PanicPolicy(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _PanicPolicy_name[_PanicPolicy_index[i]:_PanicPolicy_index[i+1]]
}
