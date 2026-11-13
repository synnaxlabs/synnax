#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from common import BUSY, IDLE, OFFLINE
from conftest import Client, core
from pools import (
    POOL_INDEX,
    WAIT_INDEX,
    PoolReport,
    count_channels,
    pool_counts,
    pool_of,
    pool_waits,
    wait_channels,
)
from publisher import Publisher

BUILD = {"self-hosted", "ubuntu-build-bot"}
MAC = {"self-hosted", "macos-build-bot"}
REVIEW = {"self-hosted", "review"}


class TestPools:
    """Tests for mapping runner labels to pools."""

    def test_should_map_runner_labels_to_a_pool(self) -> None:
        assert pool_of(BUILD | {"Linux"}) == "ubuntu-build-bot"
        assert pool_of({"self-hosted", "Linux"}) is None

    def test_should_map_an_unscaled_pool(self) -> None:
        assert pool_of(MAC) == "macos-build-bot"


class TestPoolChannels:
    """Tests for the names of the pool channels."""

    def test_should_name_the_counts_of_every_pool_and_the_total(self) -> None:
        names = count_channels()
        assert len(names) == 32
        assert "runner_macos_build_queued" in names
        assert "runner_total_busy" in names

    def test_should_name_the_wait_of_every_pool_and_the_total(self) -> None:
        names = wait_channels()
        assert len(names) == 8
        assert "runner_win_test_queue_wait" in names
        assert "runner_total_queue_wait" in names


class TestPoolCounts:
    """Tests for the queued jobs and runner counts of the pools."""

    def test_should_count_each_runner_once_per_pool(self) -> None:
        values = pool_counts(
            status={"a": BUSY, "b": IDLE, "c": OFFLINE, "d": OFFLINE, "x": IDLE},
            labels={"a": BUILD, "b": BUILD, "c": BUILD, "d": BUILD, "x": {"other"}},
            queued={},
            booting={"c"},
        )
        assert values["runner_ubuntu_build_busy"] == 1
        assert values["runner_ubuntu_build_idle"] == 1
        assert values["runner_ubuntu_build_starting"] == 1
        assert values["runner_ubuntu_build_offline"] == 1

    def test_should_total_the_pools(self) -> None:
        values = pool_counts(
            status={"a": BUSY, "m": BUSY, "n": OFFLINE},
            labels={"a": BUILD, "m": MAC, "n": MAC},
            queued={"ubuntu-build-bot": 3, "macos-build-bot": 2},
            booting=set(),
        )
        assert values["runner_ubuntu_build_queued"] == 3
        assert values["runner_macos_build_queued"] == 2
        assert values["runner_total_queued"] == 5
        assert values["runner_total_busy"] == 2
        assert values["runner_total_offline"] == 1

    def test_should_count_only_the_queue_of_an_unlisted_pool(self) -> None:
        values = pool_counts(
            status={"r": BUSY},
            labels={"r": REVIEW},
            queued={"review": 2},
            booting=set(),
        )
        assert values["runner_review_queued"] == 2
        assert values["runner_total_queued"] == 2
        assert values["runner_total_busy"] == 0

    def test_should_report_zero_without_runners_or_jobs(self) -> None:
        values = pool_counts(status={}, labels={}, queued={}, booting=set())
        assert set(values.values()) == {0}


class TestPoolWaits:
    """Tests for the queue wait of the pools."""

    def test_should_report_the_longest_wait_as_the_total(self) -> None:
        values = pool_waits({"ubuntu-build-bot": 40, "macos-build-bot": 90})
        assert values["runner_ubuntu_build_queue_wait"] == 40
        assert values["runner_macos_build_queue_wait"] == 90
        assert values["runner_total_queue_wait"] == 90

    def test_should_report_zero_without_a_queue(self) -> None:
        assert set(pool_waits({}).values()) == {0}


QUEUED = "runner_total_queued"
WAIT = "runner_total_queue_wait"


def report(client: Client) -> PoolReport:
    publisher = Publisher(
        client=core(client),
        name="pools",
        channels=[POOL_INDEX, *count_channels(), WAIT_INDEX, *wait_channels()],
        report=lambda *_: None,
    )
    return PoolReport(publisher=publisher)


class TestPoolReport:
    """Tests for writing the pool counts and waits on change."""

    def test_should_write_nothing_until_ready(self, client: Client) -> None:
        report(client).step(
            ready=False, counts={QUEUED: 0}, waits={WAIT: 0}, now=sy.TimeStamp.now()
        )
        assert client.frames == []

    def test_should_write_the_counts_and_the_waits_once_ready(
        self, client: Client
    ) -> None:
        report(client).step(
            ready=True, counts={QUEUED: 2}, waits={WAIT: 30}, now=sy.TimeStamp.now()
        )
        assert client.written(QUEUED) == [2]
        assert client.written(WAIT) == [30]

    def test_should_write_only_on_change(self, client: Client) -> None:
        r = report(client)
        for _ in range(2):
            r.step(
                ready=True, counts={QUEUED: 2}, waits={WAIT: 30}, now=sy.TimeStamp.now()
            )
        assert len(client.frames) == 1

    def test_should_write_again_after_a_failed_write(self, client: Client) -> None:
        r = report(client)
        client.failing = True
        r.step(ready=True, counts={QUEUED: 2}, waits={WAIT: 30}, now=sy.TimeStamp.now())
        client.failing = False
        r.step(ready=True, counts={QUEUED: 2}, waits={WAIT: 30}, now=sy.TimeStamp.now())
        assert client.written(QUEUED) == [2]
