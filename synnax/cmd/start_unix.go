//go:build !windows

package cmd

import (
	xfs "github.com/synnaxlabs/x/io/fs"
	"golang.org/x/sys/unix"
)

func disablePermissionBits() {
	// Mask the permission bits so all files are readable and writable
	// by the user and readable by the group.
	mask := unix.Umask(int(xfs.OS_NO))
	mask |= int(xfs.OS_OTH_RWX)
	unix.Umask(mask)
}
