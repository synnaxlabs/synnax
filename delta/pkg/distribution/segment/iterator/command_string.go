// Code generated by "stringer -type=Command"; DO NOT EDIT.

package iterator

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Open-0]
	_ = x[Next-1]
	_ = x[Prev-2]
	_ = x[First-3]
	_ = x[Last-4]
	_ = x[NextSpan-5]
	_ = x[PrevSpan-6]
	_ = x[NextRange-7]
	_ = x[Valid-8]
	_ = x[Error-9]
	_ = x[Close-10]
	_ = x[SeekFirst-11]
	_ = x[SeekLast-12]
	_ = x[SeekLT-13]
	_ = x[SeekGE-14]
}

const _Command_name = "OpenNextPrevFirstLastNextSpanPrevSpanNextRangeValidErrorCloseSeekFirstSeekLastSeekLTSeekGE"

var _Command_index = [...]uint8{0, 4, 8, 12, 17, 21, 29, 37, 46, 51, 56, 61, 70, 78, 84, 90}

func (i Command) String() string {
	if i >= Command(len(_Command_index)-1) {
		return "Command(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Command_name[_Command_index[i]:_Command_index[i+1]]
}
