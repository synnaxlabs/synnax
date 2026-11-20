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
	"fmt"
	"sync"

	xfs "github.com/synnaxlabs/x/io/fs"
)

// opFS records the writes and syncs of its files, and can hold or fail a sync.
type opFS struct {
	xfs.FS
	mu       sync.Mutex
	ops      []string
	syncErr  error
	writeErr error
	// held receives once for each sync that waits, while release is set.
	held    chan struct{}
	release chan struct{}
}

func newOpFS(fs xfs.FS) *opFS { return &opFS{FS: fs, held: make(chan struct{})} }

func (f *opFS) record(op string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ops = append(f.ops, op)
}

// Ops returns the recorded operations.
func (f *opFS) Ops() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.ops...)
}

// Hold makes every later sync wait until a send on the returned channel.
func (f *opFS) Hold() chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.release = make(chan struct{})
	return f.release
}

// Unhold lets every later sync return at once.
func (f *opFS) Unhold() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.release = nil
}

func (f *opFS) FailSync(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.syncErr = err
}

func (f *opFS) FailWrite(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writeErr = err
}

func (f *opFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return &opFile{File: file, fs: f, name: name}, nil
}

func (f *opFS) SyncDir(name string) error {
	f.record("syncdir " + name)
	return f.FS.SyncDir(name)
}

type opFile struct {
	xfs.File
	fs   *opFS
	name string
}

func (f *opFile) WriteAt(p []byte, off int64) (int, error) {
	f.fs.mu.Lock()
	err := f.fs.writeErr
	f.fs.mu.Unlock()
	if err != nil {
		return 0, err
	}
	f.fs.record(fmt.Sprintf("write %s %d %d", f.name, off, len(p)))
	return f.File.WriteAt(p, off)
}

func (f *opFile) Sync() error {
	f.fs.mu.Lock()
	release, err := f.fs.release, f.fs.syncErr
	f.fs.mu.Unlock()
	if release != nil {
		f.fs.held <- struct{}{}
		<-release
	}
	if err != nil {
		return err
	}
	f.fs.record("sync " + f.name)
	return f.File.Sync()
}
