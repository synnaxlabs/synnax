// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signal

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Starting-0]
	_ = x[Running-1]
	_ = x[Stopping-2]
	_ = x[Exited-3]
	_ = x[Failed-4]
	_ = x[ContextCanceled-5]
	_ = x[Panicked-6]
}

const _RoutineState_name = "StartingRunningStoppingExitedFailedContextCanceledPanicked"

var _RoutineState_index = [...]uint8{0, 8, 15, 23, 29, 35, 50, 58}

func (i RoutineState) String() string {
	if i >= RoutineState(len(_RoutineState_index)-1) {
		return "RoutineState(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _RoutineState_name[_RoutineState_index[i]:_RoutineState_index[i+1]]
}
