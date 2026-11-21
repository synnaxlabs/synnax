// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	"context"
	"encoding/binary"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// owners runs the log, the state, and the checkpointer of fs on exec, as the engine
// wires them, and returns the log and the state.
func owners(
	ctx context.Context,
	fs xfs.FS,
	exec func(func()),
) (*log.Log, *state.State) {
	GinkgoHelper()
	Expect(log.Prepare(fs, 1, logSize)).To(Succeed())
	Expect(checkpoint.Write(fs, 1, checkpoint.Encode(db, 1, &state.Snapshot{
		Limits: state.Limits{Segments: 1, Alignments: 1},
	}))).To(Succeed())
	var (
		l  *log.Log
		cp *checkpoint.Checkpointer
	)
	st := MustSucceed(state.Open(state.Config{
		Versions: set.New[uint16](1),
		Version:  1,
		Append:   func(e *log.Entry) { l.Append(e) },
		Executor: exec,
		OnHead:   func(g uint64, s *state.Snapshot) { cp.Head(g, s) },
	}, &state.Snapshot{Limits: state.Limits{Segments: 1, Alignments: 1}}))
	cp = MustSucceed(checkpoint.New(checkpoint.Config{
		FS:       fs,
		DB:       db,
		LogSize:  logSize,
		Prepared: func(next uint64, err error) { l.Prepared(next, err) },
		Executor: exec,
	}))
	l = MustSucceed(log.Start(log.Config{
		FS:       fs,
		Executor: exec,
		OnDurable: func(sequence uint64, records []log.Record) {
			_ = st.Durable(sequence, records)
		},
		Prepare: cp.Prepare,
		OnRefused: func(records []log.Record, err error) {
			_ = st.Refused(records, err)
		},
		Switched:   func(g uint64) { _ = st.Switch(g) },
		DB:         db,
		Generation: 1,
		Sequence:   1,
	}))
	go func() { _ = st.Run(ctx) }()
	go func() { _ = l.Run(ctx) }()
	go func() { _ = cp.Run(ctx) }()
	// The log switches only once the file of the next log exists.
	Eventually(func() bool { return MustSucceed(fs.Exists(log.Name(2))) }).
		WithTimeout(10 * time.Second).
		Should(BeTrue())
	return l, st
}

// fill appends 12 entries of 10,000 intents to l, which passes three quarters of a log
// of logSize.
func fill(l *log.Log) chan error {
	done := make(chan error, 1)
	go func() {
		for i := range uint32(12) {
			var e log.Entry
			for k := range uint32(10_000) {
				e.Add(&log.Intent{Key: 1_000_000 + i*10_000 + k})
			}
			if _, err := l.Append(&e).Wait(); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	return done
}

// sealSequence returns the sequence of the first entry of the log file b, of
// generation 2, that holds a Seal, or 0. It reads each frame at the end of the one
// before it, or at the next block boundary.
func sealSequence(b []byte) uint64 {
	GinkgoHelper()
	at := int64(log.FramesStart)
	for at+log.FrameHeaderSize <= int64(len(b)) {
		f, err := log.DecodeFrame(b[at:], db, 2, uint64(at))
		if err != nil {
			aligned := (at + log.BlockSize - 1) &^ (log.BlockSize - 1)
			if aligned == at || aligned+log.FrameHeaderSize > int64(len(b)) {
				return 0
			}
			at = aligned
			if f, err = log.DecodeFrame(b[at:], db, 2, uint64(at)); err != nil {
				return 0
			}
		}
		seq := f.Sequence
		for off := 0; off < len(f.Entries); seq++ {
			var d log.Decoder
			n := MustSucceed(d.Reset(f.Entries[off:]))
			for {
				r, ok := MustSucceed2(d.Next())
				if !ok {
					break
				}
				if _, sealed := r.(*log.Seal); sealed {
					return seq
				}
			}
			off += n
		}
		at += int64(f.Size())
	}
	return 0
}

var _ = Describe("Switch check", func() {
	It("Should switch logs when the executor runs one job at a time", func() {
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		l, _ := owners(ctx, xfs.NewMem(), executor(true))
		Eventually(fill(l)).WithTimeout(10 * time.Second).Should(Receive(BeNil()))
	})

	It("Should write a head that covers exactly the entries before the switch", func() {
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		fs := xfs.NewMem()
		l, _ := owners(ctx, fs, func(f func()) { go f() })
		Eventually(fill(l)).WithTimeout(10 * time.Second).Should(Receive(BeNil()))
		var h checkpoint.Header
		Eventually(func() error {
			var err error
			h, _, err = checkpoint.Read(fs, 2)
			return err
		}).WithTimeout(10 * time.Second).Should(Succeed())
		f := MustSucceed(fs.Open(log.Name(2), os.O_RDONLY))
		b := make([]byte, log.HeaderSize)
		MustSucceed(f.ReadAt(b, 0))
		Expect(f.Close()).To(Succeed())
		lh := MustSucceed(log.DecodeHeader(b, db, 2, 0))
		// No segment is open, so log 2 starts with no Seals entry.
		Expect(h.Sequence).To(Equal(lh.Sequence - 1))
	})

	It("Should write a head at a switch that covers no entry of the next log, when "+
		"the next log holds a Seals entry", func() {
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		fs := xfs.NewMem()
		l, st := owners(ctx, fs, func(f func()) { go f() })
		// An open segment that no writer holds gets a Seal in the next log.
		w := MustSucceed(st.OpenWriter(state.WriterConfig{
			Keys: []state.WriterKey{{Key: 5, DataType: 4}}, Clock: 5, Start: 10,
		}))
		id := MustSucceed(w.NewSegment(5))
		var e log.Entry
		e.Add(&log.Create{Segment: id, Key: 5, Clock: 5, Version: 1, DataType: 4})
		MustSucceed(l.Append(&e).Wait())
		Expect(w.Close()).To(Succeed())
		Eventually(fill(l)).WithTimeout(10 * time.Second).Should(Receive(BeNil()))
		var h checkpoint.Header
		Eventually(func() error {
			var err error
			h, _, err = checkpoint.Read(fs, 2)
			return err
		}).WithTimeout(10 * time.Second).Should(Succeed())
		f := MustSucceed(fs.Open(log.Name(2), os.O_RDONLY))
		b := make([]byte, log.FramesStart+log.MaxFrame)
		n, _ := f.ReadAt(b, 0)
		Expect(f.Close()).To(Succeed())
		b = b[:n]
		lh := MustSucceed(log.DecodeHeader(b, db, 2, 0))
		sealed := sealSequence(b)
		Expect(sealed).ToNot(BeZero(), "log 2 holds no Seal of the open segment")
		Expect(h.Sequence).To(Equal(lh.Sequence - 1))
		Expect(sealed).To(BeNumerically(">", h.Sequence))
	})

	It("Should refuse a head whose chunk table offset wraps past 2^64, with no panic",
		func() {
			b := make([]byte, 4320)
			copy(b, "CSCHKPT\x00")
			binary.LittleEndian.PutUint16(b[8:], 1)
			b[10] = 4
			binary.LittleEndian.PutUint64(b[16:], 7)
			binary.LittleEndian.PutUint64(b[24:], 3)
			chunks := uint64(18445618242517995996)
			binary.LittleEndian.PutUint64(b[64:], 4320)
			binary.LittleEndian.PutUint64(b[72:], chunks-216)
			binary.LittleEndian.PutUint32(b[80:], 9)
			binary.LittleEndian.PutUint64(b[88:], chunks)
			binary.LittleEndian.PutUint32(b[508:], place.Head(7, 3, 0).Sum(b[:508]))
			fs := xfs.NewMem()
			f := MustSucceed(fs.Open(checkpoint.Name(3), os.O_CREATE|os.O_RDWR))
			MustSucceed(f.Write(b))
			Expect(f.Close()).To(Succeed())
			var err error
			Expect(func() { _, _, err = checkpoint.Read(fs, 3) }).ToNot(Panic())
			Expect(err).To(MatchError(place.ErrCorrupt))
		},
	)
})
