#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ec2_off
from github import GitHub

RUNNER = "ubuntu-build-bot-i-010a"
PATH = f"/actions/runners?name={RUNNER}"


class Runners(GitHub):
    """Answers the runner lookup with one runner in the given state."""

    def __init__(self, *, status: str, busy: bool = False) -> None:
        super().__init__("1", "")
        self.runner = {"name": RUNNER, "status": status, "busy": busy}

    def call(self, path: str, *, method: str = "GET") -> dict:
        assert path == PATH
        return {"runners": [self.runner]}


class Instances:
    """Stands in for AWS with one running instance, recording what is stopped."""

    def __init__(self) -> None:
        self.stopped: list[str] = []
        self.ec2 = {"us-east-2": self}

    def instance(self, runner: str) -> tuple[str, dict]:
        return "i-010a6a6f95c268b59", {"region": "us-east-2"}

    def stop_instances(self, *, InstanceIds: list[str]) -> None:
        self.stopped.extend(InstanceIds)


class TestRun:
    def test_should_stop_a_runner_github_lists_offline(self) -> None:
        aws = Instances()
        ec2_off.run(RUNNER, aws=aws, github=Runners(status="offline"))
        assert aws.stopped == ["i-010a6a6f95c268b59"]

    def test_should_not_stop_an_online_runner_that_can_take_a_job(self) -> None:
        aws = Instances()
        line = ec2_off.run(RUNNER, aws=aws, github=Runners(status="online"))
        assert aws.stopped == []
        assert line is not None
        assert "online" in line

    def test_should_not_stop_a_runner_whose_job_still_runs(self) -> None:
        aws = Instances()
        github = Runners(status="offline", busy=True)
        line = ec2_off.run(RUNNER, aws=aws, github=github)
        assert aws.stopped == []
        assert line is not None
