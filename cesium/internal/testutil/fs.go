// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"io"
	"os"
	"sync/atomic"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
)

// FSFactory is the FS constructor type used by table-style tests in cesium.
// See [fstestutil.Factory].
type FSFactory = fstestutil.Factory

// FileSystems is the canonical map of FS implementation names to factories
// shared across cesium tests. See [fstestutil.FileSystems].
var FileSystems = fstestutil.FileSystems

func CopyFS(srcFS, destFS xfs.FS) error {
	items, err := srcFS.List("")
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.IsDir() {
			// Create directory in destination.
			subDestFS, err := destFS.Sub(item.Name())
			if err != nil {
				return err
			}
			subSrcFS, err := srcFS.Sub(item.Name())
			if err != nil {
				return err
			}

			if err := CopyFS(subSrcFS, subDestFS); err != nil {
				return err
			}
		} else {
			// Copy file from source to destination.
			srcFile, err := srcFS.Open(item.Name(), os.O_RDONLY)
			if err != nil {
				return err
			}

			destFile, err := destFS.Open(
				item.Name(),
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			)
			if err != nil {
				return errors.Combine(err, srcFile.Close())
			}

			if _, err := io.Copy(destFile, srcFile); err != nil {
				srcErr := srcFile.Close()
				dstErr := destFile.Close()
				return errors.Combine(err, errors.Combine(srcErr, dstErr))
			}

			if err := destFile.Sync(); err != nil {
				srcErr := srcFile.Close()
				dstErr := destFile.Close()
				return errors.Combine(err, errors.Combine(srcErr, dstErr))
			}
			err = srcFile.Close()
			if err != nil {
				return errors.Combine(err, destFile.Close())
			}
			err = destFile.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ErrFault is the failure a Fault raises in place of the operation it blocks.
var ErrFault = errors.New("faulty file system")

// FaultOp names a file system operation a FaultyFS can fail.
type FaultOp string

const (
	FaultOpOpen   FaultOp = "open"
	FaultOpReadAt FaultOp = "read_at"
	FaultOpWrite  FaultOp = "write"
	FaultOpRename FaultOp = "rename"
	FaultOpStat   FaultOp = "stat"
	FaultOpRemove FaultOp = "remove"
)

// Fault decides whether an operation on name must fail, returning the error to raise in
// its place. A nil return lets the operation through. Rename passes its old path as
// name.
type Fault = func(op FaultOp, name string) error

// FaultyFS wraps an FS to fail the operations its Fault selects. It also counts the
// file handles open against it, so a test can assert that a failed operation left none
// behind.
type FaultyFS struct {
	xfs.FS
	fault Fault
	open  *atomic.Int64
}

// OpenFaultyFS wraps wrapped so that every operation fault selects fails with the error
// fault returns. Paths reach fault exactly as the caller passed them.
func OpenFaultyFS(wrapped xfs.FS, fault Fault) *FaultyFS {
	return &FaultyFS{FS: wrapped, fault: fault, open: &atomic.Int64{}}
}

// OpenFiles returns the number of handles opened through the FS and not yet closed.
func (fs *FaultyFS) OpenFiles() int { return int(fs.open.Load()) }

func (fs *FaultyFS) Open(name string, flag int) (xfs.File, error) {
	if err := fs.fault(FaultOpOpen, name); err != nil {
		return nil, err
	}
	f, err := fs.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	fs.open.Add(1)
	return &faultyFile{File: f, fs: fs, name: name}, nil
}

// Sub returns a FaultyFS over the sub-directory, sharing this FS's fault and handle
// count. Paths reach the fault relative to the sub-FS.
func (fs *FaultyFS) Sub(name string) (xfs.FS, error) {
	sub, err := fs.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return &FaultyFS{FS: sub, fault: fs.fault, open: fs.open}, nil
}

func (fs *FaultyFS) Rename(oldPath, newPath string) error {
	if err := fs.fault(FaultOpRename, oldPath); err != nil {
		return err
	}
	return fs.FS.Rename(oldPath, newPath)
}

func (fs *FaultyFS) Stat(name string) (xfs.FileInfo, error) {
	if err := fs.fault(FaultOpStat, name); err != nil {
		return nil, err
	}
	return fs.FS.Stat(name)
}

func (fs *FaultyFS) Remove(name string) error {
	if err := fs.fault(FaultOpRemove, name); err != nil {
		return err
	}
	return fs.FS.Remove(name)
}

type faultyFile struct {
	xfs.File
	fs   *FaultyFS
	name string
}

func (f *faultyFile) ReadAt(p []byte, off int64) (int, error) {
	if err := f.fs.fault(FaultOpReadAt, f.name); err != nil {
		return 0, err
	}
	return f.File.ReadAt(p, off)
}

func (f *faultyFile) Write(p []byte) (int, error) {
	if err := f.fs.fault(FaultOpWrite, f.name); err != nil {
		return 0, err
	}
	return f.File.Write(p)
}

func (f *faultyFile) Close() error {
	f.fs.open.Add(-1)
	return f.File.Close()
}
