// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

const logSize = 4 << 20

// goExec runs each step of a log owner on a new goroutine, as production does.
func goExec(f func()) { go f() }

func noDurable(uint64, []log.Record) {}

func noRefused([]log.Record, error) {}

// durables collects what a log hands its OnDurable hook.
type durables struct {
	mu      sync.Mutex
	seqs    []uint64
	records [][]log.Record
	refused []error
}

func (d *durables) onRefused(_ []log.Record, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.refused = append(d.refused, err)
}

func (d *durables) refusals() []error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]error(nil), d.refused...)
}

func (d *durables) on(seq uint64, records []log.Record) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seqs = append(d.seqs, seq)
	d.records = append(d.records, records)
}

func (d *durables) get() ([]uint64, [][]log.Record) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]uint64(nil), d.seqs...), append([][]log.Record(nil), d.records...)
}

// bigEntry returns an entry of n bytes, a multiple of 8, that holds no Commit.
func bigEntry(n int) *log.Entry {
	GinkgoHelper()
	var e log.Entry
	e.Add(&log.Replace{Pointers: make([]log.Pointer, (n-48)/40)})
	return fill(&e, n)
}

// commitEntry returns an entry of n bytes, a multiple of 8, that holds a Commit.
func commitEntry(n int) *log.Entry {
	GinkgoHelper()
	var e log.Entry
	e.Add(&log.Commit{Samples: 1, Slots: make([]log.Slot, (n-48)/16)})
	return fill(&e, n)
}

// fill adds Restart records of 8 bytes to e until it is n bytes long.
func fill(e *log.Entry, n int) *log.Entry {
	GinkgoHelper()
	for len(e.Bytes()) < n {
		e.Add(&log.Restart{})
	}
	Expect(e.Bytes()).To(HaveLen(n))
	return e
}

func of(records ...log.Record) *log.Entry {
	var e log.Entry
	for _, r := range records {
		e.Add(r)
	}
	return &e
}

var logName = log.Name(generation)

// fileBytes returns the bytes of the log file in fs.
func fileBytes(fs xfs.FS) []byte {
	GinkgoHelper()
	return logBytes(fs, generation)
}

// logBytes returns the bytes of the file of the log of gen in fs.
func logBytes(fs xfs.FS, gen uint64) []byte {
	GinkgoHelper()
	f := MustSucceed(fs.Open(log.Name(gen), os.O_RDONLY))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	return MustSucceed(io.ReadAll(f))
}

// walk returns the frames of a log file in order. It looks for each next frame at the
// end of the last one, then at the next block boundary, and checks the sequence and the
// link of each.
func walk(b []byte) []log.Frame {
	GinkgoHelper()
	return walkLog(b, generation)
}

// walkLog returns the frames of the file of the log of gen, as walk does.
func walkLog(b []byte, gen uint64) []log.Frame {
	GinkgoHelper()
	h := MustSucceed(log.DecodeHeader(b, db, gen, 0))
	var (
		frames []log.Frame
		at     = int64(log.FramesStart)
		seq    = h.Sequence
		link   = binary.LittleEndian.Uint32(b[60:])
	)
	for {
		f, err := log.DecodeFrame(b[at:], db, gen, uint64(at))
		if err != nil {
			aligned := (at + log.BlockSize - 1) &^ (log.BlockSize - 1)
			if aligned == at {
				return frames
			}
			at = aligned
			if f, err = log.DecodeFrame(b[at:], db, gen, uint64(at)); err != nil {
				return frames
			}
		}
		Expect(f.Sequence).To(Equal(seq), "frame at %d", at)
		Expect(f.PrevCRC).To(Equal(link), "frame at %d", at)
		frames = append(frames, f)
		seq += uint64(f.Count)
		link = f.CRC
		at += int64(f.Size())
	}
}

// writes returns the offsets of the frame writes that ops record after the second
// header copy.
func writes(ops []string) []int64 {
	var out []int64
	for i, op := range ops {
		if op == fmt.Sprintf(
			"write %s %d %d",
			logName,
			log.HeaderCopy,
			log.HeaderSize,
		) {
			out = nil
			ops = ops[i+1:]
			break
		}
	}
	for _, op := range ops {
		var (
			name string
			off  int64
			n    int
		)
		if _, err := fmt.Sscanf(op, "write %s %d %d", &name, &off, &n); err == nil &&
			off >= log.FramesStart {
			out = append(out, off)
		}
	}
	return out
}

var _ = Describe("Log", func() {
	It("Should name a log by its generation in 16 hex digits", func() {
		Expect(log.Name(0xAB)).To(Equal("00000000000000ab.log"))
	})

	DescribeTable(
		"Should refuse a config with a field that is not set",
		func(change func(*log.Config), field string) {
			cfg := log.Config{
				FS:         xfs.NewMem(),
				Executor:   goExec,
				OnDurable:  noDurable,
				OnRefused:  noRefused,
				Prepare:    func(uint64) {},
				Switched:   func(uint64) {},
				Generation: 1,
				Sequence:   1,
			}
			change(&cfg)
			Expect(cfg.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(field)),
			))
			Expect(log.Start(cfg)).Error().To(MatchError(validate.ErrValidation))
		},
		Entry("file system", func(c *log.Config) { c.FS = nil }, "fs"),
		Entry("executor", func(c *log.Config) { c.Executor = nil }, "executor"),
		Entry("durable hook", func(c *log.Config) { c.OnDurable = nil }, "on_durable"),
		Entry("refused hook", func(c *log.Config) { c.OnRefused = nil }, "on_refused"),
		Entry("prepare hook", func(c *log.Config) { c.Prepare = nil }, "prepare"),
		Entry("switched hook", func(c *log.Config) { c.Switched = nil }, "switched"),
		Entry("generation", func(c *log.Config) { c.Generation = 0 }, "generation"),
		Entry("sequence", func(c *log.Config) { c.Sequence = 0 }, "sequence"),
	)

	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var (
				fs    *opFS
				l     *log.Log
				ran   chan error
				steps chan struct{}
				hook  *durables
				// prepare answers each Prepare of the log under test.
				prepare func(next uint64)
			)

			start := func(size int64, sequence uint64) {
				GinkgoHelper()
				Expect(log.Prepare(fs, generation, size)).To(Succeed())
				steps = make(chan struct{}, 1<<16)
				hook = &durables{}
				l = MustSucceed(log.Start(log.Config{
					FS: fs,
					Executor: func(f func()) {
						steps <- struct{}{}
						go f()
					},
					OnDurable:  hook.on,
					OnRefused:  hook.onRefused,
					Prepare:    func(next uint64) { prepare(next) },
					Switched:   func(uint64) {},
					DB:         db,
					Generation: generation,
					Sequence:   sequence,
					PrevCRC:    0xBEEF,
				}))
			}

			open := func(size int64) {
				GinkgoHelper()
				start(size, 5)
				ran = make(chan error, 1)
				ctx, cancel := context.WithCancel(context.Background())
				go func() { ran <- l.Run(ctx) }()
				DeferCleanup(cancel)
			}

			closeLog := func() {
				GinkgoHelper()
				Expect(l.Close()).To(Succeed())
				Eventually(ran).Should(Receive(BeNil()))
			}

			BeforeEach(func() {
				fs = newOpFS(makeFS())
				prepare = func(next uint64) { l.Prepared(next, errors.New("no space")) }
			})

			Describe("Prepare", func() {
				It("Should make a synced file of zeros and sync its entry", func() {
					Expect(log.Prepare(fs, generation, 3<<20)).To(Succeed())
					Expect(fileBytes(fs)).To(Equal(make([]byte, 3<<20)))
					Expect(fs.Ops()).To(HaveExactElements(
						"write "+logName+" 0 1048576",
						"write "+logName+" 1048576 1048576",
						"write "+logName+" 2097152 1048576",
						"sync "+logName,
						"syncdir ",
					))
				})
			})

			DescribeTable(
				"Should refuse a log file that is not a whole number of blocks, or "+
					"that ends its commits before its switch",
				func(size int64) {
					Expect(log.Prepare(fs, generation, size)).To(Succeed())
					Expect(log.Start(log.Config{
						FS:         fs,
						Executor:   goExec,
						OnDurable:  noDurable,
						OnRefused:  noRefused,
						Prepare:    func(uint64) {},
						Switched:   func(uint64) {},
						Generation: generation,
						Sequence:   1,
					})).Error().To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring("is not a whole number of blocks")),
					))
				},
				Entry("too small", int64(4*log.DeleteReserve-log.BlockSize)),
				Entry("not whole blocks", int64(logSize+100)),
			)

			It("Should write and sync both header copies at start, with the size of "+
				"the file", func() {
				open(logSize)
				DeferCleanup(closeLog)
				b := fileBytes(fs)
				want := log.Header{
					DB:         db,
					Generation: generation,
					Sequence:   5,
					Size:       logSize,
					PrevCRC:    0xBEEF,
				}
				Expect(log.DecodeHeader(b, db, generation, 0)).To(Equal(want))
				Expect(log.DecodeHeader(b[log.HeaderCopy:], db, generation,
					log.HeaderCopy)).To(Equal(want))
				Expect(fs.Ops()[len(fs.Ops())-1]).To(Equal("sync " + logName))
			})

			It(
				"Should ack an entry with its sequence once its frame is synced",
				func() {
					open(logSize)
					DeferCleanup(closeLog)
					entry := entryOf(&log.Drop{Key: 3})
					Expect(l.Append(of(&log.Drop{Key: 3})).Wait()).To(Equal(uint64(5)))
					ops := fs.Ops()
					at := len(ops) - 1
					want := fmt.Sprintf("write %s 8192 %d", logName, 32+len(entry))
					for ops[at] != want {
						at--
					}
					Expect(ops[at+1]).To(Equal("sync " + logName))
					f := walk(fileBytes(fs))[0]
					Expect(f.Count).To(Equal(uint32(1)))
					Expect(f.Entries).To(Equal(entry))
				},
			)

			It(
				"Should hand each synced entry's records to its hook in log order, "+
					"before its ack",
				func() {
					open(logSize)
					DeferCleanup(closeLog)
					release := fs.Hold()
					first := l.Append(of(&log.Drop{Key: 1}))
					Eventually(fs.held).Should(Receive())
					second := l.Append(of(&log.Drop{Key: 2}, &log.Intent{Key: 2}))
					third := l.Append(of(&log.Restart{}))
					Expect(hook.get()).To(BeEmpty())
					fs.Unhold()
					release <- struct{}{}
					Expect(first.Wait()).To(Equal(uint64(5)))
					seqs, _ := hook.get()
					Expect(seqs).To(ContainElement(uint64(5)))
					Expect(second.Wait()).To(Equal(uint64(6)))
					Expect(third.Wait()).To(Equal(uint64(7)))
					seqs, records := hook.get()
					Expect(seqs).To(Equal([]uint64{5, 6, 7}))
					Expect(records).To(Equal([][]log.Record{
						{&log.Drop{Key: 1}},
						{&log.Drop{Key: 2}, &log.Intent{Key: 2}},
						{&log.Restart{}},
					}))
				},
			)

			It(
				"Should hand no record of a frame whose sync failed to its hook",
				func() {
					open(logSize)
					fs.FailSync(errors.New("disk fault"))
					Expect(l.Append(of(&log.Drop{Key: 1})).Wait()).Error().
						To(MatchError(ContainSubstring("disk fault")))
					Expect(hook.get()).To(BeEmpty())
					Expect(l.Close()).To(MatchError(ContainSubstring("disk fault")))
					Eventually(ran).Should(Receive(BeNil()))
				},
			)

			It("Should run each step of the owner on its executor", func() {
				open(logSize)
				DeferCleanup(closeLog)
				Expect(l.Append(of(&log.Drop{Key: 3})).Wait()).To(Equal(uint64(5)))
				Eventually(steps).Should(Receive())
				Eventually(func() []int64 { return writes(fs.Ops()) }).
					Should(HaveLen(2))
				Consistently(steps).ShouldNot(Receive())
			})

			It("Should keep a copy of an entry that the caller resets", func() {
				open(logSize)
				DeferCleanup(closeLog)
				release := fs.Hold()
				first := l.Append(of(&log.Drop{Key: 1}))
				Eventually(fs.held).Should(Receive())
				e := of(&log.Drop{Key: 2})
				second := l.Append(e)
				e.Reset()
				e.Add(&log.Drop{Key: 9})
				fs.Unhold()
				release <- struct{}{}
				Expect(first.Wait()).To(Equal(uint64(5)))
				Expect(second.Wait()).To(Equal(uint64(6)))
				Expect(decodeAll(walk(fileBytes(fs))[1].Entries)).
					To(Equal([]any{log.Drop{Key: 2}}))
			})

			It("Should mark an isolated frame at the next block, and pack the next "+
				"frame after the mark", func() {
				open(logSize)
				DeferCleanup(closeLog)
				Expect(l.Append(of(&log.Drop{Key: 3})).Wait()).To(Equal(uint64(5)))
				Eventually(func() []int64 { return writes(fs.Ops()) }).
					Should(Equal([]int64{8_192, 12_288}))
				Expect(l.Append(of(&log.Drop{Key: 3})).Wait()).To(Equal(uint64(6)))
				Eventually(func() []int64 { return writes(fs.Ops()) }).
					Should(Equal([]int64{8_192, 12_288, 12_320, 16_384}))
				frames := walk(fileBytes(fs))
				Expect(frames).To(HaveLen(4))
				Expect(frames[1].Count).To(BeZero())
				Expect(frames[1].Sequence).To(Equal(uint64(6)))
			})

			It("Should write every entry queued during a sync as one frame", func() {
				open(logSize)
				DeferCleanup(closeLog)
				release := fs.Hold()
				first := l.Append(of(&log.Drop{Key: 1}))
				Eventually(fs.held).Should(Receive())
				second := l.Append(of(&log.Drop{Key: 2}))
				third := l.Append(of(&log.Drop{Key: 3}))
				fs.Unhold()
				release <- struct{}{}
				Expect(first.Wait()).To(Equal(uint64(5)))
				Expect(second.Wait()).To(Equal(uint64(6)))
				Expect(third.Wait()).To(Equal(uint64(7)))
				frames := walk(fileBytes(fs))
				Expect(frames[0].Count).To(Equal(uint32(1)))
				Expect(frames[1].Count).To(Equal(uint32(2)))
				Expect(decodeAll(frames[1].Entries)).To(Equal([]any{log.Drop{Key: 2}}))
			})

			It("Should start a frame at the next block when its block holds a "+
				"confirmed frame", func() {
				open(logSize)
				DeferCleanup(closeLog)
				size := int64(32 + len(entryOf(&log.Drop{Key: 3})))
				release := fs.Hold()
				a := l.Append(of(&log.Drop{Key: 3}))
				Eventually(fs.held).Should(Receive())
				b := l.Append(of(&log.Drop{Key: 3}))
				release <- struct{}{}
				Expect(a.Wait()).To(Equal(uint64(5)))
				Eventually(fs.held).Should(Receive())
				c := l.Append(of(&log.Drop{Key: 3}))
				fs.Unhold()
				release <- struct{}{}
				Expect(b.Wait()).To(Equal(uint64(6)))
				Expect(c.Wait()).To(Equal(uint64(7)))
				Expect(writes(fs.Ops())[:3]).
					To(Equal([]int64{8_192, 8_192 + size, 12_288}))
			})

			It("Should split a batch larger than a frame into frames", func() {
				open(32 << 20)
				DeferCleanup(closeLog)
				release := fs.Hold()
				first := l.Append(of(&log.Restart{}))
				Eventually(fs.held).Should(Receive())
				second := l.Append(bigEntry(10 << 20))
				third := l.Append(bigEntry(10 << 20))
				fs.Unhold()
				release <- struct{}{}
				Expect(first.Wait()).To(Equal(uint64(5)))
				Expect(second.Wait()).To(Equal(uint64(6)))
				Expect(third.Wait()).To(Equal(uint64(7)))
				frames := walk(fileBytes(fs))
				Expect(frames[1].Count).To(Equal(uint32(1)))
				Expect(frames[2].Count).To(Equal(uint32(1)))
			})

			It("Should stop an entry with a Commit at the delete reserve, and take "+
				"another entry there", func() {
				open(logSize)
				DeferCleanup(closeLog)
				n := logSize - log.DeleteReserve - log.FramesStart - 24
				Expect(l.Append(commitEntry(n)).Wait()).Error().To(SatisfyAll(
					MatchError(log.ErrFull),
					MatchError(ContainSubstring("does not fit in log 3")),
				))
				Expect(hook.refusals()).To(HaveExactElements(MatchError(log.ErrFull)))
				Expect(l.Append(bigEntry(n)).Wait()).To(Equal(uint64(5)))
				Expect(hook.refusals()).To(HaveLen(1))
			})

			It("Should take an entry that fits in a log that cannot switch after the "+
				"frame before it", func() {
				open(logSize)
				DeferCleanup(closeLog)
				release := fs.Hold()
				first := l.Append(of(&log.Drop{Key: 1}))
				Eventually(fs.held).Should(Receive())
				half := l.Append(bigEntry(logSize / 2))
				quarter := l.Append(bigEntry(logSize / 4))
				fs.Unhold()
				release <- struct{}{}
				Expect(first.Wait()).To(Equal(uint64(5)))
				Expect(half.Wait()).To(Equal(uint64(6)))
				Expect(quarter.Wait()).To(Equal(uint64(7)))
				Expect(hook.refusals()).To(BeEmpty())
			})

			It("Should report an entry refused at once to OnRefused", func() {
				open(logSize)
				closeLog()
				Expect(l.Append(of(&log.Drop{Key: 3})).Wait()).Error().
					To(MatchError(log.ErrClosed))
				Expect(hook.refusals()).To(HaveExactElements(MatchError(log.ErrClosed)))
			})

			It("Should stop an entry with no Commit two blocks before the end, so "+
				"both last marks fit", func() {
				open(logSize)
				n := logSize - 2*log.BlockSize - log.FramesStart - 32
				Expect(l.Append(bigEntry(n + 8)).Wait()).Error().
					To(MatchError(log.ErrFull))
				Expect(l.Append(bigEntry(n)).Wait()).To(Equal(uint64(5)))
				Eventually(func() []int64 { return writes(fs.Ops()) }).
					Should(Equal([]int64{8_192, logSize - 2*log.BlockSize}))
				closeLog()
				Expect(writes(fs.Ops())).To(Equal([]int64{
					8_192, logSize - 2*log.BlockSize, logSize - log.BlockSize,
				}))
				Expect(MustSucceed(fs.Stat(logName)).Size()).To(Equal(int64(logSize)))
			})

			It("Should refuse an entry larger than a frame", func() {
				open(logSize)
				DeferCleanup(closeLog)
				Expect(l.Append(bigEntry(log.MaxFrame)).Wait()).Error().
					To(MatchError(log.ErrFull))
			})

			DescribeTable(
				"Should fail every entry after a failed write or sync, until a restart",
				func(fail func(error)) {
					open(logSize)
					fault := errors.New("disk fault")
					fail(fault)
					Expect(
						l.Append(of(&log.Drop{Key: 1})).Wait(),
					).Error().
						To(SatisfyAll(
							MatchError(fault),
							MatchError(
								ContainSubstring("no entry is durable until a restart"),
							),
						))
					fail(nil)
					Expect(l.Append(of(&log.Drop{Key: 1})).Wait()).Error().
						To(MatchError(fault))
					Expect(l.Close()).To(MatchError(fault))
					Eventually(ran).Should(Receive(BeNil()))
				},
				Entry("sync", func(err error) { fs.FailSync(err) }),
				Entry("write", func(err error) { fs.FailWrite(err) }),
			)

			It("Should write queued entries and a clean mark at Close, then refuse "+
				"entries", func() {
				open(logSize)
				Expect(l.Append(of(&log.Drop{Key: 1})).Wait()).To(Equal(uint64(5)))
				closeLog()
				frames := walk(fileBytes(fs))
				last := frames[len(frames)-1]
				Expect(last.Clean).To(BeTrue())
				Expect(last.Sequence).To(Equal(uint64(6)))
				Expect(l.Append(of(&log.Drop{Key: 1})).Wait()).Error().
					To(MatchError(log.ErrClosed))
			})

			It("Should fail queued and later entries when the context of Run ends, "+
				"and return that error at Close", func() {
				start(logSize, 1)
				ctx, cancel := context.WithCancel(context.Background())
				ran = make(chan error, 1)
				go func() { ran <- l.Run(ctx) }()
				cancel()
				Eventually(ran).Should(Receive(MatchError(context.Canceled)))
				Expect(l.Append(of(&log.Drop{Key: 1})).Wait()).Error().
					To(MatchError(context.Canceled))
				Expect(l.Close()).To(MatchError(context.Canceled))
			})

			It("Should give each of many concurrent entries one sequence, in linked "+
				"frames", func() {
				open(8 << 20)
				DeferCleanup(closeLog)
				var (
					wg   sync.WaitGroup
					mu   sync.Mutex
					seqs []uint64
				)
				for w := range 8 {
					wg.Go(func() {
						defer GinkgoRecover()
						for i := range 100 {
							seq := MustSucceed(
								l.Append(of(&log.Drop{Key: uint32(w*100 + i)})).Wait(),
							)
							mu.Lock()
							seqs = append(seqs, seq)
							mu.Unlock()
						}
					})
				}
				wg.Wait()
				Expect(seqs).To(HaveLen(800))
				Expect(seqs).To(ContainElements(uint64(5), uint64(804)))
				var entries uint32
				for _, f := range walk(fileBytes(fs)) {
					entries += f.Count
				}
				Expect(entries).To(Equal(uint32(800)))
			})
		})
	}
})
