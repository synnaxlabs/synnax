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

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/testutil"
)

type FSFactory func() (xfs.FS, func() error)

var FileSystems = map[string]FSFactory{
	"memFS": func() (xfs.FS, func() error) {
		return testutil.MustSucceed(xfs.NewMem().Sub("testData")),
			func() error { return nil }
	},
	"osFS": func() (xfs.FS, func() error) {
		dirName := testutil.MustSucceed(os.MkdirTemp("", "test-*"))
		return testutil.MustSucceed(xfs.Default.Sub(dirName)),
			func() error { return xfs.Default.Remove(dirName) }
	},
}

var FileSystemsWithoutAssertion = map[string]FSFactory{
	"memFS": func() (xfs.FS, func() error) {
		m, err := xfs.NewMem().Sub("testData")
		if err != nil {
			panic(err)
		}
		return m, func() error { return nil }
	},
	"osFS": func() (xfs.FS, func() error) {
		dirName, err := os.MkdirTemp("", "test-*")
		if err != nil {
			panic(err)
		}
		d, err := xfs.Default.Sub(dirName)
		if err != nil {
			panic(err)
		}
		return d, func() error { return xfs.Default.Remove(dirName) }
	},
}

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

			destFile, err := destFS.Open(item.Name(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
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
