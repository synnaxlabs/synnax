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

// advance switches to the next log when its file is ready: it writes and syncs both
// header copies of it, reports the switch, and asks for the log after it. Until the
// file is ready, the live log goes on to its end limits; after a failed file it asks
// for the file again. It returns the error of a failed log.
func (l *Log) advance() error {
	next := l.gen + 1
	l.mu.Lock()
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
	l.cfg.Switched(next)
	l.cfg.Prepare(next + 1)
	return nil
}
