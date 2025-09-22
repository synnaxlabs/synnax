// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fs

import "os"

const (
	OS_NO          os.FileMode = 0
	OS_READ        os.FileMode = 0o4
	OS_WRITE       os.FileMode = 0o2
	OS_EX          os.FileMode = 0o1
	OS_USER_SHIFT  os.FileMode = 6
	OS_GROUP_SHIFT os.FileMode = 3
	OS_OTH_SHIFT   os.FileMode = 0

	OS_USER_R   = OS_READ << OS_USER_SHIFT
	OS_USER_W   = OS_WRITE << OS_USER_SHIFT
	OS_USER_X   = OS_EX << OS_USER_SHIFT
	OS_USER_RW  = OS_USER_R | OS_USER_W
	OS_USER_RWX = OS_USER_RW | OS_USER_X

	OS_OTH_R   = OS_READ << OS_OTH_SHIFT
	OS_OTH_W   = OS_WRITE << OS_OTH_SHIFT
	OS_OTH_X   = OS_EX << OS_OTH_SHIFT
	OS_OTH_RW  = OS_OTH_R | OS_OTH_W
	OS_OTH_RWX = OS_OTH_RW | OS_OTH_X
)

// CheckSufficientPermissions checks if the given actual file mode grants at least the
// permissions expected by the given threshold. If the actual file mode is insufficient,
// false is returned. Otherwise, true is returned.
func CheckSufficientPermissions(actual, threshold os.FileMode) bool {
	return actual&threshold == threshold
}
