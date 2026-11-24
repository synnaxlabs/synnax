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
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = DescribeTable(
	"Check: a handoff on the pointer log to a start in another writer's stored data",
	func(ctx SpecContext, run cesium.HandoffRun) {
		const idx, data = cesium.ChannelKey(1), cesium.ChannelKey(2)
		keys := []cesium.ChannelKey{idx, data}
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(xfs.NewMem()),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithPointerLog(),
		))
		stop := StartRun(db.Run)
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
		write := func(w *cesium.Writer, seconds ...telem.TimeStamp) {
			GinkgoHelper()
			stamps := make([]telem.TimeStamp, len(seconds))
			values := make([]int64, len(seconds))
			for i, s := range seconds {
				stamps[i], values[i] = s*telem.SecondTS, int64(s)
			}
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeries(stamps),
				telem.NewSeries(values),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
		}
		open := func(start telem.TimeStamp) cesium.WriterConfig {
			return cesium.WriterConfig{
				Channels: keys,
				Clock:    idx,
				Start:    start * telem.SecondTS,
			}
		}
		other := MustSucceed(db.OpenWriter(ctx, open(30)))
		write(other, 30, 31)
		Expect(other.Close()).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, open(10)))
		write(w, 10, 11, 12)
		// The new writer would start inside the data of the other writer, as an open
		// of it refuses.
		Expect(db.Handoff(ctx, w, open(30), nil, run)).Error().To(MatchError(
			ContainSubstring(
				"cannot open writer because there is already data in the " +
					"writer's time range",
			),
		))
		stop()
		Expect(db.Close()).To(Succeed())
	},
	Entry("in a new run", cesium.NewRun),
	Entry("in the same run", cesium.SameRun),
)
