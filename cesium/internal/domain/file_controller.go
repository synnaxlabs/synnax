// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

const extension = ".domain"

func fileKeyToName(key uint16) string {
	return strconv.Itoa(int(key)) + extension
}

// fileReaders represents readers on a file. It provides a mutex lock to prevent any
// modifications to the list of readers.
type fileReaders struct {
	open []controlledReader
	sync.RWMutex
}

type fileController struct {
	counter     *xio.Int32Counter
	counterFile io.Closer
	// released closes when a handle goes idle or a descriptor frees, waking every
	// waiter to check again. It is nil while no waiter holds it.
	released struct {
		ch chan struct{}
		sync.Mutex
	}
	// descriptors counts open and opening file handles. A handle must reserve a place
	// with reserveDescriptor before it opens, so the count never passes MaxDescriptors.
	descriptors atomic.Int32
	writers     struct {
		open map[uint16]controlledWriter
		// unopened is a set of file keys to files that are not oversize and do not have
		// any file handles for them in open.
		unopened set.Set[uint16]
		sync.RWMutex
	}
	readers struct {
		files map[uint16]*fileReaders
		sync.RWMutex
	}
	Config
}

const counterFile = "counter" + extension

func openFileController(cfg Config) (*fileController, error) {
	counterF, err := cfg.FS.Open(counterFile, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	counter, err := xio.NewInt32Counter(counterF)
	if err != nil {
		return nil, err
	}
	fc := &fileController{
		Config:      cfg,
		counter:     counter,
		counterFile: counterF,
	}
	fc.writers.open = make(map[uint16]controlledWriter, cfg.MaxDescriptors)
	fc.readers.files = make(map[uint16]*fileReaders)
	if fc.writers.unopened, err = fc.scanUnopenedFiles(); err != nil {
		return nil, err
	}
	return fc, nil
}

// realFileSizeCap returns the maximum allowed size of a file — though it may be
// exceeded if commits are sparse. fc.Config.FileSize is the nominal file size to not
// exceed, in reality, this size is set to 0.8 * the actual file size cap, therefore
// the real size is 1.25 * the nominal size.
func (fc *fileController) realFileSizeCap() telem.Size {
	return telem.Size(math.Round(1.25 * float64(fc.FileSize)))
}

func (fc *fileController) scanUnopenedFiles() (set.Set[uint16], error) {
	unopened := make(set.Set[uint16])
	for i := 1; i <= int(fc.counter.Value()); i++ {
		e, err := fc.FS.Exists(fileKeyToName(uint16(i)))
		if err != nil {
			return unopened, err
		}
		if !e {
			continue
		}

		s, err := fc.FS.Stat(fileKeyToName(uint16(i)))
		if err != nil {
			return unopened, err
		}
		if s.Size() < int64(fc.FileSize) {
			unopened.Add(uint16(i))
		}
	}

	return unopened, nil
}

// acquireWriter acquires a writer for a file in the file system. The order it acquires
// is as follows:
//
// 1. If any open file handles (writers.open) are present and are not currently
// controlled, and the file is not oversize, it is acquired.
// 2. If no open file handles are acquired, then the file controller attempts to acquire
// a handle for a closed file (writers.unopened).
// 3. If no unopened files are available, then the file controller creates a new file
// handle to a new file, as governed by counter.
func (fc *fileController) acquireWriter(
	ctx context.Context,
) (key uint16, size int64, w xio.TrackedWriteCloser, err error) {
	ctx, span := fc.T.Bench(ctx, "acquire_writer")
	defer span.End()
	err = fc.waitForHandle(ctx, func() (acquired bool, err error) {
		key, size, w, acquired, err = fc.tryAcquireWriter(ctx)
		return acquired, err
	})
	return key, size, w, span.Error(err)
}

func (fc *fileController) tryAcquireWriter(
	ctx context.Context,
) (uint16, int64, xio.TrackedWriteCloser, bool, error) {
	fc.writers.RLock()
	for fileKey, w := range fc.writers.open {
		s, err := fc.FS.Stat(fileKeyToName(fileKey))
		if err != nil {
			fc.writers.RUnlock()
			return 0, 0, nil, false, err
		}
		size := s.Size()
		if size < int64(fc.FileSize) && w.tryAcquire() {
			fc.writers.RUnlock()
			return w.fileKey, size, &w, true, nil
		}
	}
	fc.writers.RUnlock()

	if fc.reserveDescriptor() {
		w, size, err := fc.newWriter(ctx)
		if err != nil {
			return 0, size, nil, false, err
		}
		return w.fileKey, size, w, true, nil
	}
	if _, err := fc.gcWriters(); err != nil {
		return 0, 0, nil, false, err
	}
	_, err := fc.gcReaders()
	return 0, 0, nil, false, err
}

// waitForHandle calls try until it acquires a handle, waiting for a handle to go idle
// or a descriptor to free between calls. It holds no locks while it waits. It returns
// ctx's error if ctx ends first.
func (fc *fileController) waitForHandle(
	ctx context.Context,
	try func() (bool, error),
) error {
	var released <-chan struct{}
	for {
		acquired, err := try()
		if acquired || err != nil {
			return err
		}
		// Take the channel before the next try, so that a release after that try
		// closes the channel this call waits on.
		if released == nil {
			released = fc.awaitRelease()
			continue
		}
		select {
		case <-released:
			released = nil
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "cannot acquire a file descriptor")
		}
	}
}

func (fc *fileController) awaitRelease() <-chan struct{} {
	fc.released.Lock()
	defer fc.released.Unlock()
	if fc.released.ch == nil {
		fc.released.ch = make(chan struct{})
	}
	return fc.released.ch
}

func (fc *fileController) notifyRelease() {
	fc.released.Lock()
	defer fc.released.Unlock()
	if fc.released.ch != nil {
		close(fc.released.ch)
		fc.released.ch = nil
	}
}

// newWriter creates a new writing file handle from the file controller: it first
// attempts to create a file handle for files from the directory that are not at
// capacity. If there is none, it creates a new file and increments the counter.
func (fc *fileController) newWriter(
	ctx context.Context,
) (opened *controlledWriter, _ int64, _ error) {
	_, span := fc.T.Bench(ctx, "new_writer")
	fc.writers.Lock()

	defer func() {
		fc.writers.Unlock()
		span.End()
		if opened == nil {
			fc.freeDescriptor()
		}
	}()

	lastFileKey := uint16(0)
	for key := range fc.writers.unopened {
		// Optimization: prioritize writing to existing files that are not full
		// rather than at the end.
		if key == uint16(fc.counter.Value()) {
			lastFileKey = key
			continue
		}
		w, size, err := fc.openWriterFile(key, os.O_WRONLY|os.O_APPEND)
		return w, size, span.Error(err)
	}

	if lastFileKey != 0 {
		w, size, err := fc.openWriterFile(lastFileKey, os.O_WRONLY|os.O_APPEND)
		return w, size, span.Error(err)
	}

	if fc.counter.Value() >= math.MaxUint16 {
		return nil, 0, span.Error(ErrFileLimit)
	}
	nextCtr, err := fc.counter.Add(1)
	if err != nil {
		return nil, 0, span.Error(err)
	}
	w, size, err := fc.openWriterFile(
		uint16(nextCtr),
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
	)
	return w, size, span.Error(err)
}

// openWriterFile opens the file and adds its handle to writers.open. The caller must
// hold the writers lock. The file stays closed and unregistered on an error.
func (fc *fileController) openWriterFile(
	key uint16,
	flag int,
) (*controlledWriter, int64, error) {
	file, err := fc.FS.Open(fileKeyToName(key), flag)
	if err != nil {
		return nil, 0, err
	}
	base, err := xio.NewTrackedWriteCloser(file)
	if err != nil {
		return nil, 0, errors.Join(err, file.Close())
	}
	s, err := file.Stat()
	if err != nil {
		return nil, 0, errors.Join(err, file.Close())
	}
	w := controlledWriter{
		TrackedWriteCloser: base,
		controllerEntry:    newPoolEntry(key, fc),
	}
	fc.writers.open[key] = w
	delete(fc.writers.unopened, key)
	return &w, s.Size(), nil
}

func (fc *fileController) acquireReader(
	ctx context.Context,
	key uint16,
) (r *controlledReader, err error) {
	ctx, span := fc.T.Bench(ctx, "acquire_reader")
	defer span.End()
	err = fc.waitForHandle(ctx, func() (acquired bool, err error) {
		r, err = fc.tryAcquireReader(ctx, key)
		return r != nil, err
	})
	return r, err
}

func (fc *fileController) tryAcquireReader(
	ctx context.Context,
	key uint16,
) (*controlledReader, error) {
	fc.readers.RLock()
	if f, ok := fc.readers.files[key]; ok {
		f.RLock()
		for _, r := range f.open {
			if r.tryAcquire() {
				f.RUnlock()
				fc.readers.RUnlock()
				return &r, nil
			}
		}
		f.RUnlock()
	}
	fc.readers.RUnlock()

	if fc.reserveDescriptor() {
		return fc.newReader(ctx, key)
	}
	if collected, err := fc.gcReaders(); collected || err != nil {
		return nil, err
	}
	return nil, fc.closeIdleWriter()
}

func (fc *fileController) newReader(
	ctx context.Context,
	key uint16,
) (opened *controlledReader, _ error) {
	_, span := fc.T.Bench(ctx, "new_reader")
	defer func() {
		span.End()
		if opened == nil {
			fc.freeDescriptor()
		}
	}()
	file, err := fc.FS.Open(
		fileKeyToName(key),
		os.O_RDONLY,
	)
	if err != nil {
		return nil, span.Error(err)
	}

	r := controlledReader{
		ReaderAtCloser:  file,
		controllerEntry: newPoolEntry(key, fc),
	}
	fc.readers.Lock()
	f, ok := fc.readers.files[key]
	if !ok {
		fc.readers.files[key] = &fileReaders{open: []controlledReader{r}}
	} else {
		f.Lock()
		fc.readers.files[key].open = append(fc.readers.files[key].open, r)
		f.Unlock()
	}
	fc.readers.Unlock()
	return &r, err
}

func (fc *fileController) gcReaders() (successful bool, err error) {
	fc.readers.Lock()
	defer fc.readers.Unlock()
	for k, f := range fc.readers.files {
		func() {
			f.Lock()
			defer f.Unlock()
			f.open = lo.Filter(f.open, func(r controlledReader, i int) bool {
				if !r.tryAcquire() {
					// If file is held by someone else, we can't gc.
					return true
				}
				// A failed close still retires the handle and frees its place.
				err = errors.Join(err, r.HardClose())
				successful = true
				return false
			})
			if len(fc.readers.files[k].open) == 0 {
				delete(fc.readers.files, k)
			}
		}()
	}
	return successful, err
}

// gcWriters closes all open writers to oversize files.
func (fc *fileController) gcWriters() (bool, error) {
	fc.writers.Lock()
	defer fc.writers.Unlock()
	collected := false
	for k, w := range fc.writers.open {
		s, err := fc.FS.Stat(fileKeyToName(k))
		if err != nil {
			return collected, err
		}

		if s.Size() >= int64(fc.FileSize) && w.tryAcquire() {
			err = w.HardClose()
			delete(fc.writers.open, k)
			collected = true
			if err != nil {
				return collected, err
			}
		}
	}
	return collected, nil
}

// closeIdleWriter closes one idle writer handle so that a reader can use its place. A
// file with room stays open to writers through writers.unopened.
func (fc *fileController) closeIdleWriter() error {
	fc.writers.Lock()
	defer fc.writers.Unlock()
	for k, w := range fc.writers.open {
		if !w.tryAcquire() {
			continue
		}
		closeErr := w.HardClose()
		delete(fc.writers.open, k)
		s, err := fc.FS.Stat(fileKeyToName(k))
		if err != nil {
			return errors.Join(closeErr, err)
		}
		if s.Size() < int64(fc.FileSize) {
			fc.writers.unopened.Add(k)
		}
		return closeErr
	}
	return nil
}

// prepareForGC atomically checks whether a file can be garbage collected and, if so,
// removes it from the writer pool to prevent a writer from opening on it during GC.
// Returns whether GC can proceed and whether the file was in the unopened set (so
// the caller can restore it on a no-op).
func (fc *fileController) prepareForGC(fileKey uint16) (canGC, wasUnopened bool) {
	fc.writers.Lock()
	defer fc.writers.Unlock()
	if _, ok := fc.writers.open[fileKey]; ok {
		return false, false
	}
	_, wasUnopened = fc.writers.unopened[fileKey]
	fc.writers.unopened.Remove(fileKey)
	return true, wasUnopened
}

// restoreUnopened adds a file key back to the unopened writer set. Called when
// garbageCollectFile determines that no GC is needed for a file that was temporarily
// removed from the writer pool.
func (fc *fileController) restoreUnopened(fileKey uint16) {
	fc.writers.Lock()
	defer fc.writers.Unlock()
	fc.writers.unopened.Add(fileKey)
}

// rejuvenate adds a file key to the unopened writers set. If there is an open writer
// for it, it is removed. rejuvenate is called after a file is garbage collected.
func (fc *fileController) rejuvenate(fileKey uint16) error {
	fc.writers.Lock()
	defer fc.writers.Unlock()

	if w, ok := fc.writers.open[fileKey]; ok {
		if !w.tryAcquire() {
			return newResourceInUseError("writer", fileKey)
		}
		err := w.HardClose()
		delete(fc.writers.open, fileKey)
		if err != nil {
			return err
		}
	}

	s, err := fc.FS.Stat(fileKeyToName(fileKey))
	if err != nil {
		return err
	}
	if telem.Size(s.Size()) < fc.FileSize {
		fc.writers.unopened[fileKey] = struct{}{}
	}
	return nil
}

// reserveDescriptor returns true if a place is free for a new file handle. The caller
// must end the place with freeDescriptor or a hard close of the handle.
func (fc *fileController) reserveDescriptor() bool {
	for {
		n := fc.descriptors.Load()
		if int(n) >= fc.MaxDescriptors {
			return false
		}
		if fc.descriptors.CompareAndSwap(n, n+1) {
			return true
		}
	}
}

func (fc *fileController) freeDescriptor() {
	fc.descriptors.Add(-1)
	fc.notifyRelease()
}

func (fc *fileController) close() error {
	fc.writers.RLock()
	fc.readers.RLock()
	defer func() {
		fc.readers.RUnlock()
		fc.writers.RUnlock()
	}()
	var err error
	for _, w := range fc.writers.open {
		if !w.tryAcquire() {
			err = errors.Join(err, newResourceInUseError("writer", w.fileKey))
		} else {
			err = errors.Join(err, w.HardClose())
		}
	}
	for _, f := range fc.readers.files {
		f.Lock()
		for _, r := range f.open {
			if !r.tryAcquire() {
				err = errors.Join(err, newResourceInUseError("reader", r.fileKey))
			} else {
				err = errors.Join(err, r.HardClose())
			}
		}
		f.Unlock()
	}
	return errors.Join(err, fc.counterFile.Close())
}

type controlledWriter struct {
	xio.TrackedWriteCloser
	controllerEntry
}

func (c *controlledWriter) tryAcquire() bool {
	acquired := c.controllerEntry.tryAcquire()
	if acquired {
		c.Reset()
	}
	return acquired
}

func (c *controlledWriter) Close() error {
	return c.controllerEntry.Close()
}

func (c *controlledWriter) HardClose() error {
	return c.hardClose(c.TrackedWriteCloser)
}

type controlledReader struct {
	xio.ReaderAtCloser
	controllerEntry
}

func (c *controlledReader) Close() error {
	return c.controllerEntry.Close()
}

func (c *controlledReader) HardClose() error {
	return c.hardClose(c.ReaderAtCloser)
}

type controllerEntry struct {
	inUse   *atomic.Bool
	fc      *fileController
	fileKey uint16
}

func newPoolEntry(key uint16, fc *fileController) controllerEntry {
	ce := controllerEntry{fc: fc, fileKey: key, inUse: &atomic.Bool{}}
	ce.inUse.Store(true)
	return ce
}

func (ce *controllerEntry) Close() error {
	if !ce.inUse.CompareAndSwap(true, false) {
		ce.fc.L.DPanic(fmt.Sprintf("controller: entry %d already closed", ce.fileKey))
		return nil
	}
	ce.fc.notifyRelease()
	return nil
}

// hardClose closes f and frees its descriptor, even if the close fails. The entry stays
// in use, so tryAcquire can never return it and no caller can close it twice.
func (ce *controllerEntry) hardClose(f io.Closer) error {
	err := f.Close()
	ce.fc.freeDescriptor()
	return err
}

func (ce *controllerEntry) tryAcquire() bool {
	return ce.inUse.CompareAndSwap(false, true)
}
