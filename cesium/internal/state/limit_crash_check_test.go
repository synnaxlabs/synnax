// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	"context"
	"io"
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// imageFS takes a crash image of disk after each write and each sync of a file that it
// opens.
type imageFS struct {
	xfs.FS
	take func()
}

func (f imageFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return imageFile{File: file, take: f.take}, nil
}

type imageFile struct {
	xfs.File
	take func()
}

func (f imageFile) WriteAt(p []byte, off int64) (int, error) {
	defer f.take()
	return f.File.WriteAt(p, off)
}

func (f imageFile) Sync() error {
	defer f.take()
	return f.File.Sync()
}

// replay replays the valid frames of log 1 in image into a new state on the head h,
// and returns the first refusal.
func replay(image *fault.FS, h *state.Snapshot) error {
	GinkgoHelper()
	f := MustSucceed(image.Open(log.Name(1), os.O_RDONLY))
	b := MustSucceed(io.ReadAll(f))
	Expect(f.Close()).To(Succeed())
	s := MustSucceed(state.Open(config(&appended{}), h))
	at := int64(log.FramesStart)
	for at+log.FrameHeaderSize <= int64(len(b)) {
		fr, err := log.DecodeFrame(b[at:], 7, 1, uint64(at))
		if err != nil {
			aligned := (at + log.BlockSize - 1) &^ (log.BlockSize - 1)
			if aligned == at || aligned+log.FrameHeaderSize > int64(len(b)) {
				return nil
			}
			at = aligned
			if fr, err = log.DecodeFrame(b[at:], 7, 1, uint64(at)); err != nil {
				return nil
			}
		}
		seq := fr.Sequence
		for off := 0; off < len(fr.Entries); seq++ {
			var d log.Decoder
			n := MustSucceed(d.Reset(fr.Entries[off:]))
			var records []log.Record
			for {
				r, ok := MustSucceed2(d.Next())
				if !ok {
					break
				}
				records = append(records, r)
			}
			if err := s.Replay(seq, records); err != nil {
				return err
			}
			off += n
		}
		at += int64(fr.Size())
	}
	return nil
}

var _ = Describe("Check: a Reserve at each crash point", func() {
	It("Should replay with no value above a durable Reserve limit at each crash point",
		func() {
			const size = 4 << 20
			for runKey := range uint64(4) {
				disk := MustSucceed(fault.New(fault.Config{}))
				Expect(log.Prepare(disk, 1, size)).To(Succeed())
				var (
					mu     sync.Mutex
					images []*fault.FS
					l      *log.Log
					goExe  = func(f func()) { go f() }
				)
				take := func() {
					mu.Lock()
					defer mu.Unlock()
					images = append(images, MustSucceed(disk.Crash(fault.CrashConfig{
						RunKey:            runKey + uint64(len(images)),
						EntrySurvivalRate: 0.5,
						DataSurvivalRate:  0.5,
					})))
				}
				st := MustSucceed(state.Open(state.Config{
					Versions: config(&appended{}).Versions,
					Version:  1,
					Append:   func(e *log.Entry) { l.Append(e) },
					Executor: goExe,
					OnHead:   func(uint64, *state.Snapshot) {},
				}, head(1, 1)))
				l = MustSucceed(log.Start(log.Config{
					FS:       imageFS{FS: disk, take: take},
					Executor: goExe,
					OnDurable: func(seq uint64, records []log.Record) {
						_ = st.Durable(seq, records)
					},
					OnRefused: func(records []log.Record, err error) {
						_ = st.Refused(records, err)
					},
					// No next file, so every entry stays in log 1.
					Prepare: func(next uint64) {
						go l.Prepared(next, errors.New("no next log"))
					},
					Switched:   func(g uint64) { _ = st.Switch(g) },
					DB:         7,
					Generation: 1,
					Sequence:   1,
				}))
				ctx, cancel := context.WithCancel(context.Background())
				var wg sync.WaitGroup
				wg.Go(func() { _ = st.Run(ctx) })
				wg.Go(func() { _ = l.Run(ctx) })
				// 40 identities cross the first Reserve block of 16 and the next of
				// 1,024.
				keys := make([]state.WriterKey, 40)
				for i := range keys {
					keys[i] = state.WriterKey{Key: uint32(5 + i), DataType: 4}
				}
				w := MustSucceed(st.OpenWriter(state.WriterConfig{
					Keys: keys, Clock: 5, Start: 10,
				}))
				for _, k := range keys {
					id := MustSucceed(w.NewSegment(k.Key))
					var e log.Entry
					e.Add(&log.Create{
						Segment: id, Key: k.Key, Clock: 5, Version: 1, DataType: 4,
					})
					MustSucceed(l.Append(&e).Wait())
				}
				Expect(w.Close()).To(Succeed())
				Expect(l.Close()).To(Succeed())
				Expect(st.Close()).To(Succeed())
				cancel()
				wg.Wait()
				mu.Lock()
				Expect(len(images)).To(BeNumerically(">", 40))
				for i, image := range images {
					Expect(replay(image, head(1, 1))).To(
						Succeed(),
						"run key %d, crash image %d of %d",
						runKey,
						i,
						len(images),
					)
				}
				mu.Unlock()
			}
		},
	)
})
