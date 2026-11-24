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
	"path"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// orphanGateFS holds the first List of the directory of key 1 after armed is set, until
// release is closed. The orphan pass lists that directory after it lists the root.
type orphanGateFS struct {
	xfs.FS
	dir     string
	armed   *atomic.Bool
	held    chan struct{}
	release chan struct{}
}

func (f orphanGateFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	f.FS, f.dir = sub, path.Join(f.dir, name)
	return f, nil
}

func (f orphanGateFS) List(name string) ([]xfs.FileInfo, error) {
	if path.Join(f.dir, name) == "1" && f.armed.CompareAndSwap(true, false) {
		close(f.held)
		<-f.release
	}
	return f.FS.List(name)
}

var _ = Describe("Check: a Drop during the orphan pass", func() {
	It("Should not fail Run when the Drop removes a directory that the pass listed",
		func(ctx SpecContext) {
			open := func(fs xfs.FS) *cesium.DB {
				GinkgoHelper()
				return MustSucceed(cesium.Open(
					ctx,
					"",
					cesium.WithFS(fs),
					cesium.WithPointerLog(),
					cesium.WithRandom(rand.NewPCG(1, 2)),
				))
			}
			disk := MustSucceed(fault.New(fault.Config{}))
			keys := []cesium.ChannelKey{1, 2, 3}
			db := open(disk)
			stop := GoRun(ctx, db.Run)
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{Key: 2, Name: "kept", Index: 1, DataType: telem.Int64T},
				cesium.Channel{
					Key: 3, Name: "dropped", Index: 1, DataType: telem.Int64T,
				},
			)).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Clock:    1,
				Start:    10 * telem.SecondTS,
			}))
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(10, 11),
				telem.NewSeriesV[int64](1, 2),
				telem.NewSeriesV[int64](3, 4),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
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
			re := open(gate)
			gate.armed.Store(true)
			stop = GoRun(ctx, re.Run)
			Eventually(gate.held).WithTimeout(10 * time.Second).Should(BeClosed())
			dropped := make(chan error, 1)
			go func() { dropped <- re.Drop(ctx, []cesium.ChannelKey{3}) }()
			// A Drop that waits for the orphan pass returns only after the release.
			select {
			case err := <-dropped:
				release()
				Expect(err).ToNot(HaveOccurred())
			case <-time.After(time.Second):
				release()
				Eventually(dropped).WithTimeout(10 * time.Second).
					Should(Receive(Succeed()))
			}
			Expect(stop()).To(Succeed())
			Expect(re.Close()).To(Succeed())
			Expect(MustSucceed(disk.Exists("3"))).To(BeFalse())
		})
})
