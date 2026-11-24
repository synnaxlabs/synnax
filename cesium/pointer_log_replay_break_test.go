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
	"bytes"
	"context"
	"math/rand/v2"
	"path"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// brkSerial runs the jobs it gets one at a time, in the order they came, on one
// goroutine.
type brkSerial struct {
	mu      sync.Mutex
	cond    *sync.Cond
	jobs    []func()
	stopped bool
	done    chan struct{}
}

func newBrkSerial() *brkSerial {
	s := &brkSerial{done: make(chan struct{})}
	s.cond = sync.NewCond(&s.mu)
	go func() {
		defer close(s.done)
		for {
			s.mu.Lock()
			for len(s.jobs) == 0 && !s.stopped {
				s.cond.Wait()
			}
			if len(s.jobs) == 0 {
				s.mu.Unlock()
				return
			}
			job := s.jobs[0]
			s.jobs = s.jobs[1:]
			s.mu.Unlock()
			job()
		}
	}()
	return s
}

// stop ends the goroutine of s once its queue is empty.
func (s *brkSerial) stop() {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()
	s.cond.Signal()
	<-s.done
}

func (s *brkSerial) run(f func()) {
	s.mu.Lock()
	s.jobs = append(s.jobs, f)
	s.mu.Unlock()
	s.cond.Signal()
}

// brkFiles returns the bytes of every file under dir of fs, by path.
func brkFiles(fs xfs.FS, dir string, out map[string][]byte) {
	GinkgoHelper()
	for _, info := range MustSucceed(fs.List(dir)) {
		name := path.Join(dir, info.Name())
		if info.IsDir() {
			brkFiles(fs, name, out)
			continue
		}
		out[name] = brkReadFile(fs, name)
	}
}

var _ = Describe("Pointer log replay breaker", func() {
	// workload runs one fixed workload on a new database in fs and returns its files.
	workload := func(ctx context.Context) map[string][]byte {
		GinkgoHelper()
		fs := xfs.NewMem()
		serial := newBrkSerial()
		open := func() *cesium.DB {
			return MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(9, 9)),
				cesium.WithExecutor(serial.run),
			))
		}
		db := open()
		stop := StartRun(db.Run)
		idx, data, other := cesium.ChannelKey(
			1,
		), cesium.ChannelKey(
			2,
		), cesium.ChannelKey(
			3,
		)
		keys := []cesium.ChannelKey{idx, data, other}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
			cesium.Channel{
				Key:      other,
				Name:     "text",
				Index:    idx,
				DataType: telem.StringT,
			},
		)).To(Succeed())
		second := telem.TimeStamp(0)
		for range 3 {
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Clock:    idx,
				Start:    (second + 1) * telem.SecondTS,
			}))
			for i := range 40 {
				second++
				MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(second),
					telem.NewSeriesV(int64(second)),
					telem.NewSeriesV("x"),
				}), nil))
				MustSucceed2(w.Commit(ctx, nil))
				if i%3 == 0 {
					Expect(w.Sync(ctx)).To(Succeed())
				}
				if i%17 == 0 {
					Expect(w.EndStretch(ctx)).To(Succeed())
				}
			}
			Expect(w.Close()).To(Succeed())
		}
		Expect(db.Intend([]cesium.ChannelKey{other})).To(Succeed())
		Expect(db.Drop([]cesium.ChannelKey{other})).To(Succeed())
		stop()
		Expect(db.Close()).To(Succeed())
		db = open()
		stop = StartRun(db.Run)
		stop()
		Expect(db.Close()).To(Succeed())
		serial.stop()
		files := map[string][]byte{}
		brkFiles(fs, "", files)
		return files
	}

	It("Should write the same bytes to every file in two runs of one workload", func(
		ctx SpecContext,
	) {
		first, second := workload(ctx), workload(ctx)
		Expect(second).To(HaveLen(len(first)))
		for name, b := range first {
			Expect(second).To(HaveKey(name))
			Expect(bytes.Equal(second[name], b)).To(BeTrue(), "file %s differs", name)
		}
	})
})
