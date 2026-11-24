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
	"encoding/binary"
	"os"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// partReadFS counts the opens of part files to read.
type partReadFS struct {
	xfs.FS
	reads *atomic.Int64
}

func (f partReadFS) Open(name string, flag int) (xfs.File, error) {
	if strings.HasSuffix(name, ".part") && flag == os.O_RDONLY {
		f.reads.Add(1)
	}
	return f.FS.Open(name, flag)
}

// Sub keeps the count on the FS that Open gives the engine, a Sub of its root.
func (f partReadFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return partReadFS{FS: sub, reads: f.reads}, nil
}

// partedCount is the count of data keys of a parted root, and partedWriters the count
// of writers that each write one sample to every key.
const (
	partedCount   = 1_000
	partedWriters = 250
)

// partedRoot writes a root in fs whose sealed records pass the part size of 8 MiB, and
// returns its keys: the index, then each data key. Each writer starts a new alignment
// run, so each of its commits adds a pointer record to each channel. A reopen and
// Intends then switch the log, and the switch writes the sealed records to a part.
func partedRoot(
	ctx context.Context,
	fs xfs.FS,
) (*cesium.DB, func() error, []cesium.ChannelKey) {
	GinkgoHelper()
	db, closeDB := MustSucceed2(openLog(ctx, fs))
	keys := make([]cesium.ChannelKey, partedCount+1)
	channels := make([]cesium.Channel, partedCount+1)
	keys[0] = 1
	channels[0] = cesium.Channel{
		Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT,
	}
	for i := 1; i <= partedCount; i++ {
		keys[i] = cesium.ChannelKey(i + 1)
		channels[i] = cesium.Channel{
			Key: keys[i], Name: "data", Index: 1, DataType: telem.Int64T,
		}
	}
	Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
	for w := range telem.TimeStamp(partedWriters) {
		series := make([]telem.Series, partedCount+1)
		series[0] = telem.NewSeriesSecondsTSV(w + 1)
		for i := 1; i <= partedCount; i++ {
			series[i] = telem.NewSeriesV[int64](int64(w))
		}
		wr := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    keys[0],
			Start:    (w + 1) * telem.SecondTS,
		}))
		MustSucceed2(wr.Write(ctx, telem.MultiFrame(keys, series), nil))
		MustSucceed2(wr.Commit(ctx, nil))
		Expect(wr.Close()).To(Succeed())
	}
	Expect(closeDB()).To(Succeed())
	db, closeDB = MustSucceed2(openLog(ctx, fs))
	// The switch to log 2 prepares log 3.
	for range 20_000 {
		if MustSucceed(fs.Exists(log.Name(3))) {
			break
		}
		Expect(db.Intend([]cesium.ChannelKey{999})).To(Succeed())
	}
	var h checkpoint.Header
	Eventually(func() error {
		var err error
		h, _, err = checkpoint.Read(fs, 2)
		return err
	}).WithTimeout(time.Minute).Should(Succeed())
	Expect(h.Parts).ToNot(BeEmpty(), "the switch wrote no part")
	return db, closeDB, keys
}

// int64s returns the values of key in a read of db over all time.
func int64s(
	ctx context.Context,
	db *cesium.DB,
	key cesium.ChannelKey,
) ([]int64, error) {
	fr, err := db.Read(ctx, telem.TimeRangeMax, key)
	if err != nil {
		return nil, err
	}
	var out []int64
	for _, s := range fr.Get(key).Series {
		for i := 0; i+8 <= len(s.Data); i += 8 {
			out = append(out, int64(binary.LittleEndian.Uint64(s.Data[i:])))
		}
	}
	return out, nil
}

var _ = Describe("Part read check", func() {
	var (
		fs      partReadFS
		db      *cesium.DB
		closeDB func() error
		keys    []cesium.ChannelKey
	)
	BeforeEach(func(ctx SpecContext) {
		fs = partReadFS{FS: xfs.NewMem(), reads: &atomic.Int64{}}
		db, closeDB, keys = partedRoot(ctx, fs)
		DeferCleanup(func() { Expect(closeDB()).To(Succeed()) })
		// The state tags the parted segments once head 2 commits, so a read of a key
		// reads a part when its tag lands.
		Eventually(func() int64 {
			MustSucceed(int64s(ctx, db, keys[1]))
			return fs.reads.Load()
		}).WithTimeout(time.Minute).Should(BeNumerically(">", 0))
	})

	It("Should read every sample of each key through the part that holds its records",
		func(ctx SpecContext) {
			want := make([]int64, partedWriters)
			for w := range want {
				want[w] = int64(w)
			}
			before := fs.reads.Load()
			for _, key := range keys[1:] {
				Expect(int64s(ctx, db, key)).To(Equal(want), "key %d", key)
			}
			Expect(fs.reads.Load() - before).To(BeNumerically(">=", partedCount-1))
		})

	It("Should read every sample of each key through its part after a reopen",
		func(ctx SpecContext) {
			want := make([]int64, partedWriters)
			for w := range want {
				want[w] = int64(w)
			}
			Expect(closeDB()).To(Succeed())
			db, closeDB = MustSucceed2(openLog(ctx, fs))
			// The open reads each part whole, then tags the segments in the background.
			opened := fs.reads.Load()
			Eventually(func() int64 {
				MustSucceed(int64s(ctx, db, keys[1]))
				return fs.reads.Load() - opened
			}).WithTimeout(time.Minute).Should(BeNumerically(">", 0))
			before := fs.reads.Load()
			for _, key := range keys[1:] {
				Expect(int64s(ctx, db, key)).To(Equal(want), "key %d", key)
			}
			Expect(fs.reads.Load() - before).To(BeNumerically(">=", partedCount-1))
		})

	It("Should fail a read of a key whose part is gone, and not read it as empty",
		func(ctx SpecContext) {
			for _, info := range MustSucceed(fs.List("")) {
				if strings.HasSuffix(info.Name(), ".part") {
					Expect(fs.Remove(info.Name())).To(Succeed())
				}
			}
			Expect(int64s(ctx, db, keys[2])).Error().To(MatchError(os.ErrNotExist))
		})
})
