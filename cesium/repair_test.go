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
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// logFrame is a frame of a log file and its offset.
type logFrame struct {
	off   int64
	frame log.Frame
}

// framesOf returns the valid frames of the log of generation of database db in fs, in
// order, from the first frame to the first that fails at its offset and at the next
// block.
func framesOf(fs xfs.FS, db, generation uint64) []logFrame {
	GinkgoHelper()
	b := MustSucceed(xfs.ReadFile(fs, log.Name(generation)))
	var out []logFrame
	for off := int64(log.FramesStart); ; {
		f, err := log.DecodeFrame(b[off:], db, generation, uint64(off))
		if err != nil {
			next := (off + log.BlockSize - 1) &^ (log.BlockSize - 1)
			if next == off || next >= int64(len(b)) {
				return out
			}
			if f, err = log.DecodeFrame(
				b[next:],
				db,
				generation,
				uint64(next),
			); err != nil {
				return out
			}
			off = next
		}
		out = append(out, logFrame{off: off, frame: f})
		off += int64(f.Size())
	}
}

// rootFiles returns the bytes of each file under fs by its path.
func rootFiles(fs xfs.FS) map[string][]byte {
	GinkgoHelper()
	out := make(map[string][]byte)
	var walk func(dir string)
	walk = func(dir string) {
		for _, info := range MustSucceed(fs.List(dir)) {
			name := path.Join(dir, info.Name())
			if info.IsDir() {
				walk(name)
				continue
			}
			out[name] = MustSucceed(xfs.ReadFile(fs, name))
		}
	}
	walk("")
	return out
}

var _ = Describe("Repair", func() {
	const (
		idx  cesium.ChannelKey = 1
		data cesium.ChannelKey = 2
	)
	var (
		fs xfs.FS
		id uint64
	)
	// repair repairs fs with the identity id, and counts the calls of confirm.
	repair := func(ctx context.Context, confirms *int) (cesium.Report, error) {
		return cesium.Repair(ctx, "", func(cesium.Report) bool {
			*confirms++
			return false
		}, cesium.WithFS(fs), cesium.WithIdentity(id))
	}
	// open opens fs on the pointer log with the identity id.
	open := func(ctx context.Context) (*cesium.DB, error) {
		return cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithIdentity(id),
		)
	}
	// write writes and commits 4 frames of 5 samples to both keys from 1 second.
	write := func(ctx context.Context, db *cesium.DB) {
		GinkgoHelper()
		keys := []cesium.ChannelKey{idx, data}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    telem.SecondTS,
		}))
		for c := range telem.TimeStamp(4) {
			stamps := make([]telem.TimeStamp, 5)
			for i := range stamps {
				stamps[i] = (c*5 + telem.TimeStamp(i) + 1) * telem.SecondTS
			}
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeries(stamps),
				telem.NewSeriesV[int64](1, 2, 3, 4, 5),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
		}
		Expect(w.Close()).To(Succeed())
	}
	BeforeEach(func(ctx SpecContext) {
		fs, id = xfs.NewMem(), 0
		db := MustSucceed(open(ctx))
		id = db.Identity()
		stop := GoRun(ctx, db.Run)
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
		write(ctx, db)
		Expect(errors.Join(stop(), db.Close())).To(Succeed())
	})
	// spoil changes byte at of the frame that index picks among the frames of log 1,
	// and returns that frame.
	spoil := func(pick func(n int) int, at int) logFrame {
		GinkgoHelper()
		frames := framesOf(fs, id, 1)
		Expect(len(frames)).To(BeNumerically(">=", 3))
		f := frames[pick(len(frames))]
		b := MustSucceed(xfs.ReadFile(fs, log.Name(1)))
		b[f.off+int64(at)] ^= 0xFF
		Expect(xfs.WriteFile(fs, log.Name(1), b)).To(Succeed())
		return f
	}
	middle := func(n int) int { return n / 2 }

	It("Should report no damage of a healthy root, call no confirm, and write nothing",
		func(ctx SpecContext) {
			before := rootFiles(fs)
			confirms := 0
			rep := MustSucceed(repair(ctx, &confirms))
			Expect(rep.Damage).To(BeEmpty())
			Expect(rep.Lost).To(BeEmpty())
			Expect(confirms).To(BeZero())
			Expect(rootFiles(fs)).To(Equal(before))
		})

	It("Should report a bad frame that valid frames follow, and each record that it "+
		"held", func(ctx SpecContext) {
		// A changed check leaves the entries whole.
		f := spoil(middle, 4)
		Expect(open(ctx)).Error().To(SatisfyAll(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("holds a valid frame at")),
		))
		before := rootFiles(fs)
		confirms := 0
		rep := MustSucceed(repair(ctx, &confirms))
		last := f.frame.Sequence + uint64(f.frame.Count) - 1
		Expect(rep.Damage).To(Equal([]cesium.Damage{{
			Generation:    1,
			Offset:        uint64(f.off),
			FirstSequence: f.frame.Sequence,
			LastSequence:  last,
		}}))
		Expect(rep.Lost).ToNot(BeEmpty())
		for _, rec := range rep.Lost {
			Expect(rec.Sequence).To(BeNumerically(">=", f.frame.Sequence))
			Expect(rec.Sequence).To(BeNumerically("<=", last))
			Expect(rec.Kind).ToNot(BeEmpty())
			Expect(rec.Reason).To(ContainSubstring("from a damaged frame"))
		}
		Expect(rootFiles(fs)).To(Equal(before))
	})

	It("Should list by sequence alone each entry of a bad frame that does not decode",
		func(ctx SpecContext) {
			// A changed entry count bounds the entries by the frame's bytes.
			f := spoil(middle, log.FrameHeaderSize)
			rep := MustSucceed(repair(ctx, new(int)))
			Expect(rep.Damage).To(HaveLen(1))
			last := f.frame.Sequence + uint64(f.frame.Count) - 1
			Expect(rep.Lost).ToNot(BeEmpty())
			Expect(rep.Lost[len(rep.Lost)-1].Sequence).To(Equal(last))
			Expect(rep.Lost[0].Kind).To(BeEmpty())
		})

	It("Should refuse a root of the old layout", func(ctx SpecContext) {
		fs = xfs.NewMem()
		Expect(xfs.WriteFile(MustSucceed(fs.Sub("1")), "meta.json", []byte("{}"))).
			To(Succeed())
		Expect(repair(ctx, new(int))).Error().To(MatchError(errors.ErrUnsupported))
	})

	It("Should refuse another identity than the root's, and undo nothing",
		func(ctx SpecContext) {
			before := rootFiles(fs)
			id++
			Expect(repair(ctx, new(int))).Error().
				To(MatchError(ContainSubstring("which the caller holds")))
			Expect(rootFiles(fs)).To(Equal(before))
		})
})
