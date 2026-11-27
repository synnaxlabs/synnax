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
	"fmt"
	"path"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// fullDiskFS fails, as a full disk does, each write to a log file other than live, so
// no next log file can be made, and each sync of a segment file once segments is set.
// The live log file is whole from its start, so its writes need no new room.
type fullDiskFS struct {
	xfs.FS
	live     string
	segments *atomic.Bool
}

func (f fullDiskFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return fullDiskFS{FS: sub, live: f.live, segments: f.segments}, nil
}

func (f fullDiskFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	switch {
	case strings.HasSuffix(name, ".log") && path.Base(name) != f.live:
		return fullDiskFile{File: file, writes: true}, nil
	case strings.HasSuffix(name, ".seg"):
		return fullDiskFile{File: file, syncs: f.segments}, nil
	}
	return file, nil
}

type fullDiskFile struct {
	xfs.File
	writes bool
	syncs  *atomic.Bool
}

func (f fullDiskFile) WriteAt(p []byte, off int64) (int, error) {
	if f.writes {
		return 0, syscall.ENOSPC
	}
	return f.File.WriteAt(p, off)
}

func (f fullDiskFile) Sync() error {
	if f.syncs != nil && f.syncs.Load() {
		return syscall.ENOSPC
	}
	return f.File.Sync()
}

var _ = Describe("Deletes on a full disk", func() {
	It("Should log a Tombstone, and free its segment before the next frame, after "+
		"every writer's step 4 Seals are refused", func(ctx SpecContext) {
		var segmentsFull atomic.Bool
		mem := xfs.NewMem()
		db := mustOpenDBOnFS(
			ctx,
			fullDiskFS{FS: mem, live: log.Name(1), segments: &segmentsFull},
			cesium.WithPointerLog(),
		)
		idx := GenerateChannelKey()
		keys := []cesium.ChannelKey{idx}
		channels := []cesium.Channel{
			{Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT},
		}
		for range 63 {
			key := GenerateChannelKey()
			keys = append(keys, key)
			channels = append(channels, cesium.Channel{
				Key: key, Name: fmt.Sprint(key), Index: idx, DataType: telem.Int64T,
			})
		}
		Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
		// write writes and commits one sample of each key at i seconds, and syncs, so
		// each commit goes to the log as its own entry.
		write := func(w *cesium.Writer, i int) error {
			series := []telem.Series{telem.NewSeriesSecondsTSV(telem.TimeStamp(i))}
			for range keys[1:] {
				series = append(series, telem.NewSeriesV(int64(i)))
			}
			if _, _, err := w.Write(
				ctx,
				telem.MultiFrame(keys, series),
				nil,
			); err != nil {
				return err
			}
			if _, _, err := w.Commit(ctx, nil); err != nil {
				return err
			}
			return w.Sync()
		}
		filler := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    telem.SecondTS,
		}))
		var (
			err  error
			last int
		)
		for err == nil {
			last++
			err = write(filler, last)
		}
		Expect(err).To(MatchError(ContainSubstring("log file is full")))
		_ = filler.Close()
		segmentsFull.Store(true)
		held := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    telem.TimeStamp(last+1) * telem.SecondTS,
		}))
		// The failed segment sync is step 4: it logs only the Seals, which the full log
		// refuses.
		Expect(write(held, last+1)).To(MatchError(syscall.ENOSPC))
		_ = held.Close()
		Expect(db.DeleteTimeRange(ctx, keys[1:2], telem.TimeRangeMax)).To(Succeed())
		Expect(MustSucceed(db.Read(ctx, telem.TimeRangeMax, keys[1])).Len()).
			To(BeZero())
		Eventually(func() []string {
			var names []string
			for _, info := range MustSucceed(mem.List(fmt.Sprint(keys[1]))) {
				if strings.HasSuffix(info.Name(), ".seg") {
					names = append(names, info.Name())
				}
			}
			return names
		}).WithTimeout(10 * time.Second).Should(BeEmpty())
	})
})
