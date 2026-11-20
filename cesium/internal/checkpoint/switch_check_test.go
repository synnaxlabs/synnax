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
	var l *log.Log
	st := MustSucceed(state.Open(state.Config{
		Versions: set.New[uint16](1),
		Version:  1,
		Append:   func(e *log.Entry) { l.Append(e) },
		Executor: exec,
	}, &state.Snapshot{Limits: state.Limits{Segments: 1, Alignments: 1}}))
	cp := MustSucceed(checkpoint.New(checkpoint.Config{
		FS:       fs,
		DB:       db,
		LogSize:  logSize,
		Snapshot: st.Snapshot,
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
		Seals: func() ([]log.Record, error) {
			seals, err := st.SealUnheld()
			records := make([]log.Record, len(seals))
			for i := range seals {
				records[i] = &seals[i]
			}
			return records, err
		},
		Switched:   cp.Switched,
		DB:         db,
		Generation: 1,
		Sequence:   1,
	}))
	go func() { _ = st.Run(ctx) }()
	go func() { _ = l.Run(ctx) }()
	go func() { _ = cp.Run(ctx) }()
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

var _ = Describe("Switch check", func() {
	It("Should switch logs when the executor runs one job at a time", func() {
		jobs := make(chan func(), 64)
		go func() {
			for f := range jobs {
				f()
			}
		}()
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		l, _ := owners(ctx, xfs.NewMem(), func(f func()) { jobs <- f })
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
		}).Should(Succeed())
		f := MustSucceed(fs.Open(log.Name(2), os.O_RDONLY))
		b := make([]byte, log.HeaderSize)
		MustSucceed(f.ReadAt(b, 0))
		Expect(f.Close()).To(Succeed())
		lh := MustSucceed(log.DecodeHeader(b, db, 2, 0))
		// No segment is open, so log 2 starts with no Seals entry.
		Expect(h.Sequence).To(Equal(lh.Sequence - 1))
	})

	It("Should write a head at a switch that covers no entry of the next log, when "+
		"the next log starts with a Seals entry", func() {
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		fs := xfs.NewMem()
		l, st := owners(ctx, fs, func(f func()) { go f() })
		// An open segment that no writer holds gets its Seal first in the next log.
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
		}).Should(Succeed())
		f := MustSucceed(fs.Open(log.Name(2), os.O_RDONLY))
		b := make([]byte, log.FramesStart+log.MaxFrame)
		n, _ := f.ReadAt(b, 0)
		Expect(f.Close()).To(Succeed())
		b = b[:n]
		lh := MustSucceed(log.DecodeHeader(b, db, 2, 0))
		first := MustSucceed(
			log.DecodeFrame(b[log.FramesStart:], db, 2, log.FramesStart),
		)
		var d log.Decoder
		MustSucceed(d.Reset(first.Entries))
		r, ok := MustSucceed2(d.Next())
		Expect(ok).To(BeTrue())
		Expect(r).To(BeAssignableToTypeOf(&log.Seal{}))
		Expect(first.Sequence).To(Equal(lh.Sequence))
		Expect(h.Sequence).To(Equal(lh.Sequence - 1))
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
			var err error
			Expect(func() { _, _, err = checkpoint.Decode(b) }).ToNot(Panic())
			Expect(err).To(MatchError(place.ErrCorrupt))
		},
	)
})
