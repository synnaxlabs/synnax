#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# /// script
# requires-python = ">=3.12,<4"
# dependencies = ["boto3", "synnax>=0.61,<0.62"]
# ///

"""Stops the agent on every running runner, or installs it at a commit on every runner,
over SSM.

    uv run agent_deploy.py stop
    uv run agent_deploy.py install --commit <sha> --username controller --secure \
        --ca-cert <path>

Run on Sandbox-Windows by the Deploy - Runners workflow. The install starts each stopped
runner and leaves it up. It keeps the scaler off until it ends, so the scaler stops no
runner before its install, and stops the idle ones after.
"""

import argparse
import base64
from collections.abc import Callable, Iterator
from contextlib import contextmanager
from pathlib import Path
from urllib.request import Request, urlopen

import synnax as sy

from aws import AWS
from common import add_connection_args, connect, parameter_store
from scaler import SCALER_CMD, SCALER_ENABLED

SOURCE = "https://raw.githubusercontent.com/synnaxlabs/synnax/{}/scripts/runners"
IMDS = "http://169.254.169.254/latest"
POLL = 5 * sy.TimeSpan.SECOND
# Long enough for a stopped Windows runner to boot and take the commands.
TIMEOUT = 15 * sy.TimeSpan.MINUTE
UNFINISHED = {"Pending", "InProgress", "Delayed"}
GONE = {"shutting-down", "terminated"}
# The controller beats the scaler state every 5 s while it runs.
BEAT_WINDOW = 30 * sy.TimeSpan.SECOND
SWITCH_TIMEOUT = 1 * sy.TimeSpan.MINUTE

STOP_LINUX = [
    "if systemctl cat runner-agent.service > /dev/null 2>&1; then "
    "systemctl stop runner-agent.service; fi"
]
# Stopping the task alone leaves the agent processes up.
STOP_WINDOWS = [
    "Stop-ScheduledTask -TaskName 'RunnerAgent' -ErrorAction Ignore",
    "$agent = 'runner-telemetry\\\\agent\\.py|[ \"]agent(_boot)?\\.py'",
    "Get-CimInstance Win32_Process | Where-Object { "
    "$_.Name -match 'python|uv|cmd' -and $_.CommandLine -match $agent } | "
    "ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction Ignore }",
]


def metadata(request: Request) -> bytes:
    with urlopen(request, timeout=5) as response:
        return bytes(response.read())


def public_address(fetch: Callable[[Request], bytes]) -> str:
    """The public IPv4 address of this instance, where the agents reach the Core."""
    token = fetch(
        Request(
            f"{IMDS}/api/token",
            method="PUT",
            headers={"X-aws-ec2-metadata-token-ttl-seconds": "60"},
        )
    ).decode()
    address = fetch(
        Request(
            f"{IMDS}/meta-data/public-ipv4",
            headers={"X-aws-ec2-metadata-token": token},
        )
    )
    return address.decode().strip()


def install_commands(*, commit: str, host: str, ca: str) -> tuple[list[str], list[str]]:
    """The Linux and Windows commands that run the install script of the commit. ca is
    the certificate of the Core in base64."""
    source = SOURCE.format(commit)
    linux = [
        f"curl -fsSL -o /tmp/agent_install.sh {source}/agent_install.sh && "
        f"bash /tmp/agent_install.sh {commit} {host} {ca}"
    ]
    windows = [
        "[Net.ServicePointManager]::SecurityProtocol = "
        "[Net.SecurityProtocolType]::Tls12",
        "try { $install = (Invoke-WebRequest "
        f"'{source}/agent_install.ps1' -UseBasicParsing).Content; "
        f"& ([scriptblock]::Create($install)) -Commit '{commit}' -CoreHost '{host}' "
        f"-Ca '{ca}' }} catch {{ Write-Output $_; exit 1 }}",
    ]
    return linux, windows


def invocation(aws: AWS, *, region: str, command: str, iid: str) -> tuple[str, str]:
    """The status and the output of a command on an instance."""
    found = aws.ssm[region].list_command_invocations(
        CommandId=command, InstanceId=iid, Details=True
    )["CommandInvocations"]
    # SSM lists no invocation until the command reaches the instance.
    if not found:
        return "Pending", ""
    plugins = found[0]["CommandPlugins"]
    return found[0]["Status"], plugins[0]["Output"].strip() if plugins else ""


def run(
    aws: AWS, *, linux: list[str], windows: list[str], start_stopped: bool = False
) -> bool:
    """Runs the commands on every running runner and prints a line for each runner. With
    start_stopped, it first starts each stopped runner and leaves it up.

    :returns: whether the commands succeeded on every runner they were meant for.
    """
    results: dict[str, str] = {}
    outputs: dict[str, str] = {}
    sent: dict[str, str] = {}
    started: set[str] = set()
    skipped: set[str] = set()
    timer = sy.Timer()
    while True:
        runners = {i: r for i, r in aws.instances().items() if "RunnerOS" in r["tags"]}
        late = timer.elapsed() >= TIMEOUT
        for iid, info in runners.items():
            if iid in results:
                continue
            if iid in sent:
                status, output = invocation(
                    aws, region=info["region"], command=sent[iid], iid=iid
                )
                if status not in UNFINISHED or late:
                    results[iid], outputs[iid] = status, output
            elif info["running"]:
                try:
                    sent[iid] = aws.run(iid, info=info, linux=linux, windows=windows)
                # SSM rejects an instance until its SSM agent is up.
                except Exception as e:
                    if late:
                        results[iid] = f"not sent, {type(e).__name__}"
            elif not start_stopped or info["state"] in GONE:
                results[iid] = info["state"]
                skipped.add(iid)
            elif info["state"] == "stopped" and iid not in started:
                started.add(iid)
                try:
                    aws.ec2[info["region"]].start_instances(InstanceIds=[iid])
                except Exception as e:
                    results[iid] = f"not started, {type(e).__name__}"
            elif late:
                results[iid] = info["state"]
        if all(iid in results for iid in runners):
            break
        sy.sleep(POLL)
    for iid in sorted(runners):
        print(f"{iid} {runners[iid]['tags'].get('Name', '')}: {results[iid]}")
        if results[iid] != "Success" and outputs.get(iid):
            print(outputs[iid])
    if started:
        print(f"Started {len(started)} runners. The scaler stops the idle ones.")
    return all(results[i] == "Success" for i in runners if i not in skipped)


def scaler_enabled(client: sy.Synnax) -> bool | None:
    """The scaler state that the controller last reported, or None when the controller
    does not run."""
    now = sy.TimeStamp.now()
    beats = client.read(sy.TimeRange(now - BEAT_WINDOW, now), SCALER_ENABLED)
    return bool(beats[-1]) if len(beats) else None


def switch_scaler(client: sy.Synnax, *, enabled: bool) -> None:
    writer = client.open_writer(sy.TimeStamp.now(), [SCALER_CMD])
    try:
        writer.write({SCALER_CMD: enabled})
    finally:
        writer.close()


@contextmanager
def scaler_paused(client: sy.Synnax) -> Iterator[None]:
    """Keeps a scaler that is on turned off inside the block, then turns it on again.

    :raises TimeoutError: if the controller does not turn the scaler off.
    """
    if not scaler_enabled(client):
        yield
        return
    switch_scaler(client, enabled=False)
    try:
        timer = sy.Timer()
        while scaler_enabled(client):
            if timer.elapsed() >= SWITCH_TIMEOUT:
                raise TimeoutError("The controller did not turn the scaler off")
            sy.sleep(sy.TimeSpan.SECOND)
        yield
    finally:
        switch_scaler(client, enabled=True)


def main() -> int:
    p = argparse.ArgumentParser(description=(__doc__ or "").split("\n", 1)[0])
    actions = p.add_subparsers(dest="action", required=True)
    actions.add_parser("stop")
    install = actions.add_parser("install")
    install.add_argument("--commit", required=True)
    add_connection_args(install)
    args = p.parse_args()
    aws = AWS()
    if args.action == "stop":
        return 0 if run(aws, linux=STOP_LINUX, windows=STOP_WINDOWS) else 1
    linux, windows = install_commands(
        commit=args.commit,
        host=public_address(metadata),
        ca=base64.b64encode(Path(args.ca_cert).read_bytes()).decode(),
    )
    client = connect(args, store=parameter_store())
    try:
        with scaler_paused(client):
            passed = run(aws, linux=linux, windows=windows, start_stopped=True)
    finally:
        client.close()
    return 0 if passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
