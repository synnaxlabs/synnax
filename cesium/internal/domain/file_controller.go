// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"io"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

const extension = ".domain"

func fileKeyToName(key uint16) string {
	return strconv.Itoa(int(key)) + extension
}

type fileController struct {
	Config
	writers struct {
		sync.RWMutex
		open     map[uint16]controlledWriter
		unopened map[uint16]struct{}
	}
	readers struct {
		sync.RWMutex
		open map[uint16][]controlledReader
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
	fc.readers.open = make(map[uint16][]controlledReader)
	fc.release = make(chan struct{}, cfg.MaxDescriptors)

	fc.writers.unopened, err = fc.scanUnopenedFiles()
	return fc, err
}

func (fc *fileController) scanUnopenedFiles() (map[uint16]struct{}, error) {
	unopened := make(map[uint16]struct{})
	for i := 1; i <= int(fc.counter.Value()); i++ {
		e, err := fc.Config.FS.Exists(fileKeyToName(uint16(i)))
		if err != nil {
			return unopened, err
		}
		if !e {
			continue
		}

		s, err := fc.Config.FS.Stat(fileKeyToName(uint16(i)))
		if err != nil {
			return unopened, err
		}
		if s.Size() < int64(fc.FileSize) {
			unopened[uint16(i)] = struct{}{}
		}
	}

	return unopened, nil
}

func (fc *fileController) acquireWriter(ctx context.Context) (uint16, int64, xio.TrackedWriteCloser, error) {
	ctx, span := fc.T.Bench(ctx, "acquireWriter")
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

	<-fc.release
	return fc.acquireWriter(ctx)
}

// newWriter creates a new writing file handle from the file controller: it first
// attempts to create a file handle for files from the directory that are not at
// capacity. If there is none, it creates a new file and increments the counter.
func (fc *fileController) newWriter(ctx context.Context) (*controlledWriter, int64, error) {
	ctx, span := fc.T.Bench(ctx, "newWriter")
	fc.writers.Lock()

	defer func() {
		fc.writers.Unlock()
		span.End()
	}()

	for key := range fc.writers.unopened {
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
			controllerEntry:    newPoolEntry(key, fc.release),
		}
		fc.writers.open[key] = w
		delete(fc.writers.unopened, key)

		s, err := file.Stat()
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
		controllerEntry:    newPoolEntry(nextKey, fc.release),
	}
	fc.writers.open[nextKey] = w
	return &w, 0, nil
}

func (fc *fileController) acquireReader(ctx context.Context, key uint16) (xio.ReaderAtCloser, error) {
	ctx, span := fc.T.Bench(ctx, "acquireReader")
	defer span.End()

	fc.readers.RLock()
	if rs, ok := fc.readers.open[key]; ok {
		for _, r := range rs {
			if r.tryAcquire() {
				fc.readers.RUnlock()
				return &r, nil
			}
		}

		if !fc.atDescriptorLimit() {
			fc.readers.RUnlock()
			return fc.newReader(ctx, key)
		}

		<-fc.release
		fc.readers.RUnlock()
		return fc.acquireReader(ctx, key)
	}

	if !fc.atDescriptorLimit() {
		fc.readers.RUnlock()
		return fc.newReader(ctx, key)
	}

	ok, err := fc.gcReaders()
	if err != nil {
		return nil, err
	}
	fc.readers.RUnlock()
	if ok {
		return fc.acquireReader(ctx, key)
	}
	<-fc.release
	return fc.acquireReader(ctx, key)
}

func (fc *fileController) newReader(ctx context.Context, key uint16) (*controlledReader, error) {
	ctx, span := fc.T.Bench(ctx, "newReader")
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
		controllerEntry: newPoolEntry(key, fc.release),
	}
	fc.readers.Lock()
	fc.readers.open[key] = append(fc.readers.open[key], r)
	fc.readers.Unlock()
	return &r, err
}

func (fc *fileController) gcReaders() (bool, error) {
	fc.readers.Lock()
	defer fc.readers.Unlock()
	for k, v := range fc.readers.open {
		for i, r := range v {
			if r.tryAcquire() {
				err := r.Close()
				err = errors.CombineErrors(err, r.ReaderAtCloser.Close())
				fc.readers.open[k] = append(v[:i], v[i+1:]...)
				return true, err
			}
		}
	}
	return false, nil
}

// gcWriters closes all open writers that are not currently being written to
func (fc *fileController) gcWriters() (bool, error) {
	fc.writers.Lock()
	defer fc.writers.Unlock()
	for k, w := range fc.writers.open {
		if w.tryAcquire() {
			err := w.Close()
			err = errors.CombineErrors(err, w.TrackedWriteCloser.Close())
			delete(fc.writers.open, k)
			return true, err
		}
	}
	return false, nil
}

func (fc *fileController) atDescriptorLimit() bool {
	fc.writers.RLock()
	fc.readers.RLock()
	defer func() {
		fc.readers.RUnlock()
		fc.writers.RUnlock()
	}()
	return len(fc.readers.open)+len(fc.writers.open) >= fc.MaxDescriptors
}

func (fc *fileController) removeReadersWriters(ctx context.Context, key uint16) error {
	ctx, span := fc.T.Bench(ctx, "removeReadersWriters")
	defer span.End()

	fc.readers.RLock()
	_, ok := fc.readers.open[key]
	if !ok {
		return nil
	}

	c := errors.NewCatcher(errors.WithAggregation())
	for _, r := range fc.readers.open[key] {
		if r.tryAcquire() {
			c.Exec(r.Close)
		}
		c.Exec(r.ReaderAtCloser.Close)
	}

	fc.readers.RUnlock()
	fc.readers.Lock()
	delete(fc.readers.open, key)
	fc.readers.Unlock()

	w, ok := fc.writers.open[key]
	if !ok {
		return c.Error()
	}

	fc.writers.RLock()
	if w.tryAcquire() {
		c.Exec(w.Close)
	}
	c.Exec(w.TrackedWriteCloser.Close)

	fc.writers.RUnlock()
	fc.writers.Lock()
	delete(fc.writers.open, key)
	fc.writers.Unlock()

	return c.Error()
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
		c.Exec(w.TrackedWriteCloser.Close)
	}
	for _, v := range fc.readers.open {
		for _, r := range v {
			c.Exec(r.ReaderAtCloser.Close)
		}
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
		c.TrackedWriteCloser.Reset()
	}
	return acquired
}

func (c *controlledWriter) Close() error { return c.controllerEntry.Close() }

type controlledReader struct {
	controllerEntry
	xio.ReaderAtCloser
}

func (c *controlledReader) Close() error {
	return c.controllerEntry.Close()
}

// flag specifies whether the reader/writer is currently in use
type controllerEntry struct {
	flag    *atomic.Bool
	fileKey uint16
	release chan struct{}
}

func newPoolEntry(key uint16, release chan struct{}) controllerEntry {
	ce := controllerEntry{
		release: release,
		fileKey: key,
		flag:    &atomic.Bool{},
	}
	ce.flag.Store(true)
	return ce
}

func (ce *controllerEntry) Close() error {
	if !ce.flag.CompareAndSwap(true, false) {
		panic("controller: entry already closed")
	}
	select {
	case ce.release <- struct{}{}:
	default:
	}
	return nil
}

func (ce *controllerEntry) tryAcquire() bool {
	return ce.flag.CompareAndSwap(false, true)
}
