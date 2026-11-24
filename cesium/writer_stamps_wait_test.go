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
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// failingSyncFS fails each Sync of a file under dir with EIO while failing holds.
type failingSyncFS struct {
	xfs.FS
	dir     string
	failing *atomic.Bool
}

func (f failingSyncFS) Sub(name string) (xfs.FS, error) {
	if name == "" {
		return f, nil
	}
	return f.FS.Sub(name)
}

func (f failingSyncFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasPrefix(name, f.dir+"/") {
		return file, err
	}
	return failingSyncFile{File: file, failing: f.failing}, nil
}

type failingSyncFile struct {
	xfs.File
	failing *atomic.Bool
}

func (f failingSyncFile) Sync() error {
	if f.failing.Load() {
		return syscall.EIO
	}
	return f.File.Sync()
}

var _ = Describe("Data against stamps that are not durable yet", func() {
	var (
		fs        xfs.FS
		failing   *atomic.Bool
		db        *cesium.DB
		stop      func()
		idx, data cesium.ChannelKey
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		db = MustOpen(cesium.Open(ctx, "", cesium.WithFS(fs), cesium.WithPointerLog()))
		stop = StartRun(db.Run)
	}
	BeforeEach(func(ctx SpecContext) {
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		failing = &atomic.Bool{}
		fs = failingSyncFS{
			FS:      xfs.NewMem(),
			dir:     strconv.Itoa(int(idx)),
			failing: failing,
		}
		open(ctx)
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
	})
	// stamps opens a writer of the clock and commits stamps at seconds, with no sync.
	stamps := func(ctx context.Context, seconds ...telem.TimeStamp) *cesium.Writer {
		GinkgoHelper()
		c := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Clock:    idx,
			Channels: []cesium.ChannelKey{idx},
			Start:    seconds[0] * telem.SecondTS,
		}))
		MustSucceed2(c.Write(ctx, telem.UnaryFrame(
			idx,
			telem.NewSeriesSecondsTSV(seconds...),
		), nil))
		MustSucceed2(c.Commit(ctx, nil))
		return c
	}
	// values opens a writer of data from start and commits v, with no sync.
	values := func(
		ctx context.Context,
		start telem.TimeStamp,
		v ...int64,
	) *cesium.Writer {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Clock:    idx,
			Channels: []cesium.ChannelKey{data},
			Start:    start * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.UnaryFrame(data, telem.NewSeriesV(v...)), nil))
		MustSucceed2(w.Commit(ctx, nil))
		return w
	}
	read := func(ctx context.Context, key cesium.ChannelKey) []telem.Series {
		GinkgoHelper()
		return MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).Get(key).Series
	}

	It("Should end a Sync with its context while the clock is not synced, and sync "+
		"once it is", func(ctx SpecContext) {
		c := stamps(ctx, 10, 11)
		w := values(ctx, 10, 1, 2)
		deadline, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		Expect(w.Sync(deadline)).To(SatisfyAll(
			MatchError(context.DeadlineExceeded),
			MatchError(ContainSubstring(
				"clock "+strconv.Itoa(int(idx))+" has not made the stamps from",
			)),
		))
		Expect(c.Sync(ctx)).To(Succeed())
		Expect(
			w.Sync(ctx),
		).To(Succeed(), "the writer stays usable after its context ends")
		Expect(w.Close()).To(Succeed())
		Expect(c.Close()).To(Succeed())
		stop()
		Expect(db.Close()).To(Succeed())
		open(ctx)
		stored := read(ctx, data)
		Expect(stored).To(HaveLen(1))
		Expect(stored[0]).To(telem.MatchSeriesDataV[int64](1, 2))
	})

	It("Should end a Sync when the clock's sync succeeds while it waits", func(
		ctx SpecContext,
	) {
		c := stamps(ctx, 10, 11)
		w := values(ctx, 10, 1, 2)
		synced := make(chan error, 1)
		go func() { synced <- w.Sync(ctx) }()
		Consistently(synced, 100*time.Millisecond).ShouldNot(Receive())
		Expect(c.Sync(ctx)).To(Succeed())
		Eventually(synced).Should(Receive(Succeed()))
		Expect(w.Close()).To(Succeed())
		Expect(c.Close()).To(Succeed())
	})

	It("Should take back the data when a failed sync of the clock takes back its "+
		"stamps", func(ctx SpecContext) {
		failing.Store(true)
		c := stamps(ctx, 10, 11)
		w := values(ctx, 10, 1, 2)
		synced := make(chan error, 1)
		go func() { synced <- w.Sync(ctx) }()
		Expect(c.Sync(ctx)).To(MatchError(syscall.EIO))
		pulled := ContainSubstring("clock " + strconv.Itoa(int(idx)) +
			" took back the stamps from")
		Eventually(synced).Should(Receive(MatchError(pulled)))
		Expect(read(ctx, data)).To(BeEmpty())
		failing.Store(false)
		Expect(w.Commit(ctx, nil)).Error().To(MatchError(pulled))
		Expect(w.Close()).To(MatchError(pulled))
		Expect(c.Close()).To(MatchError(syscall.EIO))
		Expect(read(ctx, idx)).To(BeEmpty())
	})

	It("Should take back the data of a pinning writer that has not synced yet", func(
		ctx SpecContext,
	) {
		failing.Store(true)
		c := stamps(ctx, 10, 11)
		w := values(ctx, 10, 1, 2)
		Expect(c.Sync(ctx)).To(MatchError(syscall.EIO))
		Expect(read(ctx, data)).To(BeEmpty(), "the data leaves view in the same step")
		failing.Store(false)
		Expect(c.Close()).To(MatchError(syscall.EIO))
		later := stamps(ctx, 20, 21)
		Expect(later.Sync(ctx)).To(Succeed())
		MustSucceed2(
			w.Write(ctx, telem.UnaryFrame(data, telem.NewSeriesV[int64](3, 4)), nil),
		)
		pulled := ContainSubstring("took back the stamps")
		Expect(w.Commit(ctx, nil)).Error().To(MatchError(pulled),
			"a commit on stamps that a later writer stored still fails")
		Expect(w.Close()).To(MatchError(pulled))
		Expect(later.Close()).To(Succeed())
	})

	It("Should keep the data of a pinning writer on a durable stretch when the clock "+
		"takes back another stretch", func(ctx SpecContext) {
		c := stamps(ctx, 10, 11)
		Expect(c.Sync(ctx)).To(Succeed())
		Expect(c.Close()).To(Succeed())
		w := values(ctx, 10, 1, 2)
		failing.Store(true)
		later := stamps(ctx, 20, 21)
		Expect(later.Sync(ctx)).To(MatchError(syscall.EIO))
		failing.Store(false)
		Expect(w.Sync(ctx)).To(Succeed())
		Expect(w.Close()).To(Succeed())
		Expect(later.Close()).To(MatchError(syscall.EIO))
		stored := read(ctx, data)
		Expect(stored).To(HaveLen(1))
		Expect(stored[0]).To(telem.MatchSeriesDataV[int64](1, 2))
	})

	It("Should not wait at Close, and take back the data whose stamps are not durable",
		func(ctx SpecContext) {
			c := stamps(ctx, 10, 11)
			w := values(ctx, 10, 1, 2)
			Expect(
				w.Close(),
			).To(MatchError(ContainSubstring("has not made the stamps")))
			Expect(read(ctx, data)).To(BeEmpty())
			Expect(c.Close()).To(Succeed())
			Expect(read(ctx, idx)).To(HaveLen(1))
		})

	It("Should keep the data of a pinning writer whose stamps are durable", func(
		ctx SpecContext,
	) {
		c := stamps(ctx, 10, 11)
		Expect(c.Sync(ctx)).To(Succeed())
		w := values(ctx, 10, 1, 2)
		failing.Store(true)
		MustSucceed2(c.Write(ctx, telem.UnaryFrame(
			idx,
			telem.NewSeriesSecondsTSV(12, 13),
		), nil))
		MustSucceed2(c.Commit(ctx, nil))
		Expect(c.Sync(ctx)).To(MatchError(syscall.EIO))
		failing.Store(false)
		Expect(w.Sync(ctx)).To(Succeed())
		Expect(w.Close()).To(Succeed())
		Expect(c.Close()).To(MatchError(syscall.EIO))
		stored := read(ctx, data)
		Expect(stored).To(HaveLen(1))
		Expect(stored[0]).To(telem.MatchSeriesDataV[int64](1, 2))
	})
})
