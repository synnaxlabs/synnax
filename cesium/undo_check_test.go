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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: the files of an undo through the next Sweep", func() {
	const (
		idx  cesium.ChannelKey = 1
		data cesium.ChannelKey = 2
	)
	var (
		keys     = []cesium.ChannelKey{idx, data}
		channels = []cesium.Channel{
			{Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT},
			{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		}
		fs xfs.FS
		db *cesium.DB
		// closeDB stops and closes db.
		closeDB func() error
		// undone holds the undone files after the undo.
		undone []string
	)
	// open opens fs on the pointer log with no identity held, which undoes a database
	// in it, and with the random source rand.NewPCG(n, n).
	open := func(ctx context.Context, n uint64) {
		GinkgoHelper()
		db = MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithIdentity(0),
			cesium.WithRandom(rand.NewPCG(n, n)),
		))
		stop := GoRun(ctx, db.Run)
		closeDB = func() error { return errors.Join(stop(), db.Close()) }
	}
	// The database makes two channels on the pointer log and writes to them. The next
	// open holds no identity, so it undoes the database and starts an empty one.
	BeforeEach(func(ctx SpecContext) {
		fs = xfs.NewMem()
		open(ctx, 1)
		id := db.Identity()
		Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries([]telem.TimeStamp{telem.SecondTS, 2 * telem.SecondTS}),
			telem.NewSeries([]int64{1, 2}),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		Expect(closeDB()).To(Succeed())
		open(ctx, 2)
		undone = MustSucceed(filesOf(fs, isUndone))
		Expect(undone).To(ContainElements(
			"1/channel"+undoneSuffix(id), "2/channel"+undoneSuffix(id),
		), "the premise: the undo moved the channel files aside")
		DeferCleanup(func() { Expect(closeDB()).To(Succeed()) })
	})

	It("Should keep the undone files of each channel through the first Sweep",
		func(ctx SpecContext) {
			MustSucceed2(db.Sweep(ctx, nil))
			Expect(filesOf(fs, isUndone)).To(Equal(undone))
		})
})
