// Code generated by "stringer -type=ResponseVariant"; DO NOT EDIT.

package iterator

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[AckResponse-1]
	_ = x[DataResponse-2]
}

const _ResponseVariant_name = "AckResponseDataResponse"

var _ResponseVariant_index = [...]uint8{0, 11, 23}

func (i ResponseVariant) String() string {
	i -= 1
	if i >= ResponseVariant(len(_ResponseVariant_index)-1) {
		return "ResponseVariant(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _ResponseVariant_name[_ResponseVariant_index[i]:_ResponseVariant_index[i+1]]
}
