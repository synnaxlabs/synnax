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
	read    = 0o4
	write   = 0o2
	execute = 0o1

	ownerShift  = 6
	groupShift  = 3
	othersShift = 0

	OwnerRead             os.FileMode = read << ownerShift
	OwnerWrite            os.FileMode = write << ownerShift
	OwnerExecute          os.FileMode = execute << ownerShift
	OwnerReadWrite        os.FileMode = OwnerRead | OwnerWrite
	OwnerReadWriteExecute os.FileMode = OwnerReadWrite | OwnerExecute

	GroupRead             os.FileMode = read << groupShift
	GroupWrite            os.FileMode = write << groupShift
	GroupExecute          os.FileMode = execute << groupShift
	GroupReadWrite        os.FileMode = GroupRead | GroupWrite
	GroupReadExecute      os.FileMode = GroupRead | GroupExecute
	GroupReadWriteExecute os.FileMode = GroupReadWrite | GroupExecute

	OthersRead             os.FileMode = read << othersShift
	OthersWrite            os.FileMode = write << othersShift
	OthersExecute          os.FileMode = execute << othersShift
	OthersReadWrite        os.FileMode = OthersRead | OthersWrite
	OthersReadExecute      os.FileMode = OthersRead | OthersExecute
	OthersReadWriteExecute os.FileMode = OthersReadWrite | OthersExecute
)

// HasSufficientPermissions checks if the given actual file mode grants at least the
// permissions expected by the given threshold. If the actual file mode is insufficient,
// false is returned. Otherwise, true is returned.
func HasSufficientPermissions(actual, threshold os.FileMode) bool {
	return actual&threshold == threshold
}
