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

"""Stops the agent, or installs it at a commit, on every running runner over SSM.

    uv run agent_deploy.py stop
    uv run agent_deploy.py install --commit <sha> --ca-cert <path>

Run on Sandbox-Windows by the Deploy - Runners workflow. A runner that is not running
gets the deployed commit from agent_boot.py when it starts.
"""

import argparse
import base64
from collections.abc import Callable
from pathlib import Path
from urllib.request import Request, urlopen

import synnax as sy

from aws import AWS

SOURCE = "https://raw.githubusercontent.com/synnaxlabs/synnax/{}/scripts/runners"
IMDS = "http://169.254.169.254/latest"
POLL = 5 * sy.TimeSpan.SECOND
TIMEOUT = 5 * sy.TimeSpan.MINUTE
UNFINISHED = {"Pending", "InProgress", "Delayed"}

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


def run(aws: AWS, *, linux: list[str], windows: list[str]) -> bool:
    """Runs the commands on every running runner and prints a line for each runner.

    :returns: whether the commands succeeded on every running runner.
    """
    runners = {i: r for i, r in aws.instances().items() if "RunnerOS" in r["tags"]}
    results: dict[str, str] = {}
    outputs: dict[str, str] = {}
    sent: dict[str, str] = {}
    for iid, info in runners.items():
        if not info["running"]:
            results[iid] = info["state"]
            continue
        try:
            sent[iid] = aws.run(iid, info=info, linux=linux, windows=windows)
        # SSM rejects an instance whose SSM agent is not up yet.
        except Exception as e:
            results[iid] = f"not sent, {type(e).__name__}"
    timer = sy.Timer()
    while sent:
        for iid, command in list(sent.items()):
            status, output = invocation(
                aws, region=runners[iid]["region"], command=command, iid=iid
            )
            if status in UNFINISHED and timer.elapsed() < TIMEOUT:
                continue
            results[iid], outputs[iid] = status, output
            del sent[iid]
        if sent:
            sy.sleep(POLL)
    for iid in sorted(runners):
        name = runners[iid]["tags"].get("Name", "")
        print(f"{iid} {name}: {results[iid]}")
        if results[iid] != "Success" and outputs.get(iid):
            print(outputs[iid])
    return all(results[iid] == "Success" for iid, r in runners.items() if r["running"])


def main() -> int:
    p = argparse.ArgumentParser(description=(__doc__ or "").split("\n", 1)[0])
    actions = p.add_subparsers(dest="action", required=True)
    actions.add_parser("stop")
    install = actions.add_parser("install")
    install.add_argument("--commit", required=True)
    install.add_argument("--ca-cert", required=True, type=Path)
    args = p.parse_args()
    if args.action == "stop":
        linux, windows = STOP_LINUX, STOP_WINDOWS
    else:
        linux, windows = install_commands(
            commit=args.commit,
            host=public_address(metadata),
            ca=base64.b64encode(args.ca_cert.read_bytes()).decode(),
        )
    return 0 if run(AWS(), linux=linux, windows=windows) else 1


if __name__ == "__main__":
    raise SystemExit(main())
