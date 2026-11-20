// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment_test

import (
	"sync"

	xfs "github.com/synnaxlabs/x/io/fs"
)

// opFS records the opens, closes, syncs, and directory syncs of its files, counts the
// files open at once, and can hold or fail a sync.
type opFS struct {
	xfs.FS
	mu       sync.Mutex
	ops      []string
	open     int
	maxOpen  int
	syncErr  map[string]error
	dirErr   error
	inSync   int
	maxSyncs int
	// held receives once for each sync that waits, while release is set.
	held    chan string
	release chan struct{}
}

func newOpFS(fs xfs.FS) *opFS {
	return &opFS{FS: fs, syncErr: map[string]error{}, held: make(chan string)}
}

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

// MaxOpen returns the most files that were open at once.
func (f *opFS) MaxOpen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.maxOpen
}

// MaxSyncs returns the most file and directory syncs that ran at once.
func (f *opFS) MaxSyncs() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.maxSyncs
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

// FailSync makes every later sync of name fail with err, or pass when err is nil.
func (f *opFS) FailSync(name string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.syncErr[name] = err
}

// FailSyncDir makes every later directory sync fail with err, or pass when err is nil.
func (f *opFS) FailSyncDir(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dirErr = err
}

func (f *opFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.ops = append(f.ops, "open "+name)
	f.open++
	f.maxOpen = max(f.maxOpen, f.open)
	f.mu.Unlock()
	return &opFile{File: file, fs: f, name: name}, nil
}

// sync runs one sync of name, which holds and fails as set, and records op.
func (f *opFS) sync(name, op string, err error, do func() error) error {
	f.mu.Lock()
	release := f.release
	f.inSync++
	f.maxSyncs = max(f.maxSyncs, f.inSync)
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.inSync--
		f.mu.Unlock()
	}()
	if release != nil {
		f.held <- name
		<-release
	}
	if err != nil {
		return err
	}
	f.record(op)
	return do()
}

func (f *opFS) SyncDir(name string) error {
	f.mu.Lock()
	err := f.dirErr
	f.mu.Unlock()
	return f.sync(
		name,
		"syncdir "+name,
		err,
		func() error { return f.FS.SyncDir(name) },
	)
}

type opFile struct {
	xfs.File
	fs   *opFS
	name string
}

func (f *opFile) Sync() error {
	f.fs.mu.Lock()
	err := f.fs.syncErr[f.name]
	f.fs.mu.Unlock()
	return f.fs.sync(f.name, "sync "+f.name, err, f.File.Sync)
}

func (f *opFile) Close() error {
	f.fs.mu.Lock()
	f.fs.ops = append(f.fs.ops, "close "+f.name)
	f.fs.open--
	f.fs.mu.Unlock()
	return f.File.Close()
}
