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
	timetestutil "github.com/synnaxlabs/x/time/testutil"
)

// logSyncFS fails the next sync of a log file once armed, after the write of its frame,
// as a disk that fails an fsync does.
type logSyncFS struct {
	xfs.FS
	armed, failed *atomic.Bool
}

func (f logSyncFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f logSyncFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".log") {
		return file, err
	}
	return logSyncFile{File: file, fs: f}, nil
}

type logSyncFile struct {
	xfs.File
	fs logSyncFS
}

func (f logSyncFile) Sync() error {
	if f.fs.armed.CompareAndSwap(true, false) {
		f.fs.failed.Store(true)
		return errors.New("disk failed")
	}
	return f.File.Sync()
}

var _ = Describe("Check: a rewrite whose Replace the log refuses", func() {
	It("Should read every live sample after the restart that the failed log needs",
		func(ctx SpecContext) {
			const (
				idx  cesium.ChannelKey = 1
				data cesium.ChannelKey = 2
				n                      = 1 << 19
			)
			mem := xfs.NewMem()
			fs := logSyncFS{FS: mem, armed: &atomic.Bool{}, failed: &atomic.Bool{}}
			fake := timetestutil.NewClock(time.Unix(0, 0))
			open := func(fs xfs.FS) *cesium.DB {
				GinkgoHelper()
				return MustSucceed(cesium.Open(
					ctx,
					"",
					cesium.WithFS(fs),
					cesium.WithPointerLog(),
					cesium.WithClock(fake),
					cesium.WithGCConfig(cesium.GCConfig{Rate: 1 << 20}),
				))
			}
			stamp := func(i int) telem.TimeStamp {
				return telem.TimeStamp(i+1) * telem.SecondTS
			}
			db := open(fs)
			stop := GoRun(ctx, db.Run)
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: data, Name: "data", Index: idx, DataType: telem.Int64T,
				},
			)).To(Succeed())
			write := func(first int, values []int64) {
				GinkgoHelper()
				stamps := make([]telem.TimeStamp, len(values))
				for i := range stamps {
					stamps[i] = stamp(first + i)
				}
				keys := []cesium.ChannelKey{idx, data}
				w := MustSucceed(db.OpenWriter(
					ctx,
					cesium.WriterConfig{
						Channels: keys,
						Clock:    keys[0],
						Start:    stamp(first),
					},
				))
				MustSucceed2(w.Write(ctx, telem.MultiFrame(
					keys,
					[]telem.Series{telem.NewSeries(stamps), telem.NewSeries(values)},
				), nil))
				MustSucceed2(w.Commit(ctx, nil))
				Expect(w.Close()).To(Succeed())
			}
			values := make([]int64, n)
			for i := range values {
				values[i] = int64(i)
			}
			// The second write seals the segment of the first.
			write(0, values)
			write(1<<22, []int64{-1})
			// The delete leaves the segment above the dead share, and its rewrite takes
			// three jobs of 1 MiB, each of which waits a second of the clock.
			Expect(db.DeleteTimeRange(
				ctx,
				[]cesium.ChannelKey{data},
				stamp(0).Range(stamp(1<<17)),
			)).To(Succeed())
			Eventually(fake.Pending).Should(Equal(1))
			fs.armed.Store(true)
			Eventually(func() bool {
				if fake.Pending() > 0 {
					fake.Advance(time.Second)
				}
				return fs.failed.Load()
			}).Should(BeTrue())
			// The log fails at the sync of the entry of the Replace, after the write of
			// its frame.
			failure := "log 1 failed, and no entry is durable until a restart: " +
				"disk failed"
			Expect(stop()).To(MatchError(ContainSubstring(failure)))
			Expect(db.Close()).To(MatchError(ContainSubstring(failure)))
			// A restart reads the log as the failed sync left it.
			re := open(mem)
			stop = GoRun(ctx, re.Run)
			read := func(ctx context.Context) []int64 {
				GinkgoHelper()
				var out []int64
				for _, s := range MustSucceed(re.Read(ctx, telem.TimeRangeMax, data)).
					Get(data).Series {
					for i := range int(s.Len()) {
						out = append(out, s.ValueAt[int64](i))
					}
				}
				return out
			}
			Expect(read(ctx)).To(Equal(append(values[1<<17:], -1)))
			Expect(stop()).To(Succeed())
			Expect(re.Close()).To(Succeed())
		})
})
