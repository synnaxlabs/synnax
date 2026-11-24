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
	"math/rand/v2"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// removeFailFS fails the removal of each segment file in the directory dir while armed.
type removeFailFS struct {
	xfs.FS
	prefix, dir string
	armed       *atomic.Bool
}

func (f removeFailFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return removeFailFS{
		FS:     sub,
		prefix: path.Join(f.prefix, name),
		dir:    f.dir,
		armed:  f.armed,
	}, nil
}

func (f removeFailFS) Remove(name string) error {
	full := path.Join(f.prefix, name)
	if f.armed.Load() && path.Dir(full) == f.dir && strings.HasSuffix(full, ".seg") {
		return errors.New("disk refused the removal")
	}
	return f.FS.Remove(name)
}

var _ = Describe("Check: a removal that fails at the last unpin of a dropped key",
	func() {
		It("Should read another key after it, and remove the file at the next open",
			func(ctx SpecContext) {
				const idx, data = cesium.ChannelKey(1), cesium.ChannelKey(2)
				mem := xfs.NewMem()
				dir := strconv.FormatUint(uint64(data), 10)
				armed := &atomic.Bool{}
				open := func(fs xfs.FS) (*cesium.DB, func()) {
					GinkgoHelper()
					db := MustSucceed(cesium.Open(
						ctx,
						"",
						cesium.WithFS(fs),
						cesium.WithPointerLog(),
						cesium.WithRandom(rand.NewPCG(1, 2)),
					))
					return db, StartRun(db.Run)
				}
				db, stop := open(removeFailFS{FS: mem, dir: dir, armed: armed})
				Expect(db.CreateChannel(
					ctx,
					cesium.Channel{
						Key:      idx,
						Name:     "time",
						IsIndex:  true,
						DataType: telem.TimestampT,
					},
					cesium.Channel{
						Key:      data,
						Name:     "data",
						Index:    idx,
						DataType: telem.Int64T,
					},
				)).To(Succeed())
				Expect(db.Write(ctx, telem.SecondTS, telem.MultiFrame(
					[]cesium.ChannelKey{idx, data},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2, 3),
						telem.NewSeriesV[int64](1, 2, 3),
					},
				))).To(Succeed())
				// The reader pins the segment of data across its Drop, so its removal
				// waits for the last unpin.
				iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
					Bounds:   telem.TimeRangeMax,
					Channels: []cesium.ChannelKey{data},
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(db.Drop(ctx, []cesium.ChannelKey{data})).To(Succeed())
				armed.Store(true)
				for iter.Next(cesium.AutoSpan) {
				}
				Expect(iter.Close()).To(Succeed())
				read := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx))
				Expect(read.Get(idx).Len()).To(Equal(int64(3)))
				segments := func() int {
					n := 0
					if !MustSucceed(mem.Exists(dir)) {
						return 0
					}
					for _, info := range MustSucceed(mem.List(dir)) {
						if strings.HasSuffix(info.Name(), ".seg") {
							n++
						}
					}
					return n
				}
				Expect(segments()).To(Equal(1))
				armed.Store(false)
				stop()
				Expect(db.Close()).To(Succeed())
				db, stop = open(mem)
				Eventually(segments).WithTimeout(time.Minute).Should(BeZero())
				stop()
				Expect(db.Close()).To(Succeed())
			})
	})
