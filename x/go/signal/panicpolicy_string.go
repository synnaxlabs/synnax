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
	_ = x[propagatePanic-0]
	_ = x[recoverNoErr-1]
	_ = x[recoverErr-2]
}

const _panicPolicy_name = "propagatePanicrecoverNoErrrecoverErr"

var _panicPolicy_index = [...]uint8{0, 14, 26, 36}

func (i panicPolicy) String() string {
	if i >= panicPolicy(len(_panicPolicy_index)-1) {
		return "panicPolicy(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _panicPolicy_name[_panicPolicy_index[i]:_panicPolicy_index[i+1]]
}
