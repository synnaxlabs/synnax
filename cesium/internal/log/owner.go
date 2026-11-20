// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"sync"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

var (
	// ErrFull is the error of an entry that does not fit in the log file. Only an
	// entry of deletes may use the last DeleteReserve bytes.
	ErrFull = errors.New("log file is full")
	// ErrClosed is the error of an entry appended after Close.
	ErrClosed = errors.New("log is closed")
)

// Name returns the file name of the log of generation.
func Name(generation uint64) string { return fmt.Sprintf("%016x.log", generation) }

// Prepare makes the file of the log of generation in fs: size zeros, synced, with its
// entry in fs synced.
func Prepare(fs xfs.FS, generation uint64, size int64) error {
	f, err := fs.Open(Name(generation), os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return err
	}
	zeros := make([]byte, min(size, 1<<20))
	for off := int64(0); off < size; off += int64(len(zeros)) {
		if _, err = f.WriteAt(
			zeros[:min(int64(len(zeros)), size-off)],
			off,
		); err != nil {
			return errors.Combine(err, f.Close())
		}
	}
	if err = f.Sync(); err != nil {
		return errors.Combine(err, f.Close())
	}
	if err = f.Close(); err != nil {
		return err
	}
	return fs.SyncDir("")
}

// Config is the configuration of a log.
type Config struct {
	// FS is the root of the database.
	FS xfs.FS
	// DB is the identity of the database.
	DB uint64
	// Generation is the generation of the log.
	Generation uint64
	// Sequence is the sequence of the first entry of the log, from 1.
	Sequence uint64
	// PrevCRC is the crc field of the last frame of the generation before, or 0.
	PrevCRC uint32
	// Size is the size of the prepared log file, at most FileSize.
	Size int64
}

type pending struct {
	entry   []byte
	deletes bool
	done    chan result
}

type result struct {
	sequence uint64
	err      error
}

// Log appends entries to one log file. One goroutine runs it, and it writes every
// entry queued while its last sync ran as one frame. It is safe for concurrent use.
type Log struct {
	cfg  Config
	file xfs.File
	wake chan struct{}
	exit chan struct{}
	mu   struct {
		sync.Mutex
		queue   []pending
		err     error
		closing bool
	}
	// The fields below belong to the goroutine of Run.
	seq      uint64
	prevCRC  uint32
	prevEnd  int64
	lastEnd  int64
	needMark bool
	buf      []byte
	wbuf     []byte
	frame    Frame
	runErr   error
}

// Start writes both header copies of the prepared log of cfg, syncs it, and returns
// the log, ready for its first frame.
func Start(cfg Config) (*Log, error) {
	file, err := cfg.FS.Open(Name(cfg.Generation), os.O_RDWR)
	if err != nil {
		return nil, err
	}
	h := Header{
		DB:         cfg.DB,
		Generation: cfg.Generation,
		Sequence:   cfg.Sequence,
		Size:       uint64(cfg.Size),
		PrevCRC:    cfg.PrevCRC,
	}
	first := h.Encode(0)
	if _, err = file.WriteAt(first, 0); err == nil {
		if _, err = file.WriteAt(h.Encode(HeaderCopy), HeaderCopy); err == nil {
			err = file.Sync()
		}
	}
	if err != nil {
		return nil, errors.Combine(err, file.Close())
	}
	return &Log{
		cfg:     cfg,
		file:    file,
		wake:    make(chan struct{}, 1),
		exit:    make(chan struct{}),
		seq:     cfg.Sequence,
		prevCRC: binary.LittleEndian.Uint32(first[60:]),
		prevEnd: FramesStart,
		lastEnd: FramesStart,
	}, nil
}

// Ack reports when an appended entry is durable.
type Ack struct{ done chan result }

// Wait returns the sequence of the entry once it is durable, or the error that kept it
// from being durable.
func (a Ack) Wait() (uint64, error) {
	r := <-a.done
	return r.sequence, r.err
}

// Append queues a copy of entry and returns at once. Entries reach the log in the order
// of their Append calls. An entry with deletes may use the last DeleteReserve bytes of
// the file. After a failed write or sync, every entry fails with that error until a
// restart.
func (l *Log) Append(entry []byte, deletes bool) Ack {
	a := Ack{done: make(chan result, 1)}
	l.mu.Lock()
	if err := l.refusal(); err != nil {
		l.mu.Unlock()
		a.done <- result{err: err}
		return a
	}
	l.mu.queue = append(l.mu.queue, pending{
		entry:   append([]byte(nil), entry...),
		deletes: deletes,
		done:    a.done,
	})
	l.mu.Unlock()
	l.signal()
	return a
}

// refusal returns the error that refuses a new entry. The caller holds l.mu.
func (l *Log) refusal() error {
	if l.mu.err != nil {
		return l.mu.err
	}
	if l.mu.closing {
		return ErrClosed
	}
	return nil
}

func (l *Log) signal() {
	select {
	case l.wake <- struct{}{}:
	default:
	}
}

// Run writes queued entries until Close, or until ctx is done. At Close, it writes the
// queued entries and a clean mark, and returns nil. When ctx is done first, it fails
// the queued entries and returns the error of ctx.
func (l *Log) Run(ctx context.Context) error {
	defer close(l.exit)
	for {
		select {
		case <-ctx.Done():
			l.fail(ctx.Err())
			return ctx.Err()
		case <-l.wake:
		}
		queue, closing := l.take()
		l.write(queue)
		if closing {
			l.runErr = l.closeClean()
			return nil
		}
		if l.needMark && l.idle() {
			l.mark(false)
		}
	}
}

func (l *Log) take() ([]pending, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	queue := l.mu.queue
	l.mu.queue = nil
	return queue, l.mu.closing
}

func (l *Log) idle() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.mu.queue) == 0 && !l.mu.closing && l.mu.err == nil
}

// write writes queue as frames of at most MaxFrame bytes, one sync each, and acks each
// entry after the sync of its frame.
func (l *Log) write(queue []pending) {
	var batch []pending
	l.buf = l.buf[:0]
	start := l.nextStart()
	flush := func() {
		if len(batch) == 0 {
			return
		}
		l.frame = Frame{Count: uint32(len(batch)), Sequence: l.seq, PrevCRC: l.prevCRC}
		l.frame.Entries = l.buf
		if err := l.writeFrame(start); err != nil {
			for _, p := range batch {
				p.done <- result{err: err}
			}
		} else {
			for i, p := range batch {
				p.done <- result{sequence: l.frame.Sequence + uint64(i)}
			}
			l.needMark = true
		}
		batch, l.buf = batch[:0], l.buf[:0]
		start = l.nextStart()
	}
	for _, p := range queue {
		if err := l.failure(); err != nil {
			p.done <- result{err: err}
			continue
		}
		size := int64(FrameHeaderSize + len(l.buf) + len(p.entry))
		if size > MaxFrame {
			flush()
			size = int64(FrameHeaderSize + len(p.entry))
		}
		// A mark must still fit at the block after the last frame.
		limit := l.cfg.Size - DeleteReserve
		if p.deletes {
			limit = l.cfg.Size - BlockSize
		}
		if size > MaxFrame || start+size > limit {
			p.done <- result{err: errors.Wrapf(
				ErrFull,
				"an entry of %d bytes does not fit in log %d",
				len(p.entry),
				l.cfg.Generation,
			)}
			continue
		}
		l.buf = append(l.buf, p.entry...)
		batch = append(batch, p)
	}
	flush()
}

// nextStart returns where the next frame with entries starts. It starts where the last
// frame ends, unless the block there holds a frame before the last one, which the
// last one's sync confirmed.
func (l *Log) nextStart() int64 {
	block := l.lastEnd &^ (BlockSize - 1)
	if l.lastEnd == block || l.prevEnd <= block {
		return l.lastEnd
	}
	return block + BlockSize
}

// writeFrame writes l.frame at start and syncs. A failure fails the log.
func (l *Log) writeFrame(start int64) error {
	if err := l.failure(); err != nil {
		return err
	}
	l.wbuf = l.frame.Append(l.wbuf[:0], l.cfg.DB, l.cfg.Generation, uint64(start))
	b := l.wbuf
	if _, err := l.file.WriteAt(b, start); err != nil {
		l.fail(err)
		return l.failure()
	}
	if err := l.file.Sync(); err != nil {
		l.fail(err)
		return l.failure()
	}
	l.seq += uint64(l.frame.Count)
	l.prevCRC = l.frame.CRC
	l.prevEnd, l.lastEnd = l.lastEnd, start+int64(len(b))
	return nil
}

// mark writes and syncs a frame with no entry at the next block boundary, which
// confirms every frame before it.
func (l *Log) mark(clean bool) {
	l.frame = Frame{Sequence: l.seq, PrevCRC: l.prevCRC, Clean: clean}
	start := (l.lastEnd + BlockSize - 1) &^ (BlockSize - 1)
	if err := l.writeFrame(start); err == nil {
		l.needMark = false
	}
}

func (l *Log) closeClean() error {
	l.mark(true)
	return l.failure()
}

func (l *Log) failure() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.mu.err
}

// fail fails the log with err and every queued entry.
func (l *Log) fail(err error) {
	l.mu.Lock()
	if l.mu.err == nil {
		l.mu.err = errors.Wrapf(
			err,
			"log %d failed, and no entry is durable until a restart",
			l.cfg.Generation,
		)
	}
	failed, queue := l.mu.err, l.mu.queue
	l.mu.queue = nil
	l.mu.Unlock()
	for _, p := range queue {
		p.done <- result{err: failed}
	}
}

// Close refuses new entries, waits for Run to write the queued ones and a clean mark,
// and closes the file. It returns the error of the log, if it failed. Call it while
// Run runs, or after Run returns.
func (l *Log) Close() error {
	l.mu.Lock()
	l.mu.closing = true
	l.mu.Unlock()
	l.signal()
	<-l.exit
	return errors.Combine(l.runErr, l.file.Close())
}
