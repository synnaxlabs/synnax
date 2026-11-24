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
	"encoding/binary"
	"math/rand/v2"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// flipOnceFS flips one bit of the first read at each offset of each segment file while
// flipping holds. A second read at that offset is good.
type flipOnceFS struct {
	xfs.FS
	dir      string
	flipping *atomic.Bool
	mu       *sync.Mutex
	read     set.Set[string]
}

func (f flipOnceFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".seg") {
		return file, err
	}
	return flipOnceFile{File: file, name: path.Join(f.dir, name), fs: f}, nil
}

func (f flipOnceFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	f.FS, f.dir = sub, path.Join(f.dir, name)
	return f, nil
}

type flipOnceFile struct {
	xfs.File
	name string
	fs   flipOnceFS
}

func (f flipOnceFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := f.File.ReadAt(p, off)
	if n == 0 || !f.fs.flipping.Load() {
		return n, err
	}
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	at := f.name + "@" + string(binary.LittleEndian.AppendUint64(nil, uint64(off)))
	if !f.fs.read.Contains(at) {
		f.fs.read.Add(at)
		p[n/2] ^= 1
	}
	return n, err
}

var _ = Describe("Checked read check", func() {
	It("Should read every committed sample when the first read of each segment "+
		"block is bad", func(ctx SpecContext) {
		flipping := &atomic.Bool{}
		fs := flipOnceFS{
			FS:       xfs.NewMem(),
			flipping: flipping,
			mu:       &sync.Mutex{},
			read:     set.New[string](),
		}
		db := MustOpen(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		))
		StartRun(db.Run)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		// 2,000 samples of 8 bytes span four blocks of each segment.
		const count = 2_000
		stamps := make([]telem.TimeStamp, count)
		values := make([]int64, count)
		for i := range count {
			stamps[i] = 10*telem.SecondTS + telem.TimeStamp(i)*telem.MillisecondTS
			values[i] = int64(i) * 7
		}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    10 * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries(stamps),
			telem.NewSeries(values),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		flipping.Store(true)
		read := func(tr telem.TimeRange) []int64 {
			GinkgoHelper()
			fr := MustSucceed(db.Read(ctx, tr, data))
			var out []int64
			for _, s := range fr.Get(data).Series {
				for i := 0; i+8 <= len(s.Data); i += 8 {
					out = append(out, int64(binary.LittleEndian.Uint64(s.Data[i:])))
				}
			}
			return out
		}
		Expect(read(telem.TimeRangeMax)).To(Equal(values))
		// A bounded read looks up its stamps in the clock segment first.
		Expect(read(telem.TimeRange{
			Start: stamps[600],
			End:   stamps[1_400],
		})).To(Equal(values[600:1_400]))
	})
})
