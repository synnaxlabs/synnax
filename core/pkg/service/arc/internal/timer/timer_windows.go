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
	"sync"
	"time"
	"unsafe"

	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sys/windows"
)

const (
	// spin is how long the timer spins before its deadline. A high resolution timer
	// wakes up to about 0.5 ms late.
	spin = time.Millisecond
	// createWaitableTimerHighResolution is CREATE_WAITABLE_TIMER_HIGH_RESOLUTION.
	createWaitableTimerHighResolution = 0x2
)

var (
	kernel32                      = windows.NewLazySystemDLL("kernel32.dll")
	procQueryPerformanceCounter   = kernel32.NewProc("QueryPerformanceCounter")
	procQueryPerformanceFrequency = kernel32.NewProc("QueryPerformanceFrequency")
	procCreateWaitableTimerExW    = kernel32.NewProc("CreateWaitableTimerExW")
	procSetWaitableTimer          = kernel32.NewProc("SetWaitableTimer")
	procCancelWaitableTimer       = kernel32.NewProc("CancelWaitableTimer")
	// frequency returns the QueryPerformanceCounter counts per second.
	frequency = sync.OnceValue(func() int64 {
		return queryPerformance(procQueryPerformanceFrequency)
	})
)

// queryPerformance calls QueryPerformanceCounter or QueryPerformanceFrequency, which
// never fail on Windows XP or later.
func queryPerformance(proc *windows.LazyProc) int64 {
	var v int64
	if r, _, err := proc.Call(uintptr(unsafe.Pointer(&v))); r == 0 {
		panic(errors.Wrapf(err, "failed to call %s", proc.Name))
	}
	return v
}

// Now returns a reading of QueryPerformanceCounter. Only the span between two readings
// has meaning. Go's own clock on Windows moves only on each system tick.
func Now() time.Duration {
	c, f := queryPerformance(procQueryPerformanceCounter), frequency()
	return time.Duration(c/f*int64(time.Second) + c%f*int64(time.Second)/f)
}

// Wall returns the wall clock time from GetSystemTimePreciseAsFileTime. Go's time.Now
// on Windows moves only on each system tick.
func Wall() time.Time {
	var ft windows.Filetime
	windows.GetSystemTimePreciseAsFileTime(&ft)
	return time.Unix(0, ft.Nanoseconds())
}

// platform waits on a waitable timer that wakes spin before the deadline, then spins
// to the deadline on Now.
type platform struct {
	// handle is the waitable timer.
	handle windows.Handle
	// closed is an event that Close signals to stop the wait goroutine.
	closed windows.Handle
	// tickRaised is true when open raised the system tick, because Windows before 10
	// 1803 has no high resolution timer.
	tickRaised bool
}

func (t *Timer) open() error {
	h, _, _ := procCreateWaitableTimerExW.Call(
		0,
		0,
		createWaitableTimerHighResolution,
		windows.TIMER_ALL_ACCESS,
	)
	// Windows before 10 1803 has no high resolution timer.
	if h == 0 {
		var err error
		if h, _, err = procCreateWaitableTimerExW.Call(
			0,
			0,
			0,
			windows.TIMER_ALL_ACCESS,
		); h == 0 {
			return errors.Wrap(err, "failed to create waitable timer")
		}
		if err = windows.TimeBeginPeriod(1); err != nil {
			return errors.Join(
				errors.Wrap(err, "failed to raise the system tick"),
				windows.CloseHandle(windows.Handle(h)),
			)
		}
		t.tickRaised = true
	}
	t.handle = windows.Handle(h)
	var err error
	if t.closed, err = windows.CreateEvent(nil, 1, 0, nil); err != nil {
		return errors.Join(
			errors.Wrap(err, "failed to create timer close event"),
			t.release(),
		)
	}
	return nil
}

func (t *Timer) wait() error {
	handles := []windows.Handle{t.handle, t.closed}
	for {
		event, err := windows.WaitForMultipleObjects(handles, false, windows.INFINITE)
		if err != nil {
			return errors.Wrap(err, "failed to wait on waitable timer")
		}
		if event == windows.WAIT_OBJECT_0+1 {
			return nil
		}
		t.spinToDeadline()
		if err = t.expire(); err != nil {
			return err
		}
	}
}

// spinToDeadline spins until the deadline, unless the timer is stopped or the deadline
// is more than two spins away.
func (t *Timer) spinToDeadline() {
	for {
		t.mu.Lock()
		deadline := t.deadline
		t.mu.Unlock()
		if left := deadline - Now(); deadline == 0 || left <= 0 || left > 2*spin {
			return
		}
	}
}

func (t *Timer) arm(d time.Duration) error {
	// A negative due time is relative, in 100 ns units.
	due := min(-int64((d-spin)/100), -1)
	r, _, err := procSetWaitableTimer.Call(
		uintptr(t.handle),
		uintptr(unsafe.Pointer(&due)),
		0,
		0,
		0,
		0,
	)
	if r == 0 {
		return errors.Wrap(err, "failed to set waitable timer")
	}
	return nil
}

func (t *Timer) disarm() error {
	if r, _, err := procCancelWaitableTimer.Call(uintptr(t.handle)); r == 0 {
		return errors.Wrap(err, "failed to cancel waitable timer")
	}
	return nil
}

func (t *Timer) close() error {
	if err := windows.SetEvent(t.closed); err != nil {
		return errors.Wrap(err, "failed to signal timer close event")
	}
	<-t.done
	return errors.Join(t.err, windows.CloseHandle(t.closed), t.release())
}

// release closes the waitable timer and lowers the system tick that open raised.
func (t *Timer) release() error {
	err := windows.CloseHandle(t.handle)
	if t.tickRaised {
		err = errors.Join(
			err,
			errors.Wrap(windows.TimeEndPeriod(1), "failed to lower the system tick"),
		)
	}
	return err
}
