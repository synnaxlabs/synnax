//go:build !windows

package cmd

import (
	"github.com/arya-analytics/x/fsutil"
	"golang.org/x/sys/unix"
)

func disablePermissionBits() {
	// Mask the permission bits so all files are readable and writable
	// by the user and readable by the group.
	mask := unix.Umask(int(fsutil.OS_NO))
	mask |= int(fsutil.OS_OTH_RWX)
	unix.Umask(mask)
}
