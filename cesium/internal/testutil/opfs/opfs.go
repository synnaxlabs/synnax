// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package opfs

import (
	"fmt"
	"sync"

	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
)

// FS wraps a file system, records each operation of the kinds that New names, and
// can fail or hold a sync. It is safe for concurrent use.
type FS struct {
	xfs.FS
	record   set.Set[string]
	mu       sync.Mutex
	ops      []string
	sizes    []int
	open     int
	maxOpen  int
	inSync   int
	maxSyncs int
	syncErr  map[string]error
	dirErr   error
	writeErr error
	// held receives the name of each sync that waits, while release is set.
	held    chan string
	release chan struct{}
}

// New returns an FS over fs that records each operation of the kinds in record:
// "open", "close", "write", "sync", or "syncdir". It panics on another kind.
func New(fs xfs.FS, record ...string) *FS {
	for _, kind := range record {
		switch kind {
		case "open", "close", "write", "sync", "syncdir":
		default:
			panic("opfs: an FS cannot record " + kind)
		}
	}
	return &FS{
		FS:      fs,
		record:  set.New(record...),
		syncErr: map[string]error{},
		held:    make(chan string),
	}
}

// add records op of kind when f records that kind. The caller holds f.mu.
func (f *FS) add(kind, op string) {
	if f.record.Contains(kind) {
		f.ops = append(f.ops, kind+" "+op)
	}
}

// Ops returns the recorded operations in order, each as its kind and its file name, as
// in "sync a.seg". A write also holds its offset and length: "write a.seg 0 512". A
// sync or write that FailSync, FailSyncDir, or FailWrite fails records nothing.
func (f *FS) Ops() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.ops...)
}

// Sizes returns the length of each write, also when f does not record writes.
func (f *FS) Sizes() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.sizes...)
}

// MaxOpen returns the most files that were open at once.
func (f *FS) MaxOpen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.maxOpen
}

// MaxSyncs returns the most file and directory syncs that ran at once.
func (f *FS) MaxSyncs() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.maxSyncs
}

// Hold makes every later file and directory sync send its name on Held, then wait for a
// send on the returned channel, or for its close.
func (f *FS) Hold() chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.release = make(chan struct{})
	return f.release
}

// Unhold lets every later sync return at once.
func (f *FS) Unhold() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.release = nil
}

// Held returns the channel that receives the name of each held sync.
func (f *FS) Held() <-chan string { return f.held }

// FailSync makes every later sync of the file name fail with err, or pass when err is
// nil.
func (f *FS) FailSync(name string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.syncErr[name] = err
}

// FailSyncDir makes every later directory sync fail with err, or pass when err is nil.
func (f *FS) FailSyncDir(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dirErr = err
}

// FailWrite makes every later write fail with err and write nothing, or pass when err
// is nil.
func (f *FS) FailWrite(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writeErr = err
}

// Sub panics, since a file system of a subdirectory would skip the recorder, the
// holds, and the failures of f.
func (f *FS) Sub(name string) (xfs.FS, error) {
	panic("opfs: an FS cannot give the file system of " + name)
}

// Open opens the file name of the wrapped file system, and records it.
func (f *FS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.add("open", name)
	f.open++
	f.maxOpen = max(f.maxOpen, f.open)
	f.mu.Unlock()
	return &opFile{File: file, fs: f, name: name}, nil
}

// SyncDir syncs the directory name of the wrapped file system, held and failed as set.
func (f *FS) SyncDir(name string) error {
	f.mu.Lock()
	err := f.dirErr
	f.mu.Unlock()
	return f.sync(name, "syncdir", err, func() error { return f.FS.SyncDir(name) })
}

// sync runs one sync of name, held and failed as set, and records it as kind when err
// is nil.
func (f *FS) sync(name, kind string, err error, do func() error) error {
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
	f.mu.Lock()
	f.add(kind, name)
	f.mu.Unlock()
	return do()
}

type opFile struct {
	xfs.File
	fs   *FS
	name string
}

func (f *opFile) WriteAt(p []byte, off int64) (int, error) {
	f.fs.mu.Lock()
	if err := f.fs.writeErr; err != nil {
		f.fs.mu.Unlock()
		return 0, err
	}
	f.fs.sizes = append(f.fs.sizes, len(p))
	f.fs.add("write", fmt.Sprintf("%s %d %d", f.name, off, len(p)))
	f.fs.mu.Unlock()
	return f.File.WriteAt(p, off)
}

func (f *opFile) Sync() error {
	f.fs.mu.Lock()
	err := f.fs.syncErr[f.name]
	f.fs.mu.Unlock()
	return f.fs.sync(f.name, "sync", err, f.File.Sync)
}

func (f *opFile) Close() error {
	f.fs.mu.Lock()
	f.fs.add("close", f.name)
	f.fs.open--
	f.fs.mu.Unlock()
	return f.File.Close()
}
