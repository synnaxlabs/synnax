// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import "github.com/synnaxlabs/x/errors"

// Prepared reports that the file of the log of generation next is ready, or the error
// that kept it from being made. The log switches to a ready file before a frame would
// pass three quarters of the live log.
func (l *Log) Prepared(next uint64, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mu.asked = false
	if err != nil {
		l.mu.prepareErr = errors.Wrapf(err, "the file of log %d is not ready", next)
	} else {
		l.mu.prepared, l.mu.prepareErr = max(l.mu.prepared, next), nil
	}
	l.ready.Broadcast()
}

// advance switches to the next log once its file is ready: it writes and syncs both
// header copies of it, writes the Seals of open segments that no writer holds as its
// first entry, reports the switch, and asks for the log after it. It waits for the
// file. When the file failed, or the log closes first, it stays in the live log, whose
// end limits then apply, and returns nil; after a failure it asks for the file again.
// It returns the error of a failed log.
func (l *Log) advance() error {
	next := l.gen + 1
	l.mu.Lock()
	for l.mu.prepared < next && l.mu.prepareErr == nil && !l.mu.closing &&
		l.mu.err == nil {
		l.ready.Wait()
	}
	failed, ready := l.mu.err, l.mu.prepared >= next
	retry := !ready && l.mu.prepareErr != nil && !l.mu.asked
	if retry {
		l.mu.asked = true
	}
	l.mu.Unlock()
	switch {
	case failed != nil:
		return failed
	case retry:
		l.cfg.Prepare(next)
		return nil
	case !ready:
		return nil
	}
	old := l.file
	if err := l.open(next); err != nil {
		l.fail(err)
		return l.failure()
	}
	if err := old.Close(); err != nil {
		l.fail(err)
		return l.failure()
	}
	if err := l.writeSeals(); err != nil {
		return err
	}
	l.cfg.Switched(next)
	l.cfg.Prepare(next + 1)
	return nil
}

// writeSeals writes the Seals of open segments that no writer holds as the first entry
// of the new log.
func (l *Log) writeSeals() error {
	records, err := l.cfg.Seals()
	if err != nil {
		l.fail(err)
		return l.failure()
	}
	if len(records) == 0 {
		return nil
	}
	var e Entry
	for _, r := range records {
		e.Add(r)
	}
	l.frame = Frame{Count: 1, Sequence: l.seq, PrevCRC: l.prevCRC, Entries: e.Bytes()}
	sequence := l.seq
	if err = l.writeFrame(l.nextStart()); err != nil {
		return err
	}
	l.cfg.OnDurable(sequence, e.records)
	l.needMark = true
	return nil
}
