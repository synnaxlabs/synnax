#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
import os
import subprocess
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "run_ssm_powershell.sh"

POWERSHELL = "Write-Output 'done'\n"
MISSING = (
    "An error occurred (InvocationDoesNotExist) when calling the "
    "GetCommandInvocation operation"
)
DENIED = (
    "An error occurred (AccessDeniedException) when calling the "
    "GetCommandInvocation operation"
)

# Answers each status poll with the next line of STATUSES and repeats the last one.
AWS = """#!/usr/bin/env bash
echo "aws $*" >> "$LOG"
case "$2" in
    send-command) echo command-1 ;;
    get-command-invocation)
        if [[ "$*" == *StandardOutputContent* ]]; then
            echo "script output"
            exit 0
        fi
        line=$(head -n 1 "$STATUSES")
        if [ "$(wc -l < "$STATUSES")" -gt 1 ]; then
            tail -n +2 "$STATUSES" > "$STATUSES.next"
            mv "$STATUSES.next" "$STATUSES"
        fi
        case "$line" in
            error:*) echo "${line#error: }" >&2; exit 254 ;;
            *) echo "$line" ;;
        esac
        ;;
esac
"""

SLEEP = """#!/usr/bin/env bash
echo "sleep $1" >> "$LOG"
"""


def run(
    tmp_path: Path,
    statuses: list[str],
    timeout: int = 900,
    variables: tuple[str, ...] = (),
) -> tuple[subprocess.CompletedProcess[str], list[str]]:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    for name, body in (("aws", AWS), ("sleep", SLEEP)):
        tool = bin_dir / name
        tool.write_text(body)
        tool.chmod(0o755)
    statuses_file = tmp_path / "statuses"
    statuses_file.write_text("".join(f"{status}\n" for status in statuses))
    log = tmp_path / "log"
    powershell = tmp_path / "script.ps1"
    powershell.write_text(POWERSHELL)
    result = subprocess.run(
        [str(SCRIPT), str(powershell), *variables],
        env={
            **os.environ,
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "LOG": str(log),
            "STATUSES": str(statuses_file),
            "INSTANCE": "i-123",
            "COMMENT": "Back up the Core",
            "TIMEOUT": str(timeout),
        },
        capture_output=True,
        text=True,
    )
    return result, log.read_text().splitlines()


def sleeps(calls: list[str]) -> int:
    return sum(call.startswith("sleep") for call in calls)


class TestRunSSMPowerShell:
    """Tests for running a PowerShell script on a Windows instance through SSM."""

    def test_should_send_the_variables_and_the_script_as_an_encoded_command(
        self, tmp_path: Path
    ) -> None:
        _, calls = run(tmp_path, ["Success"], variables=("Step=stop", "Url=a'b=c"))
        script = "$Step = 'stop'\n$Url = 'a''b=c'\n" + POWERSHELL
        encoded = base64.b64encode(script.encode("utf-16-le")).decode()
        assert calls[0] == (
            "aws ssm send-command --instance-ids i-123 "
            "--document-name AWS-RunPowerShellScript --comment Back up the Core "
            '--parameters {"commands":["powershell -NoProfile -EncodedCommand '
            f'{encoded}","exit $LASTEXITCODE"],"executionTimeout":["900"]}} '
            "--query Command.CommandId --output text"
        )

    def test_should_print_the_output_of_a_script_that_succeeds(
        self, tmp_path: Path
    ) -> None:
        result, calls = run(tmp_path, ["InProgress", "InProgress", "Success"])
        assert result.returncode == 0
        assert "script output" in result.stdout
        assert sleeps(calls) == 2

    def test_should_fail_and_print_the_output_of_a_script_that_fails(
        self, tmp_path: Path
    ) -> None:
        result, _ = run(tmp_path, ["InProgress", "Failed"])
        assert result.returncode == 1
        assert "script output" in result.stdout

    def test_should_wait_for_an_invocation_that_is_not_there_yet(
        self, tmp_path: Path
    ) -> None:
        result, calls = run(tmp_path, [f"error: {MISSING}", "Success"])
        assert result.returncode == 0
        assert sleeps(calls) == 1

    def test_should_fail_at_once_on_any_other_error(self, tmp_path: Path) -> None:
        result, calls = run(tmp_path, [f"error: {DENIED}"])
        assert result.returncode == 1
        assert DENIED in result.stderr
        assert sleeps(calls) == 0

    def test_should_fail_when_the_script_outlasts_the_timeout(
        self, tmp_path: Path
    ) -> None:
        result, calls = run(tmp_path, ["InProgress"], timeout=30)
        assert result.returncode == 1
        assert "still InProgress after 30 seconds" in result.stderr
        assert sleeps(calls) == 3
