// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// readLimitFS fails every ReadAt of a data file after the first allowed ones.
type readLimitFS struct {
	xfs.FS
	allowed atomic.Int64
}

func (fs *readLimitFS) Open(name string, flag int) (xfs.File, error) {
	f, err := fs.FS.Open(name, flag)
	if err != nil || name == "index.domain" {
		return f, err
	}
	return &readLimitFile{File: f, fs: fs}, nil
}

type readLimitFile struct {
	xfs.File
	fs *readLimitFS
}

func (f *readLimitFile) ReadAt(p []byte, off int64) (int, error) {
	if f.fs.allowed.Add(-1) < 0 {
		return 0, ErrFault
	}
	return f.File.ReadAt(p, off)
}

var _ = Describe("Lookup errors", func() {
	var (
		fs  *readLimitFS
		idx *index.Domain
	)
	BeforeEach(func(ctx SpecContext) {
		fs = &readLimitFS{FS: MustSucceed(xfs.NewMem().Sub("index"))}
		fs.allowed.Store(1 << 30)
		db := MustOpen(domain.Open(domain.Config{
			FS:              fs,
			Instrumentation: PanicLogger(),
		}))
		Expect(testutil.WriteDomain(
			ctx,
			db,
			(1 * telem.SecondTS).Range(2*telem.SecondTS+1),
			telem.NewSeriesSecondsTSV(1, 2).Data,
		)).To(Succeed())
		idx = &index.Domain{DB: db}
	})

	It("Should return a zero distance when a read fails", func(ctx SpecContext) {
		fs.allowed.Store(1)
		Expect(idx.Distance(
			ctx,
			(1 * telem.SecondTS).Range(2*telem.SecondTS+1),
			index.MustBeContinuous,
		)).Error().To(MatchError(ErrFault))
	})

	It("Should return a zero stamp when a read fails", func(ctx SpecContext) {
		fs.allowed.Store(2)
		Expect(idx.Stamp(ctx, 1*telem.SecondTS, 1, index.MustBeContinuous)).
			Error().To(MatchError(ErrFault))
	})
})
