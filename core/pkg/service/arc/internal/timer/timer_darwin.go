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

// DefaultSpin is the spin that holds a wait to its deadline at the lowest CPU cost.
const DefaultSpin time.Duration = 0

// platform waits on a kqueue that holds a one-shot timer and the event that Close
// triggers. A kqueue cannot be non-blocking, so the wait goroutine blocks its thread in
// kevent.
type platform struct {
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
	return nil
}

func (t *Timer) wait() error {
	events := make([]unix.Kevent_t, 2)
	for {
		n, err := unix.Kevent(t.kq, nil, events, nil)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return errors.Wrap(err, "failed to wait on kqueue")
		}
		for _, e := range events[:n] {
			if e.Filter == unix.EVFILT_USER {
				return nil
			}
		}
		if err = t.expire(); err != nil {
			return err
		}
	}
}

func (t *Timer) arm(d time.Duration) error {
	// Adding the timer again replaces it. NOTE_CRITICAL removes the slack of up to 1 ms
	// that the kernel adds to group timers, which a time.Timer cannot remove.
	return errors.Wrap(t.change(unix.Kevent_t{
		Filter: unix.EVFILT_TIMER,
		Flags:  unix.EV_ADD | unix.EV_ONESHOT,
		Fflags: unix.NOTE_NSECONDS | unix.NOTE_CRITICAL,
		Data:   d.Nanoseconds(),
	}), "failed to arm kqueue timer")
}

func (t *Timer) disarm() error {
	err := t.change(unix.Kevent_t{Filter: unix.EVFILT_TIMER, Flags: unix.EV_DELETE})
	// ENOENT means no timer is set: it already fired, or was never set.
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	return errors.Wrap(err, "failed to stop kqueue timer")
}

func (t *Timer) close() error {
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
