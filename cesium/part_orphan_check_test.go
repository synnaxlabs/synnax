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
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// headGateFS holds each list of the root by the orphan pass, once armed, until head 2
// passes its checks.
type headGateFS struct {
	xfs.FS
	armed *atomic.Bool
}

// byOrphanPass reports whether the orphan pass of the engine is a caller.
func byOrphanPass() bool {
	pcs := make([]uintptr, 32)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(2, pcs)])
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, ".(*engine).orphans") {
			return true
		}
		if !more {
			return false
		}
	}
}

// Sub keeps the gate on the FS that Open gives the engine, a Sub of its root.
func (f headGateFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return headGateFS{FS: sub, armed: f.armed}, nil
}

func (f headGateFS) List(name string) ([]xfs.FileInfo, error) {
	if name == "" && f.armed.Load() && byOrphanPass() {
		for deadline := time.Now().Add(time.Minute); time.Now().Before(deadline); {
			if _, _, err := checkpoint.Read(f.FS, 2); err == nil {
				break
			}
			time.Sleep(time.Millisecond)
		}
	}
	return f.FS.List(name)
}

var _ = Describe("Part orphan check", func() {
	It("Should keep the parts of a head that a switch writes during the orphan pass",
		func(ctx SpecContext) {
			fs := xfs.NewMem()
			db, closeDB := MustSucceed2(openLog(ctx, fs))
			const count = 1_000
			keys := make([]cesium.ChannelKey, count+1)
			channels := make([]cesium.Channel, count+1)
			keys[0] = 1
			channels[0] = cesium.Channel{
				Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			}
			for i := 1; i <= count; i++ {
				keys[i] = cesium.ChannelKey(i + 1)
				channels[i] = cesium.Channel{
					Key: keys[i], Name: "data", Index: 1, DataType: telem.Int64T,
				}
			}
			Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
			// Each writer starts a new alignment run, so its commit adds a pointer
			// record to each channel. 250 writers pass the part size of 8 MiB.
			for w := range telem.TimeStamp(250) {
				series := make([]telem.Series, count+1)
				series[0] = telem.NewSeriesSecondsTSV(w + 1)
				for i := 1; i <= count; i++ {
					series[i] = telem.NewSeriesV[int64](int64(w))
				}
				wr := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Clock:    keys[0],
					Start:    (w + 1) * telem.SecondTS,
				}))
				MustSucceed2(wr.Write(ctx, telem.MultiFrame(keys, series), nil))
				MustSucceed2(wr.Commit(ctx, nil))
				Expect(wr.Close()).To(Succeed())
			}
			Expect(closeDB()).To(Succeed())
			// An Intend does not wait for the orphan pass, so Intends switch the log
			// while the gate holds the pass. GoExecutor allows that order.
			armed := &atomic.Bool{}
			reopened := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(headGateFS{FS: fs, armed: armed}),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
				pcgIdentity(1, 2),
			))
			armed.Store(true)
			stop := GoRun(ctx, reopened.Run)
			// The switch to log 2 prepares log 3.
			for range 20_000 {
				if MustSucceed(fs.Exists(log.Name(3))) {
					break
				}
				Expect(reopened.Intend([]cesium.ChannelKey{999})).To(Succeed())
			}
			var h checkpoint.Header
			Eventually(func() error {
				var err error
				h, _, err = checkpoint.Read(fs, 2)
				return err
			}).WithTimeout(time.Minute).Should(Succeed())
			Expect(h.Parts).ToNot(BeEmpty(), "the switch wrote no part")
			// A writer waits for the orphan pass.
			w := MustSucceed(reopened.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys[:2],
				Clock:    keys[0],
				Start:    1_000 * telem.SecondTS,
			}))
			Expect(w.Close()).To(Succeed())
			Expect(stop()).To(Succeed())
			Expect(reopened.Close()).To(Succeed())
			_, closeAgain := MustSucceed2(openLog(ctx, fs))
			Expect(closeAgain()).To(Succeed())
		})
})
