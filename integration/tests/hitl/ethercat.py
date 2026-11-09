#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""EtherCAT task tests on the Dewesoft IOLITE bus of the HITL station, on its
`Ethernet 2` adapter.

Position | Slave
---------|-----------------------------------------------------------------
1        | IOLITE GATE LATCH v2.00 BUS1, the reference clock
2 and 3  | IOLiteSTG_KK (6xSTG), reaches SAFE-OP only in DC mode
4 to 7   | IOLITEr 32xDO BUS1
"""

from pydantic import JsonValue

import synnax as sy
from tests.driver.task import ReadTaskCase, WriteTaskCase, create_channel, create_index

SLAVES = [
    "IOLITE GATE LATCH v2.00 BUS1",
    "IOLiteSTG_KK",
    "IOLiteSTG_KK",
    "IOLITEr 32xDO BUS1",
    "IOLITEr 32xDO BUS1",
    "IOLITEr 32xDO BUS1",
    "IOLITEr 32xDO BUS1",
]
"""The name of the slave at each position, from position 1."""

STGS = (2, 3)
"""The positions of the 6xSTGs."""

DO = 4
"""The position of the 32xDO that the write test drives."""

STG_DC: dict[str, JsonValue] = {
    "index": 1,
    "assign_activate": 0x0300,
    "sync0_cycle": 1_000,
    "sync1_cycle": 1_000_000_000,
}
"""The DC timing that Dewesoft gives a 6xSTG: SYNC0 every 1 µs and SYNC1 every 1 s."""

_SCAN_TIMEOUT = 30 * sy.TimeSpan.SECOND
_START_TIMEOUT = sy.TimeSpan.MINUTE
"""The Driver retries a bus start for up to a minute."""


def prepare_bus(client: sy.Synnax) -> dict[int, sy.Device]:
    """Returns the slaves of the bus by position, once the Driver reports each slave in
    SLAVES present at its position. Sets each 6xSTG to DC, as every task brings each
    slave of the bus to OP.

    :raises TimeoutError: If the Driver does not report the slaves within 30 seconds.
    """
    found: dict[int, sy.Device] = {}

    def complete() -> dict[int, sy.Device] | None:
        nonlocal found
        devices = client.devices.retrieve(
            makes=[sy.ethercat.MAKE], models=[sy.ethercat.MODEL], include_status=True
        )
        found = {}
        for d in devices:
            position = d.properties["position"]
            assert isinstance(position, int)
            if d.status is not None and d.status.variant == "success":
                found[position] = d
        names = [
            found[p].name if p in found else None for p in range(1, len(SLAVES) + 1)
        ]
        return found if names == SLAVES else None

    slaves = sy.poll(complete, timeout=_SCAN_TIMEOUT, interval=0.5)
    if slaves is None:
        names = {p: d.name for p, d in sorted(found.items())}
        raise TimeoutError(f"Expected the slaves {SLAVES} present, but found {names}")
    for pos in STGS:
        slaves[pos].properties["op_mode"] = STG_DC
        slaves[pos] = client.devices.create(slaves[pos])
    return slaves


class EtherCATRead(ReadTaskCase[sy.ethercat.ReadTask]):
    """Reads four voltages and the cycle counter of a 6xSTG in DC mode."""

    task_name = "HITL EtherCAT Read"
    SAMPLE_RATE = 100 * sy.Rate.HZ
    STREAM_RATE = 25 * sy.Rate.HZ
    COMMAND_TIMEOUT = _START_TIMEOUT

    def retrieve_device(self) -> sy.Device:
        return prepare_bus(self.client)[STGS[0]]

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.ethercat.ReadTask:
        idx = create_index(self.client, "hitl_ethercat_time")
        entries = [f"u{i}" for i in range(1, 5)] + ["BufCounter"]
        channels: list[sy.ethercat.ReadChannel] = []
        for name in entries:
            entry = sy.ethercat.find_entry(device, name)
            data_type = sy.DataType(entry.data_type)
            channels.append(
                sy.ethercat.EntryReadChannel(
                    device=device.key,
                    index=entry.index,
                    sub_index=entry.sub_index,
                    data_type=data_type,
                    channel=create_channel(
                        self.client,
                        name=f"hitl_ethercat_{name.lower()}",
                        data_type=data_type,
                        index=idx.key,
                    ),
                )
            )
        return sy.ethercat.ReadTask(
            name=task_name,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            channels=channels,
        )

    def _channel_keys(self, task: sy.ethercat.ReadTask) -> list[int]:
        return [ch.channel for ch in task.config.channels]

    def run(self) -> None:
        super().run()
        self.test_counter_advances()

    def test_counter_advances(self) -> None:
        """Checks that the slave sends new data in each cycle. A 6xSTG that the master
        reads during its move to SAFE-OP never starts its data loop, and sends zeros."""
        assert self.tsk is not None
        self.log("Testing: Cycle counter advances")
        counter = self.client.channels.retrieve("hitl_ethercat_bufcounter")
        start = sy.TimeStamp.now()
        with self.tsk.run(timeout=self.COMMAND_TIMEOUT):
            sy.sleep(self.TASK_DURATION)  # noqa: TID251 acquire for a fixed time
        counts = counter.read(sy.TimeRange(start, sy.TimeStamp.MAX)).to_numpy()
        if len(counts) < 2 or len(set(counts)) < len(counts) // 2:
            raise AssertionError(
                f"Expected the cycle counter to advance, but read {counts}"
            )


class EtherCATWrite(WriteTaskCase[sy.ethercat.WriteTask]):
    """Writes two outputs of a 32xDO and checks the state that the Driver reports."""

    task_name = "HITL EtherCAT Write"
    COMMAND_TIMEOUT = _START_TIMEOUT
    command_values: list[list[float]] | None = [[1, 0], [0, 1]]

    def retrieve_device(self) -> sy.Device:
        return prepare_bus(self.client)[DO]

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.ethercat.WriteTask:
        cmd_idx = create_index(self.client, "hitl_ethercat_do_cmd_time")
        state_idx = create_index(self.client, "hitl_ethercat_do_state_time")
        channels: list[sy.ethercat.WriteChannel] = []
        for i in (1, 2):
            entry = sy.ethercat.find_entry(device, f"LED {i}", direction="output")
            data_type = sy.DataType(entry.data_type)
            channels.append(
                sy.ethercat.EntryWriteChannel(
                    device=device.key,
                    index=entry.index,
                    sub_index=entry.sub_index,
                    data_type=data_type,
                    cmd_channel=create_channel(
                        self.client,
                        name=f"hitl_ethercat_do_cmd_{i}",
                        data_type=data_type,
                        index=cmd_idx.key,
                    ),
                    state_channel=create_channel(
                        self.client,
                        name=f"hitl_ethercat_do_state_{i}",
                        data_type=data_type,
                        index=state_idx.key,
                    ),
                )
            )
        return sy.ethercat.WriteTask(
            name=task_name, state_rate=sample_rate, channels=channels
        )

    def _channel_keys(self, task: sy.ethercat.WriteTask) -> list[int]:
        return [ch.cmd_channel for ch in task.config.channels]

    def assert_written(self) -> None:
        assert self.tsk is not None and self.command_values is not None
        expected = self.command_values[-1]
        states = [ch.state_channel for ch in self.tsk.config.channels]

        def written() -> bool | None:
            frame = self.client.read_latest(states)
            done = all(
                len(frame[k]) > 0 and frame[k][-1] == v
                for k, v in zip(states, expected, strict=True)
            )
            return True if done else None

        if sy.poll(written, timeout=5 * sy.TimeSpan.SECOND, interval=0.1) is None:
            raise AssertionError(f"Expected the 32xDO states {expected}")
