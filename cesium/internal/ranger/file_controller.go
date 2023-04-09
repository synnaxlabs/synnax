// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"github.com/synnaxlabs/alamos"
	atomicx "github.com/synnaxlabs/x/atomic"
	"github.com/synnaxlabs/x/errutil"
	xio "github.com/synnaxlabs/x/io"
	"io"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

const extension = ".ranger"

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
	counter *xio.Int32Counter
}

const counterFile = "counter"

func openFileController(cfg Config) (*fileController, error) {
	counterF, err := cfg.FS.Open(fileName(counterFile), os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	counter, err := xio.NewInt32Counter(counterF, &atomicx.Int32Counter{})
	if err != nil {
		return nil, err
	}
	fc := &fileController{
		Config:  cfg,
		counter: counter,
	}
	fc.writers.open = make([]controlledWriter, 0, cfg.MaxDescriptors)
	fc.writers.release = make(chan struct{}, cfg.MaxDescriptors)
	fc.readers.open = make(map[uint16][]controlledReader)
	fc.readers.release = make(chan struct{}, cfg.MaxDescriptors)
	return fc, nil
}

func (fc *fileController) acquireWriter(ctx context.Context) (uint16, xio.OffsetWriteCloser, error) {
	ctx, span := fc.T.Trace(ctx, "acquireWriter", alamos.InfoLevel)
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
		w, err := fc.newWriter()
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

func (fc *fileController) newWriter() (*controlledWriter, error) {
	nextKey := uint16(fc.counter.Add(1))
	if fc.counter.Error() != nil {
		return nil, fc.counter.Error()
	}
	file, err := fc.FS.Open(
		fileKeyName(nextKey),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
	)
	if err != nil {
		return nil, err
	}
	baseW, err := xio.NewOffsetWriteCloser(file, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	w := controlledWriter{
		OffsetWriteCloser: baseW,
		controllerEntry:   newPoolEntry(nextKey, fc.writers.release),
	}
	fc.writers.Lock()
	fc.writers.open = append(fc.writers.open, w)
	fc.writers.Unlock()
	return &w, nil
}

func (fc *fileController) acquireReader(key uint16) (xio.ReaderAtCloser, error) {
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
		return fc.newReader(key)
	}
	ok, err := fc.gcWriters()
	if err != nil {
		return nil, err
	}
	if ok {
		return fc.acquireReader(key)
	}
	<-fc.readers.release
	return fc.acquireReader(key)
}

func (fc *fileController) newReader(key uint16) (*controlledReader, error) {
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
			err := w.OffsetWriteCloser.Close()
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
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, w := range fc.writers.open {
		c.Exec(w.OffsetWriteCloser.Close)
	}
	for _, v := range fc.readers.open {
		for _, r := range v {
			c.Exec(r.ReaderAtCloser.Close)
		}
	}
	c.Exec(fc.counter.Close)
	return c.Error()
}

type controlledWriter struct {
	controllerEntry
	xio.OffsetWriteCloser
}

func (c *controlledWriter) tryAcquire() bool {
	acquired := c.controllerEntry.tryAcquire()
	if acquired {
		c.OffsetWriteCloser.Reset()
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
