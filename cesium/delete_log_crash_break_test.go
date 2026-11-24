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
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	breakIdx  cesium.ChannelKey = 1
	breakData cesium.ChannelKey = 2
)

func breakOpen(ctx context.Context, fs xfs.FS) (*cesium.DB, func()) {
	GinkgoHelper()
	db := MustSucceed(cesium.Open(
		ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithPointerLog(),
		cesium.WithRandom(rand.NewPCG(1, 2)),
	))
	stop := StartRun(db.Run)
	return db, func() {
		stop()
		Expect(db.Close()).To(Succeed())
	}
}

func breakWrite(ctx context.Context, db *cesium.DB, seconds ...telem.TimeStamp) {
	GinkgoHelper()
	keys := []cesium.ChannelKey{breakIdx, breakData}
	values := make([]int64, len(seconds))
	for i, s := range seconds {
		values[i] = int64(s)
	}
	w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
		Channels: keys,
		Clock:    breakIdx,
		Start:    seconds[0] * telem.SecondTS,
	}))
	MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
		telem.NewSeriesSecondsTSV(seconds...),
		telem.NewSeries(values),
	}), nil))
	MustSucceed2(w.Commit(ctx, nil))
	Expect(w.Close()).To(Succeed())
}

func breakValues(ctx context.Context, db *cesium.DB) []int64 {
	GinkgoHelper()
	series := MustSucceed(db.Read(ctx, telem.TimeRangeMax, breakData)).
		Get(breakData).Series
	out := make([]int64, 0, len(series))
	for _, s := range series {
		out = append(out, s.Unmarshal[int64]()...)
	}
	return out
}

func breakSegments(fs xfs.FS, key cesium.ChannelKey) []string {
	GinkgoHelper()
	var out []string
	for _, e := range MustSucceed(fs.List(strconv.FormatUint(uint64(key), 10))) {
		if strings.HasSuffix(e.Name(), ".seg") {
			out = append(out, e.Name())
		}
	}
	return out
}

var _ = Describe("Delete on the pointer log, broken by a crash", func() {
	It("Should not keep the file of a segment that a durable Tombstone emptied after "+
		"a crash brings the file back", func(ctx SpecContext) {
		var failures []string
		for runKey := range uint64(48) {
			disk := MustSucceed(fault.New(fault.Config{}))
			db, closeDB := breakOpen(ctx, disk)
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key:      breakIdx,
					Name:     "time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      breakData,
					Name:     "data",
					Index:    breakIdx,
					DataType: telem.Int64T,
				},
			)).To(Succeed())
			breakWrite(ctx, db, 10, 11)
			breakWrite(ctx, db, 20, 21, 22)
			Expect(db.DeleteTimeRange(
				ctx,
				[]cesium.ChannelKey{breakData},
				(0 * telem.SecondTS).Range(15*telem.SecondTS),
			)).To(Succeed())
			Expect(breakSegments(disk, breakData)).To(HaveLen(1))
			image := MustSucceed(disk.Crash(fault.CrashConfig{
				RunKey:            runKey,
				EntrySurvivalRate: 0.5,
				DataSurvivalRate:  0.5,
			}))
			closeDB()
			re, closeRe := breakOpen(ctx, image)
			Expect(breakValues(ctx, re)).To(Equal([]int64{20, 21, 22}))
			// Run returns only after the orphan pass, and a read does not wait for it.
			closeRe()
			if segs := breakSegments(image, breakData); len(segs) != 1 {
				failures = append(failures, fmt.Sprintf(
					"run key %d: segment files %v after the orphan pass", runKey, segs,
				))
			}
		}
		Expect(failures).To(BeEmpty())
	})
})
