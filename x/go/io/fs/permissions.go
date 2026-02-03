// Copyright 2026 Synnax Labs, Inc.
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
	rBit = 0o4
	wBit = 0o2
	xBit = 0o1

	userShift  = 6
	groupShift = 3
	otherShift = 0

	UserR   os.FileMode = rBit << userShift
	UserW   os.FileMode = wBit << userShift
	UserX   os.FileMode = xBit << userShift
	UserRW  os.FileMode = UserR | UserW
	UserRX  os.FileMode = UserR | UserX
	UserWX  os.FileMode = UserW | UserX
	UserRWX os.FileMode = UserRW | UserX

	GroupR   os.FileMode = rBit << groupShift
	GroupW   os.FileMode = wBit << groupShift
	GroupX   os.FileMode = xBit << groupShift
	GroupRW  os.FileMode = GroupR | GroupW
	GroupRX  os.FileMode = GroupR | GroupX
	GroupWX  os.FileMode = GroupW | GroupX
	GroupRWX os.FileMode = GroupRW | GroupX

	OtherR   os.FileMode = rBit << otherShift
	OtherW   os.FileMode = wBit << otherShift
	OtherX   os.FileMode = xBit << otherShift
	OtherRW  os.FileMode = OtherR | OtherW
	OtherRX  os.FileMode = OtherR | OtherX
	OtherWX  os.FileMode = OtherW | OtherX
	OtherRWX os.FileMode = OtherRW | OtherX
)

// HasSufficientPermissions checks if the given actual file mode grants at least the
// permissions expected by the given threshold. If the actual file mode is insufficient,
// false is returned. Otherwise, true is returned.
func HasSufficientPermissions(actual, threshold os.FileMode) bool {
	return actual&threshold == threshold
}
