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
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// switchLog appends Intends of 1,000 keys each to db on fs until the log switches to
// generation 2, which asks for the file of log 3. It first waits for the file of log 2,
// since log 1 goes on past three quarters until that file is ready. The checkpointer
// writes the head of the switch after it returns.
func switchLog(db *cesium.DB, fs xfs.FS) error {
	for deadline := time.Now().Add(30 * time.Second); ; time.Sleep(time.Millisecond) {
		info, err := fs.Stat(log.Name(2))
		if err == nil && info.Size() == log.FileSize {
			break
		}
		if time.Now().After(deadline) {
			return errors.Newf("the file of log 2 is not ready: %v", err)
		}
	}
	keys := make([]cesium.ChannelKey, 1_000)
	for i := range keys {
		keys[i] = cesium.ChannelKey(1_000_000 + i)
	}
	for range log.FileSize / 16_000 {
		if switched, err := fs.Exists(log.Name(3)); err != nil || switched {
			return err
		}
		if err := db.Intend(keys); err != nil {
			return err
		}
	}
	return errors.New("log 1 did not switch")
}

// partedDB opens a database of the pointer log layout on fs whose switch writes every
// sealed segment to a part.
func partedDB(ctx context.Context, fs xfs.FS) *cesium.DB {
	GinkgoHelper()
	return MustSucceed(cesium.Open(
		ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithPointerLog(),
		cesium.WithPartSize(1),
	))
}

// writeParted writes writers stretches of samples of data, timed by idx, to db, then
// switches log 1, so the samples move to parts. It returns the values of data.
func writeParted(
	ctx context.Context,
	db *cesium.DB,
	fs xfs.FS,
	idx, data cesium.ChannelKey,
	writers, samples int,
) []int64 {
	GinkgoHelper()
	values := make([]int64, 0, writers*samples)
	for w := range writers {
		stamps := make([]telem.TimeStamp, samples)
		vals := make([]int64, samples)
		for i := range samples {
			stamps[i] = telem.TimeStamp(w*samples+i+1) * telem.SecondTS
			vals[i] = int64(w*samples + i)
		}
		values = append(values, vals...)
		writer := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, data},
			Clock:    idx,
			Start:    stamps[0],
		}))
		MustSucceed2(writer.Write(ctx, telem.MultiFrame(
			[]cesium.ChannelKey{idx, data},
			[]telem.Series{telem.NewSeries(stamps), telem.NewSeries(vals)},
		), nil))
		MustSucceed2(writer.Commit(ctx, nil))
		Expect(writer.Close()).To(Succeed())
	}
	Expect(switchLog(db, fs)).To(Succeed())
	Eventually(func() error {
		_, _, err := checkpoint.Read(fs, 2)
		return err
	}).WithTimeout(30 * time.Second).Should(Succeed())
	Expect(partFile(fs)).ToNot(BeEmpty())
	return values
}

// partFile returns the name of a part file in the root of fs, or "".
func partFile(fs xfs.FS) string {
	GinkgoHelper()
	for _, info := range MustSucceed(fs.List("")) {
		if strings.HasSuffix(info.Name(), ".part") {
			return info.Name()
		}
	}
	return ""
}

// flipByte changes one byte at off of the file name in fs.
func flipByte(fs xfs.FS, name string, off int64) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(name, os.O_RDWR))
	b := make([]byte, 1)
	MustSucceed(f.ReadAt(b, off))
	b[0] ^= 0xFF
	MustSucceed(f.WriteAt(b, off))
	Expect(f.Close()).To(Succeed())
}

var _ = Describe("Reads through parts", func() {
	It("Should read the samples of segments that a part holds through the part "+
		"cache", func(ctx SpecContext) {
		fs := xfs.NewMem()
		db := partedDB(ctx, fs)
		stop := StartRun(db.Run)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
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
		values := writeParted(ctx, db, fs, idx, data, 3, 10)
		stop()
		Expect(db.Close()).To(Succeed())

		db = partedDB(ctx, fs)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		DeferCleanup(StartRun(db.Run))
		// A writer's open waits for the state owner, which applies the tags of the
		// parts that the open loaded first.
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, data},
			Clock:    idx,
			Start:    telem.TimeStamp(1_000_000) * telem.SecondTS,
		}))
		Expect(w.Close()).To(Succeed())

		part := partFile(fs)
		flipByte(fs, part, checkpoint.BodyStart+40)
		Expect(db.Read(ctx, telem.TimeRangeMax, data)).Error().
			To(MatchError(place.ErrCorrupt))
		flipByte(fs, part, checkpoint.BodyStart+40)
		frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
		got := make([]int64, 0, len(values))
		for _, s := range frame.Get(data).Series {
			got = append(got, s.Unmarshal[int64]()...)
		}
		Expect(got).To(Equal(values))
	})
})
