// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var errRemove = errors.New("remove failed")

// headFailFS fails the second sync of each head file, which follows the write of its
// header sector, and each remove of a head.
type headFailFS struct{ xfs.FS }

func (f headFailFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".head") {
		return file, err
	}
	return &headFailFile{File: file}, nil
}

func (f headFailFS) Remove(name string) error {
	if strings.HasSuffix(name, ".head") {
		if ok, err := f.Exists(name); err != nil || ok {
			return errors.Join(err, errRemove)
		}
	}
	return f.FS.Remove(name)
}

type headFailFile struct {
	xfs.File
	syncs int
}

func (f *headFailFile) Sync() error {
	if f.syncs++; f.syncs == 2 {
		return errSync
	}
	return f.File.Sync()
}

var _ = Describe("Part undo check", func() {
	It("Should keep the parts of a failed head that the switch could not remove",
		func() {
			fs := headFailFS{FS: xfs.NewMem()}
			c := MustSucceed(checkpoint.New(checkpoint.Config{
				FS:       fs,
				DB:       db,
				LogSize:  logSize,
				Prepared: func(uint64, error) {},
				Executor: goExec,
				PartSize: 1,
			}))
			start(c)
			c.Head(2, withSealed(sealed(1, 10)))
			Eventually(c.Err).Should(MatchError(errRemove))
			h, _ := MustSucceed2(checkpoint.Read(fs, 2))
			Expect(h.Parts).To(HaveLen(1))
			MustSucceed(checkpoint.LoadParts(fs, db, h))
		})
})
