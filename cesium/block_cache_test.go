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
	"path"
	"strconv"
	"strings"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// countFS counts the reads of the segment files of the directory dir names.
type countFS struct {
	xfs.FS
	dir, counted string
	reads        *atomic.Int64
}

func (f countFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".seg") ||
		path.Dir(path.Join(f.dir, name)) != f.counted {
		return file, err
	}
	return countFile{File: file, reads: f.reads}, nil
}

func (f countFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	f.FS, f.dir = sub, path.Join(f.dir, name)
	return f, nil
}

type countFile struct {
	xfs.File
	reads *atomic.Int64
}

func (f countFile) ReadAt(p []byte, off int64) (int, error) {
	f.reads.Add(1)
	return f.File.ReadAt(p, off)
}

var _ = Describe("Block cache", func() {
	It("Should look up the stamps of a second read in the blocks that the first "+
		"checked", func(ctx SpecContext) {
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		reads := &atomic.Int64{}
		dir := strconv.FormatUint(uint64(idx), 10)
		fs := countFS{FS: xfs.NewMem(), counted: dir, reads: reads}
		db := MustOpen(cesium.Open(ctx, "", cesium.WithFS(fs), cesium.WithPointerLog()))
		StartRun(db.Run)
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      idx,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		// Twelve full blocks of stamps leave no unfinished last block, which a reader
		// always reads from the file.
		const count = 12 * 8_192
		stamps := make([]telem.TimeStamp, count)
		values := make([]int64, count)
		for i := range count {
			stamps[i] = telem.SecondTS + telem.TimeStamp(i)*telem.MillisecondTS
			values[i] = int64(i)
		}
		keys := []cesium.ChannelKey{idx, data}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    stamps[0],
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries(stamps),
			telem.NewSeries(values),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		tr := telem.TimeRange{Start: stamps[40_000], End: stamps[60_000]}
		read := func() {
			GinkgoHelper()
			fr := MustSucceed(db.Read(ctx, tr, data))
			Expect(fr.Get(data).Len()).To(Equal(int64(20_000)))
		}
		read()
		Expect(reads.Load()).To(BeNumerically(">", 0))
		reads.Store(0)
		read()
		Expect(reads.Load()).To(BeZero())
	})
})
