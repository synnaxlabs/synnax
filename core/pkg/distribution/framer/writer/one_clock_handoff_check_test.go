// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/control"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("One clock handoff check", func() {
	var (
		db        *testDB
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
	)
	BeforeEach(func(ctx SpecContext) {
		db = mustOpenDBOnFS(ctx, FileSystems["memFS"]())
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      idx,
				Name:     "d17h_idx",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      data,
				Name:     "d17h_data",
				Index:    idx,
				DataType: telem.Int64T,
			},
		)).To(Succeed())
		keys = []cesium.ChannelKey{idx, data}
	})
	fr := func(values ...int64) cesium.Frame {
		stamps := make([]telem.TimeStamp, len(values))
		for i, v := range values {
			stamps[i] = telem.TimeStamp(v) * telem.SecondTS
		}
		return telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries(stamps),
			telem.NewSeries(values),
		})
	}
	open := func(ctx SpecContext, cfg storageConfig) *storageWriter {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, cfg))
		DeferCleanup(func() {
			Expect(w.Close()).To(Or(Succeed(), MatchError(validate.ErrValidation)))
		})
		return w
	}
	stored := func(ctx SpecContext, tr telem.TimeRange) []int64 {
		GinkgoHelper()
		var out []int64
		for _, s := range MustSucceed(db.Read(ctx, tr, data)).
			Get(data).Series {
			for i := 0; i+8 <= len(s.Data); i += 8 {
				out = append(out, int64(binary.LittleEndian.Uint64(s.Data[i:])))
			}
		}
		return out
	}
	// lengths returns the number of stored samples of the index and of the data.
	lengths := func(ctx SpecContext) (index, values int64) {
		GinkgoHelper()
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx, data))
		for _, s := range fr.Get(idx).Series {
			index += s.Len()
		}
		for _, s := range fr.Get(data).Series {
			values += s.Len()
		}
		return index, values
	}

	DescribeTable(
		"Should give the clock back from its last commit after a stream-only writer "+
			"held it",
		func(ctx SpecContext, giveBack func(*storageWriter) error) {
			a := open(ctx, storageConfig{
				ControlSubject: control.Subject{Key: "a"},
				Channels:       keys,
				Start:          10 * telem.SecondTS,
				Authorities:    []control.Authority{100},
			})
			MustSucceed(a.Write(ctx, fr(10, 11, 12)))
			MustSucceed(a.Commit(ctx))
			b := MustSucceed(db.OpenWriter(ctx, storageConfig{
				ControlSubject: control.Subject{Key: "b"},
				Channels:       keys,
				Start:          20 * telem.SecondTS,
				Mode:           writer.ModeStreamOnly,
				Authorities:    []control.Authority{200},
			}))
			Expect(giveBack(b)).To(Succeed())
			MustSucceed(a.Write(ctx, fr(13, 14)))
			MustSucceed(a.Commit(ctx))
			Expect(stored(ctx, telem.TimeRangeMax)).
				To(Equal([]int64{10, 11, 12, 13, 14}))
			Expect(stored(ctx, telem.TimeRange{
				Start: 13 * telem.SecondTS,
				End:   15 * telem.SecondTS,
			})).To(Equal([]int64{13, 14}))
		},
		Entry("close", func(b *storageWriter) error { return b.Close() }),
		Entry("set authority", func(b *storageWriter) error {
			DeferCleanup(func() { Expect(b.Close()).To(Succeed()) })
			return b.SetAuthority(storageConfig{
				Channels:    []cesium.ChannelKey{idx, data},
				Authorities: []control.Authority{0},
			})
		}),
	)

	DescribeTable(
		"Should commit the samples of the holder that a handoff took the clock from, "+
			"once the clock comes back",
		func(ctx SpecContext, giveBack func(*storageWriter) error) {
			a := open(ctx, storageConfig{
				ControlSubject: control.Subject{Key: "a"},
				Channels:       keys,
				Start:          10 * telem.SecondTS,
				Authorities:    []control.Authority{100},
			})
			MustSucceed(a.Write(ctx, fr(10, 11)))
			b := MustSucceed(db.OpenWriter(ctx, storageConfig{
				ControlSubject: control.Subject{Key: "b"},
				Channels:       keys,
				Start:          20 * telem.SecondTS,
				Authorities:    []control.Authority{200},
			}))
			Expect(giveBack(b)).To(Succeed())
			Expect(MustSucceed(a.Commit(ctx))).To(Equal(11*telem.SecondTS + 1))
			Expect(stored(ctx, telem.TimeRangeMax)).To(Equal([]int64{10, 11}))
		},
		Entry("close", func(b *storageWriter) error { return b.Close() }),
		Entry("set authority", func(b *storageWriter) error {
			DeferCleanup(func() { Expect(b.Close()).To(Succeed()) })
			return b.SetAuthority(storageConfig{
				Channels:    []cesium.ChannelKey{idx, data},
				Authorities: []control.Authority{0},
			})
		}),
	)

	It("Should commit the stamps that a handoff moves to the new holder, and the "+
		"data of the old holder after its refused write", func(ctx SpecContext) {
		a := open(ctx, storageConfig{
			ControlSubject: control.Subject{Key: "a"},
			Channels:       keys,
			Start:          10 * telem.SecondTS,
			Authorities:    []control.Authority{100},
		})
		MustSucceed(a.Write(ctx, fr(10, 11, 12, 13)))
		b := MustSucceed(db.OpenWriter(ctx, storageConfig{
			ControlSubject: control.Subject{Key: "b"},
			Channels:       []cesium.ChannelKey{idx},
			Start:          10 * telem.SecondTS,
			Authorities:    []control.Authority{200},
		}))
		Expect(a.Commit(ctx)).Error().To(MatchError(control.ErrUnauthorized))
		Expect(a.Write(ctx, fr(20))).Error().To(MatchError(control.ErrUnauthorized))
		// The commit of the new holder commits the stamps of the run that it took.
		MustSucceed(b.Write(ctx, telem.MultiFrame(
			[]cesium.ChannelKey{idx},
			[]telem.Series{telem.NewSeriesSecondsTSV(14)},
		)))
		MustSucceed(b.Commit(ctx))
		Expect(b.Close()).To(Succeed())
		// The commit of the old holder commits its data to its high water.
		Expect(MustSucceed(a.Commit(ctx))).To(Equal(13*telem.SecondTS + 1))
		index, values := lengths(ctx)
		Expect(index).To(Equal(int64(5)))
		Expect(values).To(Equal(int64(4)))
	})
})
