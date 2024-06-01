// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
	"os"
)

type FSFactory func() (xfs.FS, func() error)

var FileSystems = map[string]FSFactory{
	"memFS": func() (xfs.FS, func() error) {
		return MustSucceed(xfs.NewMem().Sub("testdata")), func() error { return nil }
	},
	"osFS": func() (xfs.FS, func() error) {
		dirName := MustSucceed(os.MkdirTemp("", "test-*"))
		return MustSucceed(xfs.Default.Sub(dirName)), func() error { return xfs.Default.Remove(dirName) }
	},
}
