#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import cast

import ec2_off
from aws import AWS
from github import GitHub

RUNNER = "ubuntu-build-bot-i-010a"
PATH = f"/actions/runners?name={RUNNER}"


class Runners(GitHub):
    """Answers the runner lookup with one runner in the given busy state."""

    def __init__(self, busy: bool) -> None:
        super().__init__("token")
        self.busy_flag = busy

    def get(self, path: str) -> dict:
        assert path == PATH
        return {"runners": [{"name": RUNNER, "busy": self.busy_flag}]}


class Instances:
    """Stands in for AWS with one running instance, recording what is stopped."""

    def __init__(self) -> None:
        self.stopped: list[str] = []
        self.ec2 = {"us-east-2": self}

    def instance(self, runner: str) -> tuple[str, dict]:
        return "i-010a6a6f95c268b59", {"region": "us-east-2"}

    def stop_instances(self, InstanceIds: list[str]) -> None:
        self.stopped.extend(InstanceIds)


class TestRun:
    def test_should_stop_an_idle_runner(self) -> None:
        aws = Instances()
        ec2_off.run(RUNNER, aws=cast(AWS, aws), github=Runners(busy=False))
        assert aws.stopped == ["i-010a6a6f95c268b59"]

    def test_should_not_stop_a_runner_github_lists_busy(self) -> None:
        aws = Instances()
        line = ec2_off.run(RUNNER, aws=cast(AWS, aws), github=Runners(busy=True))
        assert aws.stopped == []
        assert "busy" in line
