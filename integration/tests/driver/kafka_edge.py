#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax import kafka
from tests.driver.kafka_task import KafkaReadTaskCase, create_topic
from tests.driver.task import (
    StatusWatch,
    collect_samples,
    create_channel,
    create_index,
    is_running,
    status_text,
)

# The broker takes a few seconds to start, and the task a few more to rejoin its group.
RECOVERY_TIMEOUT = 60 * sy.TimeSpan.SECOND


class KafkaEdgeBrokerRestart(KafkaReadTaskCase):
    """Stops and starts the broker while a read task runs. The task warns that it lost
    the cluster, reports that it reconnected, and then reads again."""

    task_name = "Kafka Edge Broker Restart"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[kafka.ReadTopic]:
        idx = create_index(client, "kafka_restart_time")
        count = create_channel(
            client,
            name="kafka_restart_count",
            data_type=sy.DataType.INT64,
            index=idx.key,
        )
        return [
            create_topic(
                client,
                topic="plant.sensors",
                bindings=[
                    (sy.icd.IntegerJSONField(name="count", pointer="/count"), count)
                ],
            )
        ]

    def run(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()
        self.log("Testing: Broker restart while the task runs")
        keys = self._channel_keys(self.tsk)
        with (
            StatusWatch(self.client, [self.tsk.key]) as watch,
            self.tsk.run(timeout=self.COMMAND_TIMEOUT),
        ):
            collect_samples(
                self.client, keys, duration=sy.TimeSpan.SECOND, frame_size=1
            )
            self.cleanup_simulator()
            lost = watch.wait(
                lambda s: s.message == "Lost connection to cluster",
                timeout=RECOVERY_TIMEOUT,
                expected="the warning of a lost cluster",
            )
            # The cause before the fixed tail depends on what the task was doing.
            if (
                lost.variant != "warning"
                or not lost.description.endswith(
                    ". The task retries until the cluster is reachable."
                )
                or not is_running(lost)
            ):
                raise AssertionError(f"Unexpected lost status: {status_text(lost)}")
            self.start_simulator()
            back = watch.wait(
                lambda s: s.message == "Reconnected to cluster",
                timeout=RECOVERY_TIMEOUT,
                expected="the status of a reconnected cluster",
            )
            if back.variant != "success" or back.description or not is_running(back):
                raise AssertionError(
                    f"Unexpected reconnect status: {status_text(back)}"
                )
            collect_samples(
                self.client, keys, duration=sy.TimeSpan.SECOND, frame_size=1
            )
            watch.collect(sy.TimeSpan.SECOND)
        if any(s.variant == "error" for s in watch.seen):
            raise AssertionError(f"The task failed during the restart: {watch.texts()}")
