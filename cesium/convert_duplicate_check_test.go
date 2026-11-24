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
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// oldDataFiles returns the number of old data files of key in fs that hold bytes.
func oldDataFiles(fs xfs.FS, key cesium.ChannelKey) int {
	GinkgoHelper()
	n := 0
	for _, info := range MustSucceed(fs.List(strconv.FormatUint(uint64(key), 10))) {
		name := info.Name()
		if strings.HasSuffix(name, ".domain") && name != "index.domain" &&
			name != "counter.domain" && info.Size() > 0 {
			n++
		}
	}
	return n
}

// sharedStampRoot returns a root of the old layout in memory whose writer commits
// stamps 1 to 3 s, rolls to a new file, then commits 3 and 4 s, which the old layout
// takes (SY-6883). The index gives the new file the domain from 3 s + 1 ns.
func sharedStampRoot(ctx context.Context) xfs.FS {
	GinkgoHelper()
	const idx, data = cesium.ChannelKey(1), cesium.ChannelKey(2)
	keys := []cesium.ChannelKey{idx, data}
	fs := xfs.NewMem()
	db := MustSucceed(cesium.Open(
		ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithFileSizeCap(telem.Byte),
	))
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
	w := MustSucceed(db.OpenWriter(
		ctx,
		cesium.WriterConfig{Channels: keys, Clock: idx, Start: telem.SecondTS},
	))
	MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
		telem.NewSeriesSecondsTSV(1, 2, 3),
		telem.NewSeriesV[int64](1, 2, 3),
	}), nil))
	MustSucceed2(w.Commit(ctx, nil))
	// The first file is over the cap, so the next commit goes to a new file, and
	// the old layout takes the stamp of the last commit again (SY-6883).
	MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
		telem.NewSeriesSecondsTSV(3, 4),
		telem.NewSeriesV[int64](4, 5),
	}), nil))
	MustSucceed2(w.Commit(ctx, nil))
	Expect(w.Close()).To(Succeed())
	Expect(db.Close()).To(Succeed())
	Expect(oldDataFiles(fs, idx)).To(Equal(2))
	return fs
}

var _ = Describe("Check: conversion of a stamp that two old files share", func() {
	It("Should convert a root whose writer wrote its last committed stamp again in a "+
		"new file", func(ctx SpecContext) {
		const idx, data = cesium.ChannelKey(1), cesium.ChannelKey(2)
		keys := []cesium.ChannelKey{idx, data}
		fs := sharedStampRoot(ctx)
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		))
		stop := StartRun(db.Run)
		// The repeated stamp is cut as a rise check cuts it inside one file, and the
		// later stamp of the new file stays.
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, keys...))
		var stamps []telem.TimeStamp
		for _, s := range fr.Get(idx).Series {
			for i := 0; i+8 <= len(s.Data); i += 8 {
				stamp := telem.TimeStamp(binary.LittleEndian.Uint64(s.Data[i:]))
				stamps = append(stamps, stamp)
			}
		}
		var values []int64
		for _, s := range fr.Get(data).Series {
			for i := 0; i+8 <= len(s.Data); i += 8 {
				values = append(values, int64(binary.LittleEndian.Uint64(s.Data[i:])))
			}
		}
		Expect(stamps).To(Equal([]telem.TimeStamp{
			telem.SecondTS,
			2 * telem.SecondTS,
			3 * telem.SecondTS,
			4 * telem.SecondTS,
		}))
		Expect(values).To(Equal([]int64{1, 2, 3, 5}))
		stop()
		Expect(db.Close()).To(Succeed())
	})

	It("Should refuse to convert a file whose domain overlaps the one before it, and "+
		"name a file of the domain", func(ctx SpecContext) {
		fs := sharedStampRoot(ctx)
		// The index of the clock gives file 2 a domain from 3 s, inside the domain of
		// file 1, which no cut can place.
		b := MustSucceed(xfs.ReadFile(fs, "1/index.domain"))
		moved := false
		for i := 0; i+26 <= len(b); i += 26 {
			if binary.LittleEndian.Uint16(b[i+16:]) == 2 {
				binary.LittleEndian.PutUint64(b[i:], uint64(3*telem.SecondTS))
				moved = true
			}
		}
		Expect(moved).To(BeTrue())
		Expect(xfs.WriteFile(fs, "1/index.domain", b)).To(Succeed())
		Expect(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		)).Error().To(MatchError(SatisfyAny(
			ContainSubstring("1/2.domain"),
			ContainSubstring("2/2.domain"),
		)))
	})
})
