#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy
from tests.task_targets import TaskTargets
from x import payload
from x.strings import random_name


@pytest.mark.tcp
class TestTCPReadTask:
    """Tests for TCP read task configuration."""

    def test_payload_type(self) -> None:
        """Should create a task of the TCP read type."""
        task = sy.tcp.ReadTask(device="server-1", name="Pump read")
        pld = task.to_payload()
        assert pld.type == "tcp_read"
        assert pld.name == "Pump read"

    def test_configure_and_retrieve_read_task(
        self, client: sy.Synnax, task_targets: TaskTargets
    ) -> None:
        """Should store a read config that the task class parses again."""
        level = sy.icd.UnsignedBinaryField(name="level", start_bit=0, bit_length=8)
        message = sy.icd.Message(
            name="Reading", payload=sy.icd.BinaryPayload(fields=[level])
        )
        ic = client.icds.create(
            sy.ICD(name=random_name(), messages=[message], framing=sy.icd.CobsFraming())
        )
        task = sy.tcp.ReadTask(
            name="Read",
            device="some-server",
            data_saving_disabled=True,
            entries=[
                sy.icd.ReadEntry(
                    icd=ic.key,
                    messages=[
                        sy.icd.ReadMessage(
                            message=message.key,
                            bindings=[
                                sy.icd.ReadBinding(field=level.key, channel=1234)
                            ],
                        )
                    ],
                    poll=sy.icd.Poll(query=b"\x01", rate=sy.Rate(5)),
                )
            ],
        )
        task_targets.attach(task.config)
        client.tasks.configure(task)
        parsed = sy.tcp.ReadTask(client.tasks.retrieve(key=task.key))
        assert parsed.config.data_saving_disabled is True
        assert parsed.config.entries[0].poll == sy.icd.Poll(
            query=b"\x01", rate=sy.Rate(5)
        )
        assert parsed.config.entries == task.config.entries


@pytest.mark.tcp
class TestTCPWriteTask:
    """Tests for TCP write task configuration."""

    def test_payload_type(self) -> None:
        """Should create a task of the TCP write type."""
        task = sy.tcp.WriteTask(name="Write", device="server-1")
        assert task.to_payload().type == "tcp_write"

    def test_configure_and_retrieve_write_task(
        self, client: sy.Synnax, task_targets: TaskTargets
    ) -> None:
        """Should store a write config that the task class parses again."""
        valve = sy.icd.UnsignedBinaryField(name="valve", start_bit=0, bit_length=8)
        unit = sy.icd.UnsignedBinaryField(name="unit", start_bit=8, bit_length=8)
        message = sy.icd.Message(
            name="Command", payload=sy.icd.BinaryPayload(fields=[valve, unit])
        )
        ic = client.icds.create(
            sy.ICD(name=random_name(), messages=[message], framing=sy.icd.CobsFraming())
        )
        task = sy.tcp.WriteTask(
            name="Write",
            device="some-server",
            targets=[
                sy.icd.WriteTarget(
                    icd=ic.key,
                    message=sy.icd.WriteMessage(
                        message=message.key,
                        bindings=[
                            sy.icd.ChannelWriteBinding(field=valve.key, channel=4321),
                            sy.icd.ValueWriteBinding(
                                field=unit.key,
                                value=sy.icd.UnsignedWriteValue(
                                    value=payload.Unsigned(1)
                                ),
                            ),
                        ],
                    ),
                    period=100 * sy.TimeSpan.MILLISECOND,
                )
            ],
        )
        task_targets.attach(task.config)
        client.tasks.configure(task)
        parsed = sy.tcp.WriteTask(client.tasks.retrieve(key=task.key))
        assert parsed.config.targets[0].period == 100 * sy.TimeSpan.MILLISECOND
        assert parsed.config.targets == task.config.targets
