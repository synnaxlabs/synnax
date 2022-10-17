package file

import "github.com/synnaxlabs/x/kfs"

type Key uint16

type (
	FS   = kfs.FS[Key]
	File = kfs.File[Key]
)
