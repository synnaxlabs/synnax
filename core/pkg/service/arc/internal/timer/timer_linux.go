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
	"os"
	"time"

	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sys/unix"
)

// platform reads a non-blocking timerfd through the Go netpoller. The netpoller wakes
// on the fd at the timerfd's precision, not at its own 1 ms timeout precision.
type platform struct {
	// fd is the timerfd. It stays separate from file because File.Fd makes the fd
	// blocking.
	fd int
	// file reads fd through the netpoller.
	file *os.File
	// done closes when the read goroutine exits.
	done chan struct{}
	// err is the read goroutine's failure. It is safe to read after done closes.
	err error
}

func (t *Timer) open() error {
	fd, err := unix.TimerfdCreate(
		unix.CLOCK_MONOTONIC,
		unix.TFD_NONBLOCK|unix.TFD_CLOEXEC,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create timerfd")
	}
	t.fd = fd
	t.file = os.NewFile(uintptr(fd), "timerfd")
	t.done = make(chan struct{})
	go t.read()
	return nil
}

func (t *Timer) read() {
	defer close(t.done)
	buf := make([]byte, 8)
	for {
		if _, err := t.file.Read(buf); err != nil {
			if !errors.Is(err, os.ErrClosed) {
				t.err = errors.Wrap(err, "failed to read timerfd")
			}
			// Wakes the receiver, so its next Reset or Stop returns the error.
			t.fire()
			return
		}
		t.fire()
	}
}

// Reset stops the timer and starts it again, so it fires after d. A d of zero or less
// fires right away.
func (t *Timer) Reset(d time.Duration) error {
	if err := t.Stop(); err != nil {
		return err
	}
	if d <= 0 {
		t.fire()
		return nil
	}
	return t.arm(d)
}

// Stop stops the timer and drops a pending fire.
func (t *Timer) Stop() error {
	select {
	case <-t.done:
		return t.err
	default:
	}
	t.drain()
	return t.arm(0)
}

// arm starts the timerfd, or stops it when d is zero.
func (t *Timer) arm(d time.Duration) error {
	spec := unix.ItimerSpec{Value: unix.NsecToTimespec(d.Nanoseconds())}
	return errors.Wrap(
		unix.TimerfdSettime(t.fd, 0, &spec, nil),
		"failed to arm timerfd",
	)
}

// Close stops the timer and releases the timerfd.
func (t *Timer) Close() error {
	err := t.file.Close()
	<-t.done
	return errors.Join(err, t.err)
}
