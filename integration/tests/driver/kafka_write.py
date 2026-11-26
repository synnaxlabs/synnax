#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from collections.abc import Callable

import synnax as sy
from synnax import kafka
from tests.driver.kafka_task import (
    KafkaWriteTaskCase,
    create_target,
    open_consumer,
    wait_for_record,
)
from tests.driver.task import (
    StatusWatch,
    create_channel,
    create_index,
    is_running,
    status_text,
)

# Each command that the bounds case sends out of range, with the reason it is refused.
REFUSALS = [(150.0, "150 is above the max 100"), (-5.0, "-5 is below the min 0")]


def write_commands(client: sy.Synnax, commands: list[tuple[sy.Channel, float]]) -> None:
    """Write each command in one frame, with the time now on its index."""
    now = sy.TimeStamp.now()
    frame: dict[int, float | sy.TimeStamp] = {}
    for ch, value in commands:
        frame[ch.index] = now
        frame[ch.key] = value
    with client.open_writer(start=now, channels=list(frame)) as writer:
        writer.write(frame)


def warning_of(description: str) -> Callable[[sy.task.Status], bool]:
    """Return whether a status is a warning with description."""
    return lambda s: s.variant == "warning" and s.description == description


class KafkaWriteJSON(KafkaWriteTaskCase):
    """Sends each command as a JSON record with no key, next to a field that sends its
    default."""

    task_name = "Kafka Write JSON"

    def assert_written(self) -> None:
        with open_consumer() as consumer:
            wait_for_record(
                consumer,
                topic="plant.valve.set",
                expected={"position": 100, "source": "synnax"},
            )

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[kafka.WriteTarget]:
        idx = create_index(client, "kafka_valve_cmd_time")
        cmd = create_channel(
            client,
            name="kafka_valve_cmd",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        return [
            create_target(
                client,
                topic="plant.valve.set",
                bindings=[
                    (sy.icd.FloatJSONField(name="position", pointer="/position"), cmd)
                ],
                fields=[
                    sy.icd.StringJSONField(
                        name="source", pointer="/source", default="synnax"
                    )
                ],
            )
        ]


class KafkaWriteTargets(KafkaWriteTaskCase):
    """Sends commands to three targets: two on one topic with different record keys,
    and one on a topic of its own with no key."""

    task_name = "Kafka Write Targets"

    def assert_written(self) -> None:
        with open_consumer() as consumer:
            wait_for_record(
                consumer,
                topic="plant.commands",
                key="pump",
                expected={"speed": 100},
            )
            wait_for_record(
                consumer,
                topic="plant.commands",
                key="heater",
                expected={"power": 101},
            )
            wait_for_record(consumer, topic="plant.alarm.set", expected={"level": 102})

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[kafka.WriteTarget]:
        idx = create_index(client, "kafka_targets_cmd_time")
        pump, heater, alarm = (
            create_channel(
                client,
                name=f"kafka_{name}_cmd",
                data_type=sy.DataType.FLOAT64,
                index=idx.key,
            )
            for name in ("pump", "heater", "alarm")
        )
        return [
            create_target(
                client,
                topic="plant.commands",
                record_key="pump",
                bindings=[
                    (sy.icd.FloatJSONField(name="speed", pointer="/speed"), pump)
                ],
            ),
            create_target(
                client,
                topic="plant.commands",
                record_key="heater",
                bindings=[
                    (sy.icd.FloatJSONField(name="power", pointer="/power"), heater)
                ],
            ),
            create_target(
                client,
                topic="plant.alarm.set",
                bindings=[
                    (sy.icd.FloatJSONField(name="level", pointer="/level"), alarm)
                ],
            ),
        ]


class KafkaWriteEachChannel(KafkaWriteTaskCase):
    """Fills one message from two command channels. A command on either channel sends
    a record, and the other field keeps its last command."""

    task_name = "Kafka Write Each Channel"

    def assert_written(self) -> None:
        with open_consumer() as consumer:
            wait_for_record(
                consumer,
                topic="plant.point.set",
                expected={"x": 100, "y": 101},
            )

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[kafka.WriteTarget]:
        x, y = (
            create_channel(
                client,
                name=f"kafka_point_{axis}_cmd",
                data_type=sy.DataType.FLOAT64,
                index=create_index(client, f"kafka_point_{axis}_cmd_time").key,
            )
            for axis in ("x", "y")
        )
        return [
            create_target(
                client,
                topic="plant.point.set",
                bindings=[
                    (sy.icd.FloatJSONField(name="x", pointer="/x"), x),
                    (sy.icd.FloatJSONField(name="y", pointer="/y"), y),
                ],
            )
        ]

    def run(self) -> None:
        super().run()
        self.test_command_on_each_channel()

    def test_command_on_each_channel(self) -> None:
        assert self.tsk is not None
        self.log("Testing: A command on each bound channel")
        x, y = (self.client.channels.retrieve(k) for k in self._channel_keys(self.tsk))
        with self.tsk.run(timeout=self.COMMAND_TIMEOUT), open_consumer() as consumer:
            steps: list[tuple[list[tuple[sy.Channel, float]], dict[str, float]]] = [
                ([(x, 1), (y, 2)], {"x": 1, "y": 2}),
                ([(x, 7)], {"x": 7, "y": 2}),
                ([(y, 8)], {"x": 7, "y": 8}),
            ]
            for commands, expected in steps:
                write_commands(self.client, commands)
                wait_for_record(
                    consumer,
                    topic="plant.point.set",
                    expected=expected,
                )


class KafkaWriteBounds(KafkaWriteTaskCase):
    """Refuses a command above the max or below the min of its field with a warning,
    and sends the next command in range."""

    task_name = "Kafka Write Bounds"
    command_values: list[list[float]] | None = [[25.0], [75.0]]

    def assert_written(self) -> None:
        with open_consumer() as consumer:
            wait_for_record(
                consumer,
                topic="plant.limit.set",
                expected={"setpoint": 75},
            )

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[kafka.WriteTarget]:
        idx = create_index(client, "kafka_limit_cmd_time")
        cmd = create_channel(
            client,
            name="kafka_limit_cmd",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        return [
            create_target(
                client,
                topic="plant.limit.set",
                bindings=[
                    (
                        sy.icd.FloatJSONField(
                            name="setpoint", pointer="/setpoint", min=0, max=100
                        ),
                        cmd,
                    )
                ],
            )
        ]

    def run(self) -> None:
        super().run()
        self.test_refuses_out_of_range()

    def test_refuses_out_of_range(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Refuse commands outside the min and max")
        cmd = self.client.channels.retrieve(self._channel_keys(self.tsk)[0])
        with (
            StatusWatch(self.client, [self.tsk.key]) as watch,
            self.tsk.run(timeout=self.COMMAND_TIMEOUT),
            open_consumer() as consumer,
        ):
            for value, reason in REFUSALS:
                write_commands(self.client, [(cmd, value)])
                description = (
                    "Command for target plant.limit.set: message plant.limit.set: "
                    f"field setpoint: {reason}: validation error"
                )
                warning = watch.wait(
                    warning_of(description),
                    timeout=10 * sy.TimeSpan.SECOND,
                    expected=f"the warning {description!r}",
                )
                message = re.fullmatch(r"Skipped \d+ of \d+ commands?", warning.message)
                if message is None or not is_running(warning):
                    raise AssertionError(f"Unexpected warning: {status_text(warning)}")
            write_commands(self.client, [(cmd, 50)])
            records = wait_for_record(
                consumer,
                topic="plant.limit.set",
                expected={"setpoint": 50},
            )
        sent = [v for _, v in records]
        refused = [v for v in sent if v in ({"setpoint": 150}, {"setpoint": -5})]
        if refused:
            raise AssertionError(f"The task sent refused commands: {refused}")
