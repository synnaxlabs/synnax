// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package start

import (
	"github.com/synnaxlabs/x/io/fs"
	"golang.org/x/sys/unix"
)

// DisablePermissionBits stops the process from giving other users any permission on the
// files that it creates. It does nothing on Windows.
func DisablePermissionBits() {
	mask := unix.Umask(0)
	mask |= int(fs.OtherRWX)
	unix.Umask(mask)
}
