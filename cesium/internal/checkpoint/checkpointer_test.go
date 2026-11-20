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
	"os"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

const logSize = 2 << 20

func goExec(f func()) { go f() }

// prepared records what a checkpointer reports to its Prepared hook.
type prepared struct {
	mu   sync.Mutex
	next []uint64
	errs []error
}

func (p *prepared) on(next uint64, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.next = append(p.next, next)
	p.errs = append(p.errs, err)
}

func (p *prepared) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.next)
}

// start runs c until the spec ends.
func start(c *checkpoint.Checkpointer) {
	GinkgoHelper()
	ctx, cancel := context.WithCancel(context.Background())
	ran := make(chan error, 1)
	go func() { ran <- c.Run(ctx) }()
	DeferCleanup(func() {
		err := c.Err()
		if err != nil {
			Expect(c.Close()).To(MatchError(err))
		} else {
			Expect(c.Close()).To(Succeed())
		}
		Eventually(ran).Should(Receive(BeNil()))
		cancel()
	})
}

var _ = Describe("Checkpointer", func() {
	DescribeTable(
		"Should refuse a config with a field that is not set",
		func(change func(*checkpoint.Config), field string) {
			cfg := checkpoint.Config{
				FS:      xfs.NewMem(),
				LogSize: logSize,
				Snapshot: func() (*state.Snapshot, error) {
					return &state.Snapshot{}, nil
				},
				Prepared: func(uint64, error) {},
				Executor: goExec,
			}
			change(&cfg)
			Expect(checkpoint.New(cfg)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(field)),
			))
		},
		Entry("file system", func(c *checkpoint.Config) { c.FS = nil }, "fs"),
		Entry("log size", func(c *checkpoint.Config) { c.LogSize = 0 }, "log_size"),
		Entry("snapshot", func(c *checkpoint.Config) { c.Snapshot = nil }, "snapshot"),
		Entry("prepared", func(c *checkpoint.Config) { c.Prepared = nil }, "prepared"),
		Entry("executor", func(c *checkpoint.Config) { c.Executor = nil }, "executor"),
	)

	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var (
				fs xfs.FS
				p  *prepared
				// snapErr holds the error of the next Snapshot, or nil.
				snapErr atomic.Pointer[error]
				c       *checkpoint.Checkpointer
			)
			BeforeEach(func() {
				fs, p = makeFS(), &prepared{}
				snapErr.Store(nil)
				c = MustSucceed(checkpoint.New(checkpoint.Config{
					FS:      fs,
					DB:      db,
					LogSize: logSize,
					Snapshot: func() (*state.Snapshot, error) {
						if err := snapErr.Load(); err != nil {
							return nil, *err
						}
						return sample(2), nil
					},
					Prepared: p.on,
					Executor: goExec,
				}))
				start(c)
			})

			exists := func(name string) bool {
				GinkgoHelper()
				return MustSucceed(fs.Exists(name))
			}

			It("Should make the file of the next log and report it", func() {
				f := MustSucceed(fs.Open(checkpoint.Name(4), os.O_CREATE|os.O_RDWR))
				Expect(f.Close()).To(Succeed())
				c.Prepare(4)
				Eventually(p.count).Should(Equal(1))
				Expect(p.next).To(Equal([]uint64{4}))
				Expect(p.errs).To(Equal([]error{nil}))
				Expect(
					MustSucceed(fs.Stat(log.Name(4))).Size(),
				).To(Equal(int64(logSize)))
				Expect(exists(checkpoint.Name(4))).To(BeFalse())
			})

			It(
				"Should write the head of a switched log and keep the older head",
				func() {
					for _, g := range []uint64{1, 2, 3} {
						Expect(log.Prepare(fs, g, logSize)).To(Succeed())
					}
					empty := checkpoint.Encode(db, 1, &state.Snapshot{Sequence: 1})
					Expect(checkpoint.Write(fs, 1, empty)).To(Succeed())
					c.Switched(2)
					Eventually(func() error { return readErr(fs, 2) }).Should(Succeed())
					_, s := MustSucceed2(checkpoint.Read(fs, 2))
					Expect(s.Sequence).To(Equal(sample(2).Sequence))
					Expect(exists(checkpoint.Name(1))).To(BeTrue())
					c.Switched(3)
					Eventually(func() bool { return exists(checkpoint.Name(1)) }).
						Should(BeFalse())
					Expect(exists(log.Name(1))).To(BeFalse())
					Expect(exists(checkpoint.Name(2))).To(BeTrue())
					Expect(c.Err()).ToNot(HaveOccurred())
				},
			)

			It("Should leave the older head and its logs when a head fails, until a "+
				"later head", func() {
				for _, g := range []uint64{1, 2, 3} {
					Expect(log.Prepare(fs, g, logSize)).To(Succeed())
				}
				empty := checkpoint.Encode(db, 1, &state.Snapshot{Sequence: 1})
				Expect(checkpoint.Write(fs, 1, empty)).To(Succeed())
				fault := errors.New("state failed")
				snapErr.Store(&fault)
				c.Switched(2)
				Eventually(c.Err).Should(MatchError(fault))
				Expect(exists(checkpoint.Name(2))).To(BeFalse())
				Expect(exists(log.Name(1))).To(BeTrue())
				snapErr.Store(nil)
				c.Switched(3)
				Eventually(func() error { return readErr(fs, 3) }).Should(Succeed())
				Eventually(c.Err).Should(Succeed())
				Expect(exists(checkpoint.Name(1))).To(BeTrue(), "the newest older head")
				Expect(exists(log.Name(1))).To(BeTrue())
			})

			It("Should report a full disk when heads fail with many logs", func() {
				for g := range uint64(checkpoint.MaxLogs + 1) {
					f := MustSucceed(fs.Open(log.Name(g+1), os.O_CREATE|os.O_RDWR))
					Expect(f.Close()).To(Succeed())
				}
				fault := errors.New("no head")
				snapErr.Store(&fault)
				c.Switched(18)
				Eventually(c.Err).Should(MatchError(checkpoint.ErrDiskFull))
			})
		})
	}

	It("Should switch a log of a running state and write a head of its durable state",
		func() {
			fs := xfs.NewMem()
			Expect(log.Prepare(fs, 1, logSize)).To(Succeed())
			Expect(checkpoint.Write(fs, 1, checkpoint.Encode(db, 1, &state.Snapshot{
				Limits: state.Limits{Segments: 1, Alignments: 1},
			}))).To(Succeed())
			var l *log.Log
			st := MustSucceed(state.Open(state.Config{
				Versions: set.New[uint16](1),
				Version:  1,
				Append:   func(e *log.Entry) { l.Append(e) },
				Executor: goExec,
			}, &state.Snapshot{Limits: state.Limits{Segments: 1, Alignments: 1}}))
			cp := MustSucceed(checkpoint.New(checkpoint.Config{
				FS:       fs,
				DB:       db,
				LogSize:  logSize,
				Snapshot: st.Snapshot,
				Prepared: func(next uint64, err error) { l.Prepared(next, err) },
				Executor: goExec,
			}))
			l = MustSucceed(log.Start(log.Config{
				FS:       fs,
				Executor: goExec,
				OnDurable: func(seq uint64, records []log.Record) {
					Expect(st.Durable(seq, records)).To(Succeed())
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
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)
			var wg sync.WaitGroup
			for _, run := range []func(context.Context) error{st.Run, l.Run, cp.Run} {
				wg.Go(func() { Expect(run(ctx)).To(Succeed()) })
			}
			DeferCleanup(func() {
				Expect(l.Close()).To(Succeed())
				Expect(cp.Close()).To(Succeed())
				Expect(st.Close()).To(Succeed())
				wg.Wait()
			})

			w := MustSucceed(st.OpenWriter(state.WriterConfig{
				Keys:  []state.WriterKey{{Key: 5, DataType: 4}},
				Clock: 5,
				Start: 10,
			}))
			id := MustSucceed(w.NewSegment(5))
			n := MustSucceed(w.NewAlignment())
			c := &log.Commit{
				Samples: 2, Alignment: uint64(n) << 32, First: 10, Last: 20,
				Slots: []log.Slot{{Segment: id, Length: 16, TailCRC: 7}},
			}
			Expect(w.Commit(c)).To(Succeed())
			var e log.Entry
			e.Add(&log.Create{Segment: id, Key: 5, Clock: 5, Version: 1, DataType: 4})
			e.Add(c)
			MustSucceed(l.Append(&e).Wait())
			Expect(w.Close()).To(Succeed())
			for i := range uint32(12) {
				var filler log.Entry
				for k := range uint32(10_000) {
					filler.Add(&log.Intent{Key: 1_000_000 + i*10_000 + k})
				}
				MustSucceed(l.Append(&filler).Wait())
			}
			Eventually(func() error { return readErr(fs, 2) }).Should(Succeed())
			Eventually(cp.Err).Should(Succeed())
			_, head := MustSucceed2(checkpoint.Read(fs, 2))
			live := MustSucceed(st.Snapshot())
			Expect(head.Sequence).To(BeNumerically("<=", live.Sequence))
			Expect(head.Channels).To(Equal(live.Channels))
			Expect(
				head.Channels[0].Segments[0].Sealed,
			).To(BeTrue(), "sealed at the switch")
			Expect(head.Intents).To(HaveLen(len(live.Intents)))
		})
})
