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
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Orphan pass before a delete", func() {
	It("Should end before a DeleteTimeRange unlinks the file of a segment that it "+
		"buries", func(ctx SpecContext) {
		// open opens fs stating identity id, 0 for a new root.
		open := func(fs xfs.FS, id uint64) *cesium.DB {
			GinkgoHelper()
			return MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs),
				cesium.WithPointerLog(),
				cesium.WithIdentity(id),
			))
		}
		keys := []cesium.ChannelKey{1, 2}
		// write writes 5 samples from start seconds in one writer, which seals one
		// segment of each key at its close.
		write := func(ctx context.Context, db *cesium.DB, start telem.TimeStamp) {
			GinkgoHelper()
			stamps := make([]telem.TimeStamp, 5)
			values := make([]int64, 5)
			for i := range stamps {
				stamps[i] = (start + telem.TimeStamp(i)) * telem.SecondTS
				values[i] = int64(start) + int64(i)
			}
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Clock:    keys[0],
				Start:    stamps[0],
			}))
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeries(stamps),
				telem.NewSeries(values),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
		}
		disk := xfs.NewMem()
		db := open(disk, 0)
		stop := GoRun(ctx, db.Run)
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      1,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{Key: 2, Name: "data", Index: 1, DataType: telem.Int64T},
		)).To(Succeed())
		write(ctx, db, 1)
		write(ctx, db, 10)
		id := db.Identity()
		Expect(stop()).To(Succeed())
		Expect(db.Close()).To(Succeed())
		gate := orphanGateFS{
			FS:      disk,
			armed:   &atomic.Bool{},
			held:    make(chan struct{}),
			release: make(chan struct{}),
		}
		release := sync.OnceFunc(func() { close(gate.release) })
		DeferCleanup(release)
		db = open(gate, id)
		gate.armed.Store(true)
		stop = GoRun(ctx, db.Run)
		Eventually(gate.held).WithTimeout(10 * time.Second).Should(BeClosed())
		deleted := make(chan error, 1)
		go func() {
			deleted <- db.DeleteTimeRange(
				ctx,
				[]cesium.ChannelKey{2},
				telem.TimeRange{Start: telem.SecondTS, End: 6 * telem.SecondTS},
			)
		}()
		Consistently(deleted).WithTimeout(200 * time.Millisecond).ShouldNot(Receive())
		release()
		Eventually(deleted).WithTimeout(10 * time.Second).Should(Receive(Succeed()))
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, 2))
		Expect(fr.Get(2).Series).To(HaveLen(1))
		Expect(fr.Get(2).Series[0].Data).To(Equal(
			telem.NewSeriesV[int64](10, 11, 12, 13, 14).Data,
		))
		Expect(stop()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
})
