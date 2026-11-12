#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from common import BUSY, IDLE, OFFLINE
from pools import count_channels, pool_counts, pool_of, pool_waits, wait_channels

BUILD = {"self-hosted", "ubuntu-build-bot"}
MAC = {"self-hosted", "macos-build-bot"}


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
        assert len(names) == 45
        assert "runner_macos_build_queued" in names
        assert "runner_total_busy" in names

    def test_should_name_the_wait_of_every_pool_and_the_total(self) -> None:
        names = wait_channels()
        assert len(names) == 9
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
