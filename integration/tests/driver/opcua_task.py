#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""OPC UA-specific task test cases."""

import ssl
from abc import abstractmethod

from examples.opcua import OPCUASim

import synnax as sy
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import ReadTaskCase, WriteTaskCase


class OPCUAReadTaskCase(SimulatorCase, ReadTaskCase[sy.opcua.ReadTask]):
    """Base class for OPC UA read task tests."""

    sim_classes = [OPCUASim]
    SAMPLE_RATE = 50 * sy.Rate.HZ
    array_mode: bool = False
    array_size: int = 5

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]: ...

    def _channel_keys(self, task: sy.opcua.ReadTask) -> list[int]:
        return [ch.channel for ch in task.config.channels]

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.opcua.ReadTask:
        """Create an OPC UA read task."""
        channels = self.create_channels(self.client)

        return sy.opcua.ReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            array_mode=self.array_mode,
            array_size=self.array_size,
            channels=channels,
        )


class OPCUASecuredReadTaskCase(OPCUAReadTaskCase):
    """Checks only that a secured read task connects and reads. The unsecured cases
    cover the rest of the task lifecycle."""

    def run(self) -> None:
        self.test_start_and_stop()


class OPCUAWriteTaskCase(SimulatorCase, WriteTaskCase[sy.opcua.WriteTask]):
    """Base class for OPC UA write task tests."""

    sim_classes = [OPCUASim]
    SAMPLE_RATE = 50 * sy.Rate.HZ

    def _channel_keys(self, task: sy.opcua.WriteTask) -> list[int]:
        return [ch.cmd_channel for ch in task.config.channels]

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.WriteChannel]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.opcua.WriteTask:
        """Create an OPC UA write task."""
        channels = self.create_channels(self.client)
        return sy.opcua.WriteTask(
            name=task_name,
            device=device.key,
            channels=channels,
        )


class OPCUASecuredWriteTaskCase(OPCUAWriteTaskCase):
    """Checks only that a secured write task connects and writes. The unsecured case
    covers the rest of the task lifecycle."""

    def run(self) -> None:
        self.test_send_commands()


def pinned_certificate(dev: sy.Device) -> bytes | None:
    """Return the DER bytes of the server certificate that the device trusts, if any."""
    connection = dev.properties.get("connection")
    if not isinstance(connection, dict):
        return None
    pem = connection.get("server_certificate")
    if not isinstance(pem, str) or pem == "":
        return None
    return ssl.PEM_cert_to_DER_cert(pem)
