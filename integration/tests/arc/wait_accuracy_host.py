#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import platform
import statistics
import sys
import time
import urllib.request
from collections.abc import Callable

import psutil

from framework.test_case import TestCase

WAIT_MS = 10
COUNT = 40
SPIN_MS = 200
CPU_SAMPLE_S = 1.0
IMDS = "http://169.254.169.254/latest"
IMDS_TIMEOUT_S = 1.0


def measure(wait: Callable[[], None]) -> str:
    """Returns the spread of how late COUNT calls of a WAIT_MS wait return."""
    late_ms: list[float] = []
    for _ in range(COUNT):
        start = time.perf_counter_ns()
        wait()
        late_ms.append((time.perf_counter_ns() - start) / 1e6 - WAIT_MS)
    late_ms.sort()
    p90 = late_ms[int(0.9 * (COUNT - 1))]
    return (
        f"min {late_ms[0]:.3f}, median {statistics.median(late_ms):.3f}, "
        f"p90 {p90:.3f}, max {late_ms[-1]:.3f} ms late"
    )


def sleep() -> None:
    time.sleep(WAIT_MS / 1e3)


def longest_stall() -> str:
    """Spins for SPIN_MS and returns the longest gap between two clock reads. The gap
    is the longest time the host held this thread off its core.
    """
    last = time.perf_counter_ns()
    end = last + SPIN_MS * 1_000_000
    longest = 0
    over_1ms = 0
    while last < end:
        now = time.perf_counter_ns()
        longest = max(longest, now - last)
        if now - last > 1_000_000:
            over_1ms += 1
        last = now
    return f"{longest / 1e6:.3f} ms longest, {over_1ms} over 1 ms"


def instance_type() -> str:
    """Returns the EC2 instance type of the host, or a note if there is none."""
    try:
        token_request = urllib.request.Request(
            f"{IMDS}/api/token",
            method="PUT",
            headers={"X-aws-ec2-metadata-token-ttl-seconds": "60"},
        )
        with urllib.request.urlopen(token_request, timeout=IMDS_TIMEOUT_S) as res:
            token = res.read().decode()
        type_request = urllib.request.Request(
            f"{IMDS}/meta-data/instance-type",
            headers={"X-aws-ec2-metadata-token": token},
        )
        with urllib.request.urlopen(type_request, timeout=IMDS_TIMEOUT_S) as res:
            return str(res.read().decode())
    except OSError:
        return "not an EC2 instance"


if sys.platform == "win32":
    import ctypes
    from ctypes import wintypes

    HIGH_RESOLUTION = 0x00000002
    TIMER_ALL_ACCESS = 0x1F0003
    INFINITE = 0xFFFFFFFF
    THREAD_PRIORITY_NORMAL = 0
    THREAD_PRIORITY_TIME_CRITICAL = 15

    kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel32.CreateWaitableTimerExW.restype = wintypes.HANDLE
    kernel32.CreateWaitableTimerExW.argtypes = [
        ctypes.c_void_p,
        wintypes.LPCWSTR,
        wintypes.DWORD,
        wintypes.DWORD,
    ]
    kernel32.SetWaitableTimer.argtypes = [
        wintypes.HANDLE,
        ctypes.POINTER(wintypes.LARGE_INTEGER),
        wintypes.LONG,
        ctypes.c_void_p,
        ctypes.c_void_p,
        wintypes.BOOL,
    ]
    kernel32.WaitForSingleObject.argtypes = [wintypes.HANDLE, wintypes.DWORD]
    kernel32.CloseHandle.argtypes = [wintypes.HANDLE]
    kernel32.GetCurrentThread.restype = wintypes.HANDLE
    kernel32.SetThreadPriority.argtypes = [wintypes.HANDLE, ctypes.c_int]
    kernel32.GetCurrentProcess.restype = wintypes.HANDLE
    kernel32.SetProcessInformation.argtypes = [
        wintypes.HANDLE,
        ctypes.c_int,
        ctypes.c_void_p,
        wintypes.DWORD,
    ]

    PROCESS_POWER_THROTTLING = 4
    EXECUTION_SPEED = 0x1
    IGNORE_TIMER_RESOLUTION = 0x4

    class PowerThrottlingState(ctypes.Structure):
        _fields_ = [
            ("version", wintypes.ULONG),
            ("control_mask", wintypes.ULONG),
            ("state_mask", wintypes.ULONG),
        ]

    def stop_power_throttling() -> str:
        """Tells Windows to honor the timer requests of this process at all times.
        Windows can ignore them for a process with no visible window.
        """
        state = PowerThrottlingState(1, EXECUTION_SPEED | IGNORE_TIMER_RESOLUTION, 0)
        if kernel32.SetProcessInformation(
            kernel32.GetCurrentProcess(),
            PROCESS_POWER_THROTTLING,
            ctypes.byref(state),
            ctypes.sizeof(state),
        ):
            return "with power throttling off"
        return f"power throttling not set (error {ctypes.get_last_error()})"

    def timer_resolution() -> str:
        """Returns the current, best, and default timer resolution of Windows."""
        default, best, current = wintypes.ULONG(), wintypes.ULONG(), wintypes.ULONG()
        ctypes.WinDLL("ntdll").NtQueryTimerResolution(
            ctypes.byref(default), ctypes.byref(best), ctypes.byref(current)
        )
        return (
            f"{current.value / 1e4:.4f} ms current, {best.value / 1e4:.4f} ms best, "
            f"{default.value / 1e4:.4f} ms default"
        )

    def measure_timer(flags: int) -> str:
        """Returns the spread of a waitable timer that Windows makes with flags."""
        handle = kernel32.CreateWaitableTimerExW(None, None, flags, TIMER_ALL_ACCESS)
        if not handle:
            return f"unavailable, error {ctypes.get_last_error()}"
        due = wintypes.LARGE_INTEGER(-WAIT_MS * 10_000)

        def wait() -> None:
            kernel32.SetWaitableTimer(handle, ctypes.byref(due), 0, None, None, False)
            kernel32.WaitForSingleObject(handle, INFINITE)

        result = measure(wait)
        kernel32.CloseHandle(handle)
        return result

    def measure_timers(label: str) -> list[str]:
        return [
            f"{label}, timer resolution: {timer_resolution()}",
            f"{label}, sleep {WAIT_MS} ms: {measure(sleep)}",
            f"{label}, high-resolution timer: {measure_timer(HIGH_RESOLUTION)}",
            f"{label}, standard timer: {measure_timer(0)}",
        ]

    def measure_windows() -> list[str]:
        """Returns the timer spreads as found, then with each of these added in turn:
        power throttling off, a 1 ms timer resolution, and a time-critical thread.
        """
        winmm = ctypes.WinDLL("winmm")
        thread = kernel32.GetCurrentThread()
        lines = measure_timers("as found")
        label = stop_power_throttling()
        lines += measure_timers(label)
        winmm.timeBeginPeriod(1)
        label += ", timeBeginPeriod(1)"
        lines += measure_timers(label)
        kernel32.SetThreadPriority(thread, THREAD_PRIORITY_TIME_CRITICAL)
        lines += measure_timers(f"{label}, time-critical thread")
        kernel32.SetThreadPriority(thread, THREAD_PRIORITY_NORMAL)
        winmm.timeEndPeriod(1)
        return lines


class WaitAccuracyHost(TestCase):
    """Logs the CPU load of the host and how it holds short waits with no Arc
    runtime. The numbers are the reference for the wait accuracy cases.
    """

    def run(self) -> None:
        self.log(f"host: {platform.platform()}, {os.cpu_count()} cores")
        self.log(f"instance type: {instance_type()}")
        load = psutil.cpu_percent(interval=CPU_SAMPLE_S, percpu=True)
        self.log(f"cpu load per core over {CPU_SAMPLE_S} s: {load} %")
        self.log(f"stall in a {SPIN_MS} ms spin: {longest_stall()}")
        if sys.platform == "win32":
            for line in measure_windows():
                self.log(line)
        else:
            self.log(f"sleep {WAIT_MS} ms: {measure(sleep)}")
        self.log(f"stall in a {SPIN_MS} ms spin: {longest_stall()}")
