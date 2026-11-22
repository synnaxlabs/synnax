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
	"os"
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

var _ = Describe("Orphan pass", func() {
	It("Should end before a Sweep removes the directory of a torn key", func(
		ctx SpecContext,
	) {
		open := func(fs xfs.FS) *cesium.DB {
			GinkgoHelper()
			return MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs),
				cesium.WithPointerLog(),
			))
		}
		disk := xfs.NewMem()
		db := open(disk)
		stop := GoRun(ctx, db.Run)
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key:      1,
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		})).To(Succeed())
		Expect(stop()).To(Succeed())
		Expect(db.Close()).To(Succeed())
		// A channel file that fails its check leaves key 1, with no segment, torn.
		f := MustSucceed(disk.Open("1/channel", os.O_RDWR))
		MustSucceed(f.WriteAt([]byte{0xFF}, 8))
		Expect(f.Close()).To(Succeed())
		gate := orphanGateFS{
			FS:      disk,
			armed:   &atomic.Bool{},
			held:    make(chan struct{}),
			release: make(chan struct{}),
		}
		release := sync.OnceFunc(func() { close(gate.release) })
		DeferCleanup(release)
		db = open(gate)
		gate.armed.Store(true)
		stop = GoRun(ctx, db.Run)
		Eventually(gate.held).WithTimeout(10 * time.Second).Should(BeClosed())
		swept := make(chan error, 1)
		go func() {
			_, _, err := db.Sweep(ctx, nil)
			swept <- err
		}()
		Consistently(swept).WithTimeout(200 * time.Millisecond).ShouldNot(Receive())
		release()
		Eventually(swept).WithTimeout(10 * time.Second).Should(Receive(Succeed()))
		Expect(stop()).To(Succeed())
		Expect(db.Close()).To(Succeed())
		Expect(disk.Exists("1")).To(BeFalse())
	})
})
