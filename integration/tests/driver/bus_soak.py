#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Fault-cycling soak of the bus read tasks.

Each Soak case runs one read task until the soak ends while it cycles faults: the
device goes away and comes back, then a burst of bad input arrives. Samples must
resume after each fault. DriverResources samples the Driver process meanwhile and
fails when its memory, open descriptors, or threads keep rising. Run every case in
one asynchronous sequence. The soak runs for three hours, and SYNNAX_SOAK_MINUTES
changes its length, such as for a short local run.
"""

import os
import random
import sys
from abc import ABC, abstractmethod
from dataclasses import dataclass

import can
import psutil
from examples.simulators import can_frames
from examples.simulators.scpi import SCPISim
from examples.simulators.serial_frames import Fault, SerialFrameSim

import synnax as sy
from framework.test_case import TestCase
from framework.trend import rise
from tests.driver.bus_task import BusCase
from tests.driver.can_read import CANRead
from tests.driver.serial_read import SerialRead
from tests.driver.tcp_read import TCPRead
from tests.driver.udp_read import UDPRead

DEFAULT_MINUTES = 180
UNPLUGGED = 5 * sy.TimeSpan.SECOND
BURST = 10 * sy.TimeSpan.SECOND
# Longer than the TCP read's poll timeout, so every reply in the burst arrives late.
LATE_REPLY = 0.75
GARBAGE_RATE = 200 * sy.Rate.HZ
SAMPLE_INTERVAL = 30 * sy.TimeSpan.SECOND
# Caches and pools fill early in a run, so the trend leaves out its first part.
WARMUP = 0.25
MIB = 1024**2
DRIVER_NAMES = ("driver", "driver.exe")


def soak_duration() -> sy.TimeSpan:
    """Returns the length of the soak, from SYNNAX_SOAK_MINUTES."""
    minutes = float(os.environ.get("SYNNAX_SOAK_MINUTES", DEFAULT_MINUTES))
    return sy.TimeSpan.from_seconds(minutes * 60)


class Soak(BusCase, ABC):
    """Runs a read task until the soak ends, cycling faults on its device."""

    def run(self) -> None:
        self.set_manual_timeout(soak_duration())
        cycles = 0
        with self.task.run():
            self.assert_resumes()
            while self.should_continue:
                self.cleanup_simulator()
                sy.sleep(UNPLUGGED)
                self.start_simulator()
                self.assert_resumes()
                self.burst()
                self.assert_resumes()
                cycles += 1
        self.log(f"Ran {cycles} fault cycles")

    @abstractmethod
    def burst(self) -> None:
        """Sends bad input to the task for BURST, then stops."""


class TCPSoak(Soak, TCPRead):
    """Hangs the instrument, then delays its replies past the poll timeout."""

    def burst(self) -> None:
        sim = self.simulator(SCPISim)
        sim.hang()
        sy.sleep(BURST / 2)
        sim.unhang()
        sim.delay(LATE_REPLY)
        sy.sleep(BURST / 2)
        sim.delay(0)


class SerialSoak(Soak, SerialRead):
    """Mixes every fault the serial simulator has into its stream."""

    def burst(self) -> None:
        sim = self.simulator(SerialFrameSim)
        sim.inject(Fault.GARBAGE | Fault.TRUNCATED | Fault.BAD_CHECKSUM | Fault.UNKNOWN)
        sy.sleep(BURST)
        sim.inject(Fault.NONE)


class UDPSoak(Soak, UDPRead):
    """Cycles only the device. The task takes datagrams from the simulator's address
    alone, so bad input from the test would never reach the decoder."""

    def burst(self) -> None:
        pass


class CANSoak(Soak, CANRead):
    """Sends random frames on the bus: stray identifiers, and the fast message's
    identifier with too few bytes."""

    def burst(self) -> None:
        rng = random.Random(0)
        timer = sy.Timer()
        with can.Bus(interface="socketcan", channel=can_frames.CHANNEL) as bus:
            while timer.elapsed() < BURST:
                bus.send(_stray_frame(rng))
                sy.sleep(GARBAGE_RATE.period)


def _stray_frame(rng: random.Random) -> can.Message:
    if rng.random() < 0.25:
        return can.Message(
            arbitration_id=can_frames.FAST_ID, data=rng.randbytes(rng.randint(0, 7))
        )
    extended = rng.random() < 0.5
    return can.Message(
        arbitration_id=rng.getrandbits(29 if extended else 11),
        is_extended_id=extended,
        data=rng.randbytes(rng.randint(0, 8)),
    )


@dataclass
class Usage:
    """What the Driver process held at one time."""

    seconds: float
    pid: int
    memory: float
    descriptors: float
    threads: float


# How far each resource may rise after the warm-up before the soak fails.
LIMITS = {"memory": 32 * MIB, "descriptors": 16, "threads": 8}


def find_driver() -> psutil.Process:
    """Returns the Driver process that a Core started.

    :raises RuntimeError: If no such process runs.
    """
    for proc in psutil.process_iter(["name"]):
        if proc.info["name"] not in DRIVER_NAMES:
            continue
        try:
            parent = proc.parent()
        except psutil.Error:
            continue
        if parent is not None and parent.name().startswith("synnax"):
            return proc
    raise RuntimeError("No Driver process started by a Core is running")


def measure(proc: psutil.Process, seconds: float) -> Usage:
    """Returns what proc holds now, at the given seconds into the soak. Windows counts
    handles as descriptors."""
    with proc.oneshot():
        if sys.platform == "win32":
            descriptors = proc.num_handles()
        else:
            descriptors = proc.num_fds()
        return Usage(
            seconds=seconds,
            pid=proc.pid,
            memory=proc.memory_info().rss,
            descriptors=descriptors,
            threads=proc.num_threads(),
        )


class DriverResources(TestCase):
    """Samples the Driver every SAMPLE_INTERVAL until the soak ends. Fails when the
    Driver restarts, or when a resource rises by more than its limit after the
    warm-up."""

    def run(self) -> None:
        self.set_manual_timeout(soak_duration())
        soak = sy.Timer()
        usages = [measure(find_driver(), 0)]
        interval = sy.Timer()
        while self.should_continue:
            if interval.elapsed() < SAMPLE_INTERVAL:
                continue
            interval = sy.Timer()
            usage = measure(find_driver(), soak.elapsed().seconds)
            self.log(f"{usage}")
            if usage.pid != usages[0].pid:
                self.fail(f"The Driver restarted: PID {usages[0].pid} -> {usage.pid}")
            usages.append(usage)
        kept = usages[int(len(usages) * WARMUP) :]
        if len(kept) < 2:
            self.fail(f"Took {len(usages)} samples, too few to find a trend")
        seconds = [u.seconds for u in kept]
        rises = {
            name: rise(seconds, [getattr(u, name) for u in kept]) for name in LIMITS
        }
        self.log(f"Rises after the warm-up: {rises}")
        over = [
            f"{name} rose by {rises[name]:.0f}, over its limit of {limit}"
            for name, limit in LIMITS.items()
            if rises[name] > limit
        ]
        if over:
            self.fail("; ".join(over))
