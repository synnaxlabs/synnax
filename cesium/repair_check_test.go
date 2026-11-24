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
	"math/rand/v2"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// damaged is a root with a bad frame in the middle of log 1 and valid frames after it
// (fault 5b).
type damaged struct {
	fs        xfs.FS
	id        uint64
	idx, data cesium.ChannelKey
	// first is the alignment of the first sample, and bad the offset of the frame that
	// holds the second of the three commits.
	first telem.Alignment
	bad   int64
}

// commits is the number of samples in each of the three commits of a damaged root.
const commits = 10000

// frames returns the offsets of the frames of log 1 that hold entries.
func frames(fs xfs.FS, id uint64) []int64 {
	GinkgoHelper()
	f := MustSucceed(fs.Open(log.Name(1), os.O_RDONLY))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	info := MustSucceed(f.Stat())
	b := make([]byte, info.Size())
	MustSucceed(f.ReadAt(b, 0))
	var out []int64
	end := info.Size() - log.FrameHeaderSize
	for off := int64(log.FramesStart); off <= end; off += 8 {
		if binary.LittleEndian.Uint32(b[off:]) != 0x464C5043 {
			continue
		}
		fr, err := log.DecodeFrame(b[off:], id, 1, uint64(off))
		if err == nil && fr.Count > 0 {
			out = append(out, off)
		}
	}
	return out
}

// damage builds a root with three synced commits of commits samples each, then a
// delete of 10 samples inside the second commit, and flips a byte in the frame of the
// second commit.
func damage(ctx context.Context) damaged {
	GinkgoHelper()
	d := damaged{fs: xfs.NewMem()}
	db := MustSucceed(cesium.Open(
		ctx,
		"",
		cesium.WithFS(d.fs),
		cesium.WithPointerLog(),
		cesium.WithRandom(rand.NewPCG(1, 2)),
	))
	d.id = db.Identity()
	stop := StartRun(db.Run)
	d.idx, d.data = GenerateChannelKey(), GenerateChannelKey()
	keys := []cesium.ChannelKey{d.idx, d.data}
	Expect(db.CreateChannel(
		ctx,
		cesium.Channel{
			Key: d.idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		},
		cesium.Channel{Key: d.data, Name: "data", Index: d.idx, DataType: telem.Int64T},
	)).To(Succeed())
	w := MustSucceed(db.OpenWriter(
		ctx,
		cesium.WriterConfig{Channels: keys, Start: repairStamp(0)},
	))
	var before []int64
	for c := range 3 {
		stamps := make([]telem.TimeStamp, commits)
		values := make([]int64, commits)
		for i := range stamps {
			p := c*commits + i
			stamps[i], values[i] = repairStamp(p), int64(p)
		}
		fr, _ := MustSucceed2(w.Write(ctx, telem.MultiFrame(
			keys,
			[]telem.Series{telem.NewSeries(stamps), telem.NewSeries(values)},
		), nil))
		if c == 0 {
			d.first = fr.Get(d.idx).Series[0].Alignment
		}
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Sync()).To(Succeed())
		after := frames(d.fs, d.id)
		Expect(len(after)).To(BeNumerically(">", len(before)))
		if c == 1 {
			d.bad = after[len(after)-1]
		}
		before = after
	}
	Expect(w.Close()).To(Succeed())
	Expect(db.DeleteTimeRange(ctx, keys, telem.TimeRange{
		Start: repairStamp(15000),
		End:   repairStamp(15010),
	})).To(Succeed())
	stop()
	Expect(db.Close()).To(Succeed())
	Expect(frames(d.fs, d.id)[len(frames(d.fs, d.id))-1]).To(BeNumerically(">", d.bad))
	flipLog(d.fs, d.bad+log.FrameHeaderSize+16)
	return d
}

// flipLog flips every bit of the byte at offset at of log 1.
func flipLog(fs xfs.FS, at int64) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
	b := make([]byte, 1)
	MustSucceed(f.ReadAt(b, at))
	b[0] ^= 0xFF
	MustSucceed(f.WriteAt(b, at))
	Expect(f.Close()).To(Succeed())
}

func repairStamp(p int) telem.TimeStamp {
	return telem.SecondTS + telem.TimeStamp(p)*telem.MillisecondTS
}

var _ = Describe("Check: repair of log damage", func() {
	It("Should build a root that the open refuses with a valid frame after the bad one",
		func(ctx SpecContext) {
			d := damage(ctx)
			Expect(cesium.Open(
				ctx,
				"",
				cesium.WithFS(d.fs),
				cesium.WithPointerLog(),
				cesium.WithIdentity(d.id),
			)).Error().To(SatisfyAll(
				MatchError(place.ErrCorrupt),
				MatchError(ContainSubstring("holds a valid frame at")),
			))
		})
})
