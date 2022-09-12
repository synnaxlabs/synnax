package core

import "github.com/synnaxlabs/x/kfs"

type FileKey uint16

type (
	FS   = kfs.FS[FileKey]
	File = kfs.File[FileKey]
)
