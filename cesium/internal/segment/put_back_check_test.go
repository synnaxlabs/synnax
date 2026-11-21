// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var errWrite = errors.New("write fault")

// failFS opens files whose writes fail while *fail is true.
type failFS struct {
	xfs.FS
	fail *bool
}

func (f failFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return failFile{File: file, fail: f.fail}, nil
}

type failFile struct {
	xfs.File
	fail *bool
}

func (f failFile) WriteAt(p []byte, off int64) (int, error) {
	if *f.fail {
		return 0, errWrite
	}
	return f.File.WriteAt(p, off)
}

var _ = Describe("Put back check", func() {
	It("Should put the writer back to its last commit after a failed write", func() {
		fail := false
		fs := failFS{FS: xfs.NewMem(), fail: &fail}
		budget := MustSucceed(segment.NewBudget(8, 8, segment.GoExecutor))
		h := MustSucceed(budget.Create(fs, "3.seg"))
		w := segment.NewWriter(h, cfg)
		data := pattern(40)
		Expect(w.Write(data[:16])).To(Succeed())
		length, tail := w.Commit()
		fail = true
		Expect(w.Write(data[16:32])).To(MatchError(errWrite))
		Expect(w.Len()).To(Equal(int64(length)))
		fail = false
		Expect(w.Write(data[32:])).To(Succeed())
		again, againTail := w.Commit()
		Expect(again).To(Equal(length + 8))
		Expect(againTail).To(Equal(place.Update(tail, data[32:])))
		Expect(h.Close()).To(Succeed())
	})
})
