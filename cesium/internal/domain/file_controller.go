// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/alamos"
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
	sync.RWMutex
	open []controlledReader
}

type fileController struct {
	Config
	writers struct {
		sync.RWMutex
		open map[uint16]controlledWriter
		// unopened is a set of file keys to files that are not oversize and do not have
		// any file handles for them in open.
		unopened set.Set[uint16]
	}
	readers struct {
		sync.RWMutex
		files map[uint16]*fileReaders
	}
	release     chan struct{}
	counter     *xio.Int32Counter
	counterFile io.Closer
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
	fc.release = make(chan struct{}, cfg.MaxDescriptors)
	if fc.writers.unopened, err = fc.scanUnopenedFiles(); err != nil {
		return nil, err
	}
	return fc, nil
}

// realFileSizeCap returns the maximum allowed size of a file — though it may be
// exceeded if commits are sparse. fc.ConfigValues.FileSize is the nominal file size to not
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
func (fc *fileController) acquireWriter(ctx context.Context) (uint16, int64, xio.TrackedWriteCloser, error) {
	ctx, span := fc.T.Bench(ctx, "acquire_writer")
	defer span.End()

	fc.writers.RLock()
	for fileKey, w := range fc.writers.open {
		s, err := fc.FS.Stat(fileKeyToName(fileKey))
		if err != nil {
			return 0, 0, nil, err
		}
		size := s.Size()
		if size < int64(fc.FileSize) && w.tryAcquire() {
			fc.writers.RUnlock()
			return w.fileKey, size, &w, nil
		}
	}

	fc.writers.RUnlock()

	if !fc.atDescriptorLimit() {
		w, size, err := fc.newWriter(ctx)
		if err != nil {
			return 0, size, nil, err
		}
		return w.fileKey, size, w, span.Error(err)
	}

	ok, err := fc.gcWriters()
	if err != nil {
		return 0, 0, nil, span.Error(err)
	}
	if ok {
		return fc.acquireWriter(ctx)
	}

	ok, err = fc.gcReaders()
	if err != nil {
		return 0, 0, nil, span.Error(err)
	}
	if ok {
		return fc.acquireWriter(ctx)
	}

	<-fc.release
	return fc.acquireWriter(ctx)
}

// newWriter creates a new writing file handle from the file controller: it first
// attempts to create a file handle for files from the directory that are not at
// capacity. If there is none, it creates a new file and increments the counter.
func (fc *fileController) newWriter(ctx context.Context) (*controlledWriter, int64, error) {
	_, span := fc.T.Bench(ctx, "new_writer")
	fc.writers.Lock()

	defer func() {
		fc.writers.Unlock()
		span.End()
	}()

	lastFileKey := uint16(0)
	for key := range fc.writers.unopened {
		// Optimization: prioritize writing to existing files that are not full
		// rather than at the end.
		if key == uint16(fc.counter.Value()) {
			lastFileKey = key
			continue
		}

		file, err := fc.FS.Open(fileKeyToName(key), os.O_WRONLY|os.O_APPEND)
		if err != nil {
			return nil, 0, span.Error(err)
		}
		base, err := xio.NewTrackedWriteCloser(file)
		if err != nil {
			return nil, 0, span.Error(err)
		}
		w := controlledWriter{
			TrackedWriteCloser: base,
			controllerEntry:    newPoolEntry(key, fc.release, fc.Instrumentation),
		}
		fc.writers.open[key] = w
		delete(fc.writers.unopened, key)

		s, err := file.Stat()
		if err != nil {
			return nil, 0, span.Error(err)
		}
		return &w, s.Size(), nil
	}

	if lastFileKey != 0 {
		file, err := fc.FS.Open(fileKeyToName(lastFileKey), os.O_WRONLY|os.O_APPEND)
		if err != nil {
			return nil, 0, span.Error(err)
		}
		base, err := xio.NewTrackedWriteCloser(file)
		if err != nil {
			return nil, 0, span.Error(err)
		}
		w := controlledWriter{
			TrackedWriteCloser: base,
			controllerEntry:    newPoolEntry(lastFileKey, fc.release, fc.Instrumentation),
		}
		fc.writers.open[lastFileKey] = w
		delete(fc.writers.unopened, lastFileKey)

		s, err := file.Stat()
		if err != nil {
			return nil, 0, span.Error(err)
		}
		return &w, s.Size(), span.Error(err)
	}

	nextKey_, err := fc.counter.Add(1)
	if err != nil {
		return nil, 0, span.Error(err)
	}
	nextKey := uint16(nextKey_)
	file, err := fc.FS.Open(
		fileKeyToName(nextKey),
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
	)
	if err != nil {
		return nil, 0, span.Error(err)
	}
	base, err := xio.NewTrackedWriteCloser(file)
	if err != nil {
		return nil, 0, span.Error(err)
	}
	w := controlledWriter{
		TrackedWriteCloser: base,
		controllerEntry:    newPoolEntry(nextKey, fc.release, fc.Instrumentation),
	}
	fc.writers.open[nextKey] = w
	return &w, 0, nil
}

func (fc *fileController) acquireReader(ctx context.Context, key uint16) (*controlledReader, error) {
	ctx, span := fc.T.Bench(ctx, "acquire_reader")
	defer span.End()

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

	if !fc.atDescriptorLimit() {
		return fc.newReader(ctx, key)
	}

	ok, err := fc.gcReaders()
	if err != nil {
		return nil, err
	}
	if ok {
		return fc.acquireReader(ctx, key)
	}

	ok, err = fc.gcWriters()
	if err != nil {
		return nil, err
	}
	if ok {
		return fc.acquireReader(ctx, key)
	}
	<-fc.release
	return fc.acquireReader(ctx, key)
}

func (fc *fileController) newReader(ctx context.Context, key uint16) (*controlledReader, error) {
	_, span := fc.T.Bench(ctx, "new_reader")
	defer span.End()
	file, err := fc.FS.Open(
		fileKeyToName(key),
		os.O_RDONLY,
	)
	if err != nil {
		return nil, span.Error(err)
	}

	r := controlledReader{
		ReaderAtCloser:  file,
		controllerEntry: newPoolEntry(key, fc.release, fc.Instrumentation),
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
			f.open = lo.Filter[controlledReader](f.open, func(r controlledReader, i int) bool {
				if !r.tryAcquire() {
					// If file is held by someone else, we can't gc.
					return true
				}
				// If we encounter an error, we can't gc.
				if err = r.HardClose(); err != nil {
					return true
				}
				successful = true
				// Good to GC, remove from open.
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
			if err != nil {
				return collected, err
			}
			delete(fc.writers.open, k)
			collected = true
		}
	}
	return collected, nil
}

func (fc *fileController) hasWriter(fileKey uint16) bool {
	fc.writers.RLock()
	defer fc.writers.RUnlock()

	_, ok := fc.writers.open[fileKey]
	return ok
}

// rejuvenate adds a file key to the unopened writers set. If there is an open writer
// for it, it is removed. rejuvenate is called after a file is garbage collected.
func (fc *fileController) rejuvenate(fileKey uint16) error {
	fc.writers.Lock()
	defer fc.writers.Unlock()

	if w, ok := fc.writers.open[fileKey]; ok {
		if !w.tryAcquire() {
			return newErrResourceInUse("writer", fileKey)
		}
		if err := w.TrackedWriteCloser.Close(); err != nil {
			return err
		}
		delete(fc.writers.open, fileKey)
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

func (fc *fileController) atDescriptorLimit() bool {
	fc.writers.RLock()
	fc.readers.RLock()
	defer func() {
		fc.readers.RUnlock()
		fc.writers.RUnlock()
	}()
	readerCount := 0
	for _, f := range fc.readers.files {
		f.RLock()
		readerCount += len(f.open)
		f.RUnlock()
	}
	return readerCount+len(fc.writers.open) >= fc.MaxDescriptors
}

func (fc *fileController) close() error {
	fc.writers.RLock()
	fc.readers.RLock()
	defer func() {
		fc.readers.RUnlock()
		fc.writers.RUnlock()
	}()
	c := errors.NewCatcher(errors.WithAggregation())
	for _, w := range fc.writers.open {
		c.Exec(func() error {
			if !w.tryAcquire() {
				return newErrResourceInUse("writer", w.fileKey)
			}
			return w.HardClose()
		})
	}
	for _, f := range fc.readers.files {
		f.Lock()
		for _, r := range f.open {
			c.Exec(func() error {
				if !r.tryAcquire() {
					return newErrResourceInUse("reader", r.fileKey)
				}
				return r.HardClose()
			})
		}
		f.Unlock()
	}
	c.Exec(fc.counterFile.Close)
	return c.Error()
}

type controlledWriter struct {
	controllerEntry
	xio.TrackedWriteCloser
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
	if err := c.controllerEntry.Close(); err != nil {
		return err
	}

	return c.TrackedWriteCloser.Close()
}

type controlledReader struct {
	controllerEntry
	xio.ReaderAtCloser
}

func (c *controlledReader) Close() error {
	return c.controllerEntry.Close()
}

func (c *controlledReader) HardClose() error {
	if err := c.controllerEntry.Close(); err != nil {
		return err
	}

	return c.ReaderAtCloser.Close()
}

type controllerEntry struct {
	ins     alamos.Instrumentation
	inUse   *atomic.Bool
	fileKey uint16
	release chan struct{}
}

func newPoolEntry(key uint16, release chan struct{}, ins alamos.Instrumentation) controllerEntry {
	ce := controllerEntry{
		ins:     ins,
		release: release,
		fileKey: key,
		inUse:   &atomic.Bool{},
	}
	ce.inUse.Store(true)
	return ce
}

func (ce *controllerEntry) Close() error {
	if !ce.inUse.CompareAndSwap(true, false) {
		ce.ins.L.DPanic(fmt.Sprintf("controller: entry %d already closed", ce.fileKey))
		return nil
	}
	select {
	case ce.release <- struct{}{}:
	default:
	}
	return nil
}

func (ce *controllerEntry) tryAcquire() bool {
	return ce.inUse.CompareAndSwap(false, true)
}
