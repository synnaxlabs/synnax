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

func fileName(name string) string {
	return name + extension
}

func fileKeyName(key uint16) string {
	return fileName(strconv.Itoa(int(key)))
}

type fileController struct {
	Config
	writers struct {
		sync.RWMutex
		release chan struct{}
		open    []controlledWriter
	}
	readers struct {
		sync.RWMutex
		release chan struct{}
		open    map[uint16][]controlledReader
	}
	counter     *xio.Int32Counter
	counterFile io.Closer
}

const counterFile = "counter"

func openFileController(cfg Config) (*fileController, error) {
	counterF, err := cfg.FS.Open(fileName(counterFile), os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	fc := &fileController{
		Config:      cfg,
		counter:     xio.NewInt32Counter(counterF),
		counterFile: counterF,
	}
	fc.writers.open = make([]controlledWriter, 0, cfg.MaxDescriptors)
	fc.writers.release = make(chan struct{}, cfg.MaxDescriptors)
	fc.readers.open = make(map[uint16][]controlledReader)
	fc.readers.release = make(chan struct{}, cfg.MaxDescriptors)
	return fc, nil
}

func (fc *fileController) acquireWriter(ctx context.Context) (uint16, xio.TrackedWriteCloser, error) {
	ctx, span := fc.T.Bench(ctx, "acquireWriter")
	defer span.End()
	// attempt to pull a writer from the pool of open writers
	fc.writers.RLock()
	for _, w := range fc.writers.open {
		if w.tryAcquire() {
			fc.writers.RUnlock()
			return w.fileKey, &w, nil
		}
	}
	fc.writers.RUnlock()

	// if we aren't at the descriptor limit, create a new writer
	if !fc.atDescriptorLimit() {
		w, err := fc.newWriter(ctx)
		return w.fileKey, w, span.Error(err)
	}

	// otherwise, do a best effort garbage collection of the readers
	ok, err := fc.gcReaders()
	if err != nil {
		return 0, nil, span.Error(err)
	}
	if ok {
		return fc.acquireWriter(ctx)
	}

	// if we still can't acquire a writer, wait for one to be released
	<-fc.writers.release
	return fc.acquireWriter(ctx)
}

func (fc *fileController) newWriter(ctx context.Context) (*controlledWriter, error) {
	ctx, span := fc.T.Bench(ctx, "newWriter")
	defer span.End()
	nextKey_, err := fc.counter.Add(1)
	if err != nil {
		return nil, span.Error(err)
	}
	nextKey := uint16(nextKey_)
	file, err := fc.FS.Open(
		fileKeyName(nextKey),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
	)
	if err != nil {
		return nil, span.Error(err)
	}
	base, err := xio.NewTrackedWriteCloser(file)
	if err != nil {
		return nil, span.Error(err)
	}
	w := controlledWriter{
		TrackedWriteCloser: base,
		controllerEntry:    newPoolEntry(nextKey, fc.writers.release),
	}
	fc.writers.Lock()
	fc.writers.open = append(fc.writers.open, w)
	fc.writers.Unlock()
	return &w, err
}

func (fc *fileController) acquireReader(ctx context.Context, key uint16) (xio.ReaderAtCloser, error) {
	ctx, span := fc.T.Bench(ctx, "acquireReader")
	defer span.End()
	fc.readers.RLock()
	if opts, ok := fc.readers.open[key]; ok {
		// iterate over the open and find the first available reader
		for _, r := range opts {
			if !r.tryAcquire() {
				fc.readers.RUnlock()
				return &r, nil
			}
		}
	}
	fc.readers.RUnlock()
	if !fc.atDescriptorLimit() {
		return fc.newReader(ctx, key)
	}
	ok, err := fc.gcWriters()
	if err != nil {
		return nil, span.Error(err)
	}
	if ok {
		return fc.acquireReader(ctx, key)
	}
	<-fc.readers.release
	return fc.acquireReader(ctx, key)
}

func (fc *fileController) newReader(ctx context.Context, key uint16) (*controlledReader, error) {
	file, err := fc.FS.Open(
		fileKeyName(key),
		os.O_RDONLY,
	)
	if err != nil {
		return nil, err
	}
	r := controlledReader{
		ReaderAtCloser:  file,
		controllerEntry: newPoolEntry(key, fc.readers.release),
	}
	fc.readers.Lock()
	fc.readers.open[key] = append(fc.readers.open[key], r)
	fc.readers.Unlock()
	return &r, err
}

func (fc *fileController) atDescriptorLimit() bool {
	fc.writers.RLock()
	fc.readers.RLock()
	defer func() {
		fc.writers.RUnlock()
		fc.readers.RUnlock()
	}()
	return len(fc.readers.open)+len(fc.writers.open) >= fc.MaxDescriptors
}

func (fc *fileController) gcReaders() (bool, error) {
	fc.readers.RLock()
	for k, v := range fc.readers.open {
		for i, r := range v {
			if r.tryAcquire() {
				fc.readers.RUnlock()
				err := r.ReaderAtCloser.Close()
				fc.readers.Lock()
				fc.readers.open[k] = append(v[:i], v[i+1:]...)
				fc.readers.Unlock()
				return true, err
			}
		}
	}
	return false, nil
}

func (fc *fileController) gcWriters() (bool, error) {
	fc.writers.RLock()
	for i, w := range fc.writers.open {
		if w.tryAcquire() {
			fc.writers.RUnlock()
			fc.writers.Lock()
			err := w.TrackedWriteCloser.Close()
			fc.writers.open = append(fc.writers.open[:i], fc.writers.open[i+1:]...)
			fc.writers.Unlock()
			return true, err
		}
	}
	return false, nil
}

func (fc *fileController) close() error {
	fc.writers.Lock()
	defer fc.writers.Unlock()
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

func (c *controlledReader) Close() error { return c.controllerEntry.Close() }

type controllerEntry struct {
	flag    *atomic.Bool
	fileKey uint16
	release chan<- struct{}
}

func newPoolEntry(key uint16, release chan<- struct{}) controllerEntry {
	ce := controllerEntry{
		release: release,
		fileKey: key,
		flag:    &atomic.Bool{},
	}
	ce.flag.Store(true)
	return ce
}

func (pe *controllerEntry) Close() error {
	if !pe.flag.CompareAndSwap(true, false) {
		panic("controller: entry already closed")
	}
	select {
	case pe.release <- struct{}{}:
	default:
	}
	return nil
}

func (pe *controllerEntry) tryAcquire() bool {
	return pe.flag.CompareAndSwap(false, true)
}

func (pe *controllerEntry) available() bool { return pe.flag.Load() }
