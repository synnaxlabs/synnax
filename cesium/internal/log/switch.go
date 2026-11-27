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
	l.mu.asked = false
	if err != nil {
		l.mu.prepareErr = errors.Wrapf(err, "the file of log %d is not ready", next)
	} else {
		l.mu.prepared, l.mu.prepareErr = max(l.mu.prepared, next), nil
	}
	l.mu.Unlock()
}

// nextReady reports whether the file of the next log is ready. Until it is, the live
// log goes on to its end limits; after a failed file it asks for the file again. It
// returns the error of a failed log.
func (l *Log) nextReady() (bool, error) {
	next := l.gen + 1
	l.mu.Lock()
	failed, ready := l.mu.err, l.mu.prepared >= next
	retry := !ready && l.mu.prepareErr != nil && !l.mu.asked
	if retry {
		l.mu.asked = true
	}
	l.mu.Unlock()
	if failed != nil {
		return false, failed
	}
	if retry {
		l.cfg.Prepare(next)
	}
	return ready, nil
}

// switchIfReady flushes the batch with flush and switches to the next log when its file
// is ready, so a batch past three quarters goes on while it is not. It reports whether
// it switched, and returns the error of a failed log, after a flush.
func (l *Log) switchIfReady(flush func()) (bool, error) {
	ready, err := l.nextReady()
	if err == nil && !ready {
		return false, nil
	}
	flush()
	if err != nil {
		return false, err
	}
	return true, l.advance()
}

// advance switches to the next log, whose file is ready: it writes and syncs both
// header copies of it, reports the switch, and asks for the log after it. It returns
// the error of a failed log.
func (l *Log) advance() error {
	next := l.gen + 1
	old := l.file
	if err := l.open(next); err != nil {
		l.fail(err)
		return l.failure()
	}
	if err := old.Close(); err != nil {
		l.fail(err)
		return l.failure()
	}
	l.cfg.OnSwitched(next)
	l.cfg.Prepare(next + 1)
	return nil
}
