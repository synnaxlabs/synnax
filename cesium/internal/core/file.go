package core

import "github.com/synnaxlabs/x/kfs"

// FileCounter is a counter that generates incrementing, available file keys.
type FileCounter interface {
	// NextFile returns the next available file key.
	NextFile() (FileKey, error)
}

// FileKey is a uint16 key that uniquely identifies a file in the file system.
type FileKey uint16

type (
	// FS is a key-value file system.
	FS = kfs.FS[FileKey]
	// File is a file in cesium's file system.
	File = kfs.File[FileKey]
)
