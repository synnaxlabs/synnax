#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
import sys
from pathlib import Path
from typing import cast
from urllib.request import Request

import pytest
import synnax as sy

import agent_deploy
from agent_deploy import (
    POLL,
    STOP_LINUX,
    STOP_WINDOWS,
    install_commands,
    public_address,
    run,
)
from aws import AWS

COMMIT = "a" * 40
UBUNTU = "i-0aaa"
WINDOWS = "i-0bbb"
LINUX_COMMANDS = ["linux"]
WINDOWS_COMMANDS = ["windows"]


def instance(
    name: str, *, windows: bool = False, state: str = "running", runner: bool = True
) -> dict:
    tags = {"Name": name}
    if runner:
        tags["RunnerOS"] = "windows" if windows else "ubuntu"
    return {
        "region": "us-east-2",
        "state": state,
        "running": state == "running",
        "windows": windows,
        "tags": tags,
    }


class Fleet:
    """Stands in for AWS: holds the instances, records the commands each one gets, and
    answers the statuses queued for an instance, the last one for good. A status of None
    is a command that SSM does not list yet. A started instance runs at the next look,
    and SSM rejects an instance for the number of sends in rejections."""

    def __init__(self, **instances: dict) -> None:
        self.found = {i.replace("_", "-"): info for i, info in instances.items()}
        self.sent: dict[str, list[str]] = {}
        self.statuses: dict[str, list[str | None]] = {}
        self.outputs: dict[str, str] = {}
        self.rejections: dict[str, int] = {}
        self.started: list[str] = []
        self.full = False
        self.ssm = {"us-east-2": self}
        self.ec2 = {"us-east-2": self}

    def instances(self) -> dict[str, dict]:
        for info in self.found.values():
            if info["state"] == "pending":
                info.update(state="running", running=True)
        return self.found

    def start_instances(self, *, InstanceIds: list[str]) -> None:
        if self.full:
            raise RuntimeError("InsufficientInstanceCapacity")
        self.started.extend(InstanceIds)
        for iid in InstanceIds:
            self.found[iid]["state"] = "pending"

    def run(self, iid: str, *, info: dict, linux: list[str], windows: list[str]) -> str:
        if self.rejections.get(iid):
            self.rejections[iid] -= 1
            raise ValueError("InvalidInstanceId")
        self.sent[iid] = windows if info["windows"] else linux
        return f"command-{iid}"

    def list_command_invocations(
        self, *, CommandId: str, InstanceId: str, Details: bool
    ) -> dict:
        assert CommandId == f"command-{InstanceId}"
        queue = self.statuses.get(InstanceId, ["Success"])
        status = queue.pop(0) if len(queue) > 1 else queue[0]
        if status is None:
            return {"CommandInvocations": []}
        plugin = {"Output": self.outputs.get(InstanceId, "")}
        return {"CommandInvocations": [{"Status": status, "CommandPlugins": [plugin]}]}


def send(fleet: Fleet, *, start_stopped: bool = False) -> bool:
    return run(
        cast(AWS, fleet),
        linux=LINUX_COMMANDS,
        windows=WINDOWS_COMMANDS,
        start_stopped=start_stopped,
    )


@pytest.fixture(autouse=True)
def sleeps(monkeypatch: pytest.MonkeyPatch) -> list[sy.TimeSpan]:
    """The spans the script sleeps for, which take no time in any test."""
    spans: list[sy.TimeSpan] = []
    monkeypatch.setattr(sy, "sleep", spans.append)
    return spans


@pytest.fixture
def fleet() -> Fleet:
    return Fleet(
        i_0aaa=instance("ubuntu-build-bot"),
        i_0bbb=instance("windows-build-bot", windows=True),
    )


class TestRun:
    """Tests for running commands on the runners of the fleet."""

    def test_should_send_each_running_runner_the_commands_of_its_system(
        self,
        fleet: Fleet,
        sleeps: list[sy.TimeSpan],
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        assert send(fleet)
        assert fleet.sent == {UBUNTU: LINUX_COMMANDS, WINDOWS: WINDOWS_COMMANDS}
        assert capsys.readouterr().out == (
            "i-0aaa ubuntu-build-bot: Success\ni-0bbb windows-build-bot: Success\n"
        )

    def test_should_skip_an_instance_that_is_not_a_runner(self) -> None:
        fleet = Fleet(i_0ccc=instance("Sandbox-Windows", windows=True, runner=False))
        assert send(fleet, start_stopped=True)
        assert fleet.sent == {}

    def test_should_pass_over_a_runner_that_is_not_running(
        self,
        fleet: Fleet,
        sleeps: list[sy.TimeSpan],
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        fleet.found[WINDOWS] = instance("windows-build-bot", state="stopped")
        assert send(fleet)
        assert list(fleet.sent) == [UBUNTU]
        assert fleet.started == []
        assert "i-0bbb windows-build-bot: stopped\n" in capsys.readouterr().out

    def test_should_wait_until_each_command_ends(
        self, fleet: Fleet, sleeps: list[sy.TimeSpan]
    ) -> None:
        fleet.statuses[UBUNTU] = [None, "Pending", "InProgress", "Success"]
        assert send(fleet)
        assert sleeps == [POLL, POLL, POLL, POLL]

    def test_should_fail_and_print_the_output_of_a_failed_command(
        self,
        fleet: Fleet,
        sleeps: list[sy.TimeSpan],
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        fleet.statuses[WINDOWS] = ["Failed"]
        fleet.outputs[WINDOWS] = "RunnerAgent did not stay up\n"
        assert not send(fleet)
        assert capsys.readouterr().out.endswith(
            "i-0bbb windows-build-bot: Failed\nRunnerAgent did not stay up\n"
        )

    def test_should_send_again_until_ssm_takes_the_runner(
        self, fleet: Fleet, sleeps: list[sy.TimeSpan]
    ) -> None:
        fleet.rejections[UBUNTU] = 2
        assert send(fleet)
        assert fleet.sent[UBUNTU] == LINUX_COMMANDS

    def test_should_fail_a_runner_that_ssm_does_not_take_in_time(
        self,
        fleet: Fleet,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        monkeypatch.setattr(agent_deploy, "TIMEOUT", sy.TimeSpan(0))
        fleet.rejections[UBUNTU] = 1
        fleet.statuses[WINDOWS] = ["InProgress"]
        assert not send(fleet)
        out = capsys.readouterr().out
        assert "i-0aaa ubuntu-build-bot: not sent, ValueError\n" in out
        assert "i-0bbb windows-build-bot: InProgress\n" in out


class TestStart:
    """Tests for starting the stopped runners before an install."""

    def test_should_start_a_stopped_runner_and_send_it_the_commands(
        self,
        fleet: Fleet,
        sleeps: list[sy.TimeSpan],
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        fleet.found[WINDOWS] = instance(
            "windows-build-bot", windows=True, state="stopped"
        )
        assert send(fleet, start_stopped=True)
        assert fleet.started == [WINDOWS]
        assert fleet.sent[WINDOWS] == WINDOWS_COMMANDS
        assert capsys.readouterr().out.endswith(
            "Started 1 runners. The scaler stops the idle ones.\n"
        )

    def test_should_start_a_runner_once_it_is_stopped(
        self, fleet: Fleet, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        fleet.found[UBUNTU] = instance("ubuntu-build-bot", state="stopping")

        def sleep(span: sy.TimeSpan) -> None:
            if fleet.found[UBUNTU]["state"] == "stopping":
                fleet.found[UBUNTU]["state"] = "stopped"

        monkeypatch.setattr(sy, "sleep", sleep)
        assert send(fleet, start_stopped=True)
        assert fleet.started == [UBUNTU]

    def test_should_skip_a_terminated_runner(
        self, fleet: Fleet, sleeps: list[sy.TimeSpan]
    ) -> None:
        fleet.found[UBUNTU] = instance("ubuntu-build-bot", state="terminated")
        assert send(fleet, start_stopped=True)
        assert fleet.started == []
        assert list(fleet.sent) == [WINDOWS]

    def test_should_fail_a_runner_that_does_not_start(
        self,
        fleet: Fleet,
        sleeps: list[sy.TimeSpan],
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        fleet.found[UBUNTU] = instance("ubuntu-build-bot", state="stopped")
        fleet.full = True
        assert not send(fleet, start_stopped=True)
        assert "i-0aaa ubuntu-build-bot: not started, RuntimeError\n" in (
            capsys.readouterr().out
        )

    def test_should_fail_a_runner_that_does_not_stop_in_time(
        self,
        fleet: Fleet,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        monkeypatch.setattr(agent_deploy, "TIMEOUT", sy.TimeSpan(0))
        fleet.found[UBUNTU] = instance("ubuntu-build-bot", state="stopping")
        assert not send(fleet, start_stopped=True)
        assert "i-0aaa ubuntu-build-bot: stopping\n" in capsys.readouterr().out


class TestInstallCommands:
    """Tests for the commands that install the agent on a runner."""

    def test_should_run_the_install_script_of_the_commit_on_each_system(self) -> None:
        linux, windows = install_commands(commit=COMMIT, host="203.0.113.7", ca="Y2E=")
        source = f"https://raw.githubusercontent.com/synnaxlabs/synnax/{COMMIT}"
        assert linux == [
            f"curl -fsSL -o /tmp/agent_install.sh {source}/scripts/runners/"
            f"agent_install.sh && bash /tmp/agent_install.sh {COMMIT} 203.0.113.7 Y2E="
        ]
        assert f"'{source}/scripts/runners/agent_install.ps1'" in windows[-1]
        assert f"-Commit '{COMMIT}' -CoreHost '203.0.113.7' -Ca 'Y2E='" in windows[-1]
        assert windows[-1].endswith("catch { Write-Output $_; exit 1 }")


class TestPublicAddress:
    """Tests for reading the public address of the box."""

    def test_should_ask_the_metadata_service_with_a_session_token(self) -> None:
        requests: list[Request] = []

        def fetch(request: Request) -> bytes:
            requests.append(request)
            return b"token" if request.get_method() == "PUT" else b"203.0.113.7\n"

        assert public_address(fetch) == "203.0.113.7"
        token, address = requests
        assert token.full_url == "http://169.254.169.254/latest/api/token"
        assert token.get_header("X-aws-ec2-metadata-token-ttl-seconds") == "60"
        assert address.full_url.endswith("/meta-data/public-ipv4")
        assert address.get_header("X-aws-ec2-metadata-token") == "token"


def start(monkeypatch: pytest.MonkeyPatch, fleet: Fleet, *argv: str) -> int:
    monkeypatch.setattr(sys, "argv", ["agent_deploy.py", *argv])
    monkeypatch.setattr(agent_deploy, "AWS", lambda: fleet)
    monkeypatch.setattr(agent_deploy, "metadata", lambda request: b"203.0.113.7")
    return agent_deploy.main()


class TestMain:
    """Tests for the two actions of the script."""

    def test_should_stop_the_agent_on_each_runner(
        self, fleet: Fleet, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        assert start(monkeypatch, fleet, "stop") == 0
        assert fleet.sent == {UBUNTU: STOP_LINUX, WINDOWS: STOP_WINDOWS}

    def test_should_install_the_agent_with_the_certificate_of_the_core(
        self, fleet: Fleet, monkeypatch: pytest.MonkeyPatch, tmp_path: Path
    ) -> None:
        cert = tmp_path / "ca.crt"
        cert.write_bytes(b"certificate")
        argv = ("install", "--commit", COMMIT, "--ca-cert", str(cert))
        assert start(monkeypatch, fleet, *argv) == 0
        ca = base64.b64encode(b"certificate").decode()
        assert fleet.sent[UBUNTU][0].endswith(f"{COMMIT} 203.0.113.7 {ca}")

    def test_should_exit_with_an_error_when_a_runner_fails(
        self, fleet: Fleet, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        fleet.statuses[UBUNTU] = ["Failed"]
        assert start(monkeypatch, fleet, "stop") == 1
