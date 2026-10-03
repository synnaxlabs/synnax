// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package timer

import (
	"time"

	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sys/unix"
)

// platform waits on a kqueue that holds a one-shot timer and the event that Close
// triggers. A kqueue cannot be non-blocking, so the wait goroutine blocks its thread in
// kevent.
type platform struct {
	waiter
	// kq is the kqueue.
	kq int
}

func (t *Timer) open() error {
	kq, err := unix.Kqueue()
	if err != nil {
		return errors.Wrap(err, "failed to create kqueue")
	}
	t.kq = kq
	if err = t.change(unix.Kevent_t{
		Filter: unix.EVFILT_USER,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
	}); err != nil {
		return errors.Join(
			errors.Wrap(err, "failed to register timer close event"),
			unix.Close(kq),
		)
	}
	t.done = make(chan struct{})
	go t.wait()
	return nil
}

func (t *Timer) wait() {
	// The last fire wakes the receiver after done closes, so its next Reset or Stop
	// returns err.
	defer t.fire()
	defer close(t.done)
	events := make([]unix.Kevent_t, 2)
	for {
		n, err := unix.Kevent(t.kq, nil, events, nil)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			t.err = errors.Wrap(err, "failed to wait on kqueue")
			return
		}
		for _, e := range events[:n] {
			if e.Filter == unix.EVFILT_USER {
				return
			}
		}
		t.fire()
	}
}

// Reset stops the timer and starts it again, so it fires after d. A d of zero or less
// fires right away.
func (t *Timer) Reset(d time.Duration) error {
	if d <= 0 {
		err := t.Stop()
		t.fire()
		return err
	}
	if err := t.failure(); err != nil {
		return err
	}
	t.drain()
	// Adding the timer again replaces it and drops a fire that was not read.
	// NOTE_CRITICAL removes the slack of up to 1 ms that the kernel adds to group
	// timers, which a time.Timer cannot remove.
	return errors.Wrap(t.change(unix.Kevent_t{
		Filter: unix.EVFILT_TIMER,
		Flags:  unix.EV_ADD | unix.EV_ONESHOT,
		Fflags: unix.NOTE_NSECONDS | unix.NOTE_CRITICAL,
		Data:   d.Nanoseconds(),
	}), "failed to arm kqueue timer")
}

// Stop stops the timer and drops a pending fire.
func (t *Timer) Stop() error {
	if err := t.failure(); err != nil {
		return err
	}
	t.drain()
	err := t.change(unix.Kevent_t{Filter: unix.EVFILT_TIMER, Flags: unix.EV_DELETE})
	// ENOENT means no timer is set: it already fired, or was never set.
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	return errors.Wrap(err, "failed to stop kqueue timer")
}

// Close stops the timer and releases its kqueue.
func (t *Timer) Close() error {
	if err := t.change(unix.Kevent_t{
		Filter: unix.EVFILT_USER,
		Fflags: unix.NOTE_TRIGGER,
	}); err != nil {
		return errors.Wrap(err, "failed to signal timer close event")
	}
	<-t.done
	return errors.Join(t.err, unix.Close(t.kq))
}

// change applies one change to the kqueue.
func (t *Timer) change(e unix.Kevent_t) error {
	_, err := unix.Kevent(t.kq, []unix.Kevent_t{e}, nil, nil)
	return err
}
