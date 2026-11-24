// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"context"
	"math/rand/v2"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// logWritesFS counts the bytes written to log 2 through it.
type logWritesFS struct {
	xfs.FS
	written *atomic.Int64
}

func (f logWritesFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f logWritesFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || name != log.Name(2) {
		return file, err
	}
	return logWritesFile{File: file, written: f.written}, nil
}

type logWritesFile struct {
	xfs.File
	written *atomic.Int64
}

func (f logWritesFile) Write(p []byte) (int, error) {
	n, err := f.File.Write(p)
	f.written.Add(int64(n))
	return n, err
}

func (f logWritesFile) WriteAt(p []byte, off int64) (int, error) {
	n, err := f.File.WriteAt(p, off)
	f.written.Add(int64(n))
	return n, err
}

var _ = Describe("Check: the next log of a recovery past three quarters of a small log",
	func() {
		const size = log.MinFileSize
		open := func(ctx context.Context, fs xfs.FS) func() error {
			GinkgoHelper()
			db := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
				cesium.WithLogFileSize(telem.Size(size)),
			))
			stop := GoRun(ctx, db.Run)
			return func() error { return errors.Join(stop(), db.Close()) }
		}
		// crashed returns an image, taken while the DB runs, of a root whose log 1
		// holds Intends past three quarters of it. The disk has no room for log 2, so
		// the DB never switches, and the image has room.
		crashed := func(ctx SpecContext) *fault.FS {
			GinkgoHelper()
			disk := MustSucceed(fault.New(fault.Config{Capacity: size + 1<<20}))
			db := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(disk),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
				cesium.WithLogFileSize(telem.Size(size)),
			))
			stop := GoRun(ctx, db.Run)
			// Each Intend takes one block of the log.
			for range size/log.BlockSize*3/4 + 32 {
				Expect(db.Intend([]cesium.ChannelKey{1})).To(Succeed())
			}
			image := MustSucceed(disk.Crash(fault.CrashConfig{}))
			Expect(errors.Join(stop(), db.Close())).To(Succeed())
			return image
		}

		It("Should prepare the new log at the log file size", func(ctx SpecContext) {
			image := crashed(ctx)
			Expect(MustSucceed(image.Exists(log.Name(2)))).To(BeFalse())
			Expect(open(ctx, image)()).To(Succeed())
			Expect(MustSucceed(image.Stat(log.Name(2))).Size()).To(Equal(int64(size)))
		})

		It("Should take a whole new log of the log file size as it is",
			func(ctx SpecContext) {
				image := crashed(ctx)
				Expect(log.Prepare(image, 2, size)).To(Succeed())
				written := &atomic.Int64{}
				Expect(open(ctx, logWritesFS{FS: image, written: written})()).
					To(Succeed())
				Expect(MustSucceed(image.Stat(log.Name(2))).Size()).
					To(Equal(int64(size)))
				// Recovery writes the header copies and the Restart frame of log 2, and
				// does not prepare it again.
				Expect(written.Load()).To(BeNumerically("<", size/2))
			})
	})
