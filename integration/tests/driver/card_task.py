#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Base class for the ARINC 429 and MIL-STD-1553 task cases, which run on the cards
the Driver simulates in process."""

from collections.abc import Iterator
from contextlib import ExitStack, contextmanager
from typing import Any

import numpy as np

import synnax as sy
from synnax import arinc429, mil1553
from tests.driver.bus_task import BusCase, Collector

# The terminal address the simulated remote terminal owns.
REMOTE_TERMINAL = 5

type PeerTask = (
    arinc429.ReadTask | arinc429.WriteTask | mil1553.ReadTask | mil1553.WriteTask
)


class Card:
    """A channel of a card the Driver simulates in process. A case lists it in
    device_classes, since nothing runs outside the Driver."""

    device_name: str
    channel_names: tuple[str, ...] = ()


class ARINC429Card(Card):
    device_name = "ARINC 429 Simulated Channel"

    @staticmethod
    def create_device(rack_key: int) -> sy.Device:
        return arinc429.Device(
            backend=arinc429.BACKEND_SIMULATED,
            name=ARINC429Card.device_name,
            location="simulated",
            rack=rack_key,
        )


class BusController(Card):
    device_name = "MIL-STD-1553 Simulated Bus Controller"

    @staticmethod
    def create_device(rack_key: int) -> sy.Device:
        return mil1553.Device(
            backend=mil1553.BACKEND_SIMULATED,
            name=BusController.device_name,
            location="simulated",
            rack=rack_key,
            role="bus_controller",
        )


class RemoteTerminal(Card):
    device_name = "MIL-STD-1553 Simulated Remote Terminal"

    @staticmethod
    def create_device(rack_key: int) -> sy.Device:
        return mil1553.Device(
            backend=mil1553.BACKEND_SIMULATED,
            name=RemoteTerminal.device_name,
            location="simulated",
            rack=rack_key,
            role="remote_terminal",
            terminals=[REMOTE_TERMINAL],
        )


class CardCase(BusCase):
    """Runs a task on a simulated card. Peer tasks on the same bus stand in for the
    other end of it."""

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self._peers: list[PeerTask] = []

    def device(self, card: type[Card]) -> sy.Device:
        """Returns the device of a card in device_classes."""
        return self.client.devices.retrieve(name=card.device_name)

    def configure_peer[T: PeerTask](self, task: T) -> T:
        """Configures a peer task that teardown deletes."""
        self.client.tasks.configure(task)
        self._peers.append(task)
        return task

    @contextmanager
    def run_all(self, *tasks: PeerTask) -> Iterator[None]:
        """Runs the case's task and tasks for the duration of the block."""
        with ExitStack() as stack:
            for t in tasks:
                stack.enter_context(t.run())
            stack.enter_context(self.task.run())
            yield

    def teardown(self) -> None:
        try:
            for peer in self._peers:
                with self._try_to("delete peer task"):
                    self.client.tasks.delete(peer.key)
        finally:
            super().teardown()


def wait_for_value(
    samples: Collector,
    key: int,
    value: float,
    timeout: sy.TimeSpan = 10 * sy.TimeSpan.SECOND,
) -> None:
    """Reads until the channel key has a sample equal to value.

    :raises AssertionError: If no such sample arrives before the timeout.
    """
    timer = sy.Timer()
    while timer.elapsed() < timeout:
        samples.drain()
        if np.any(samples[key].astype(np.float64) == value):
            return
        sy.sleep(0.05)
    raise AssertionError(f"Channel {key} received no {value} within {timeout}")
