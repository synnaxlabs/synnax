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
	return nil
}

func (t *Timer) wait() error {
	buf := make([]byte, 8)
	for {
		if _, err := t.file.Read(buf); err != nil {
			if errors.Is(err, os.ErrClosed) {
				return nil
			}
			return errors.Wrap(err, "failed to read timerfd")
		}
		if err := t.expire(); err != nil {
			return err
		}
	}
}

func (t *Timer) arm(d time.Duration) error {
	spec := unix.ItimerSpec{Value: unix.NsecToTimespec(d.Nanoseconds())}
	return errors.Wrap(
		unix.TimerfdSettime(t.fd, 0, &spec, nil),
		"failed to arm timerfd",
	)
}

func (t *Timer) disarm() error { return t.arm(0) }

func (t *Timer) close() error {
	err := t.file.Close()
	<-t.done
	return errors.Join(err, t.err)
}
