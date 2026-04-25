// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fs

import "sync"

// Op categorizes a recorded filesystem event.
type Op string

const (
	// OpOpen is recorded for every successful Open call on a Recorder or any FS
	// derived from it via Sub.
	OpOpen Op = "open"
	// OpReadAt is recorded for every ReadAt call on a File returned by a
	// Recorder-wrapped FS.
	OpReadAt Op = "read_at"
	// OpRead is recorded for every Read call on a File returned by a
	// Recorder-wrapped FS.
	OpRead Op = "read"
	// OpWriteAt is recorded for every WriteAt call on a File returned by a
	// Recorder-wrapped FS.
	OpWriteAt Op = "write_at"
	// OpWrite is recorded for every Write call on a File returned by a
	// Recorder-wrapped FS.
	OpWrite Op = "write"
)

// Event is a single recorded filesystem event. Length is the requested byte
// length passed by the caller, not the number of bytes actually transferred,
// so the event reflects intent and is recorded even if the underlying call
// returned a short read or an error.
type Event struct {
	// Op is the kind of operation that occurred.
	Op Op
	// Name is the file path the operation targeted, relative to the FS the
	// caller used. When the event was produced via a Sub-derived FS, Name is
	// relative to that sub-FS, not to the parent Recorder.
	Name string
	// Offset is the byte offset for ReadAt and WriteAt; zero for other ops.
	Offset int64
	// Length is the requested byte length for read and write ops; zero for Open.
	Length int
}

// Recorder wraps an FS and records every Open performed against it as well as
// every Read, ReadAt, Write, and WriteAt performed against files it returns.
// FS instances derived via Sub share the parent's event log, so events from
// every branch of the wrapped tree appear in a single ordered timeline.
//
// Recorder is intended for use in tests that need to assert on the I/O
// behavior of a component without introducing test-only hooks into production
// code. It is safe for concurrent use.
type Recorder struct {
	// FS is the underlying filesystem the Recorder delegates to. Methods that
	// the Recorder does not need to instrument (Sub aside, all non-Open methods)
	// pass through to this FS unchanged via promoted embedding.
	FS
	// log is the shared append-only event log this Recorder writes to. The same
	// log pointer is shared across the parent Recorder and every sub-FS derived
	// from it via Sub, so events from every branch appear in a single timeline.
	log *eventLog
}

// NewRecorder returns a Recorder that delegates to inner and records every
// Open call and every read or write performed on files returned by it.
func NewRecorder(inner FS) *Recorder {
	return &Recorder{FS: inner, log: &eventLog{}}
}

// Events returns a copy of every event recorded so far, in the order they
// occurred.
func (r *Recorder) Events() []Event { return r.log.events() }

// EventsFor returns a copy of every event recorded for the given file name,
// in the order they occurred.
func (r *Recorder) EventsFor(name string) []Event {
	out := make([]Event, 0)
	for _, e := range r.log.events() {
		if e.Name == name {
			out = append(out, e)
		}
	}
	return out
}

// Reset clears every event recorded by this Recorder and by every sub-FS
// derived from it.
func (r *Recorder) Reset() { r.log.reset() }

// Open opens name on the underlying FS. On success it records an OpOpen event
// for name and returns a File that records every Read, ReadAt, Write, and
// WriteAt performed against it into this Recorder's event log. Failed opens
// return the inner error and are not recorded.
func (r *Recorder) Open(name string, flag int) (File, error) {
	f, err := r.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	r.log.record(Event{Op: OpOpen, Name: name})
	return &recordingFile{File: f, name: name, log: r.log}, nil
}

// Sub returns a Recorder rooted at name within the underlying FS. The returned
// Recorder shares this Recorder's event log, so events generated through it
// appear in the same timeline as events generated directly through the parent.
func (r *Recorder) Sub(name string) (FS, error) {
	inner, err := r.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return &Recorder{FS: inner, log: r.log}, nil
}

// eventLog is the shared append-only event store backing one or more
// Recorders. A single log is shared by a parent Recorder and every sub-FS
// derived from it via Sub.
type eventLog struct {
	// mu guards buffer; held briefly during record, events, and reset so the
	// log is safe to use from concurrent readers, writers, and observers.
	mu sync.Mutex
	// buffer is the recorded events in insertion order.
	buffer []Event
}

func (l *eventLog) record(e Event) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buffer = append(l.buffer, e)
}

func (l *eventLog) events() []Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Event, len(l.buffer))
	copy(out, l.buffer)
	return out
}

func (l *eventLog) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buffer = nil
}

// recordingFile wraps a File returned by Recorder.Open so that every read or
// write performed against it is recorded into the Recorder's event log.
// Methods the Recorder does not instrument (Close, Truncate, Stat, Sync) pass
// through to the embedded File unchanged.
type recordingFile struct {
	// File is the underlying file the recordingFile delegates to.
	File
	// name is the path the file was opened with, used as the Name on every
	// recorded Event so callers can scope queries to a single file.
	name string
	// log is the shared event log this file records into; the same pointer
	// held by the Recorder that produced this file.
	log *eventLog
}

func (f *recordingFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := f.File.ReadAt(p, off)
	f.log.record(Event{Op: OpReadAt, Name: f.name, Offset: off, Length: len(p)})
	return n, err
}

func (f *recordingFile) Read(p []byte) (int, error) {
	n, err := f.File.Read(p)
	f.log.record(Event{Op: OpRead, Name: f.name, Length: len(p)})
	return n, err
}

func (f *recordingFile) WriteAt(p []byte, off int64) (int, error) {
	n, err := f.File.WriteAt(p, off)
	f.log.record(Event{Op: OpWriteAt, Name: f.name, Offset: off, Length: len(p)})
	return n, err
}

func (f *recordingFile) Write(p []byte) (int, error) {
	n, err := f.File.Write(p)
	f.log.record(Event{Op: OpWrite, Name: f.name, Length: len(p)})
	return n, err
}

var (
	_ FS   = (*Recorder)(nil)
	_ File = (*recordingFile)(nil)
)
