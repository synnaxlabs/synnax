// Code generated by "stringer -type=Role"; DO NOT EDIT.

package freighter

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Client-1]
	_ = x[Server-2]
}

const _Role_name = "ClientServer"

var _Role_index = [...]uint8{0, 6, 12}

func (i Role) String() string {
	i -= 1
	if i >= Role(len(_Role_index)-1) {
		return "Role(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _Role_name[_Role_index[i]:_Role_index[i+1]]
}
