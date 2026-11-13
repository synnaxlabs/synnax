#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import subprocess
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "sign_macos_binary.sh"

# Logs the codesign flags that name the entitlements file, and that file's keys.
CODESIGN = """#!/usr/bin/env bash
while [ $# -gt 0 ]; do
    if [ "$1" = "--entitlements" ]; then
        grep -o 'cs\\.[a-z-]*' "$2" | sed 's/^cs\\./entitlement /' >> "$LOG"
    fi
    shift
done
"""

NOOP = """#!/usr/bin/env bash
"""

# Stands in for notarytool. A submit or a wait fails while its fails file holds a
# count above zero, and counts it down.
XCRUN = """#!/usr/bin/env bash
command=$2
if [ "$command" = "wait" ]; then
    echo "wait $3" >> "$LOG"
else
    echo "$command" >> "$LOG"
fi
fails=$(cat "$STATE/$command" 2>/dev/null || echo 0)
if [ "$fails" -gt 0 ]; then
    echo $((fails - 1)) > "$STATE/$command"
    echo '{"message":"failed"}'
    exit 1
fi
if [ "$command" = "submit" ]; then
    echo '{"id":"abc"}'
else
    echo "{\\"id\\":\\"abc\\",\\"status\\":\\"$STATUS\\"}"
fi
"""

SLEEP = """#!/usr/bin/env bash
echo "sleep $1" >> "$LOG"
"""

TOOLS = (
    ("codesign", CODESIGN),
    ("security", NOOP),
    ("zip", NOOP),
    ("xcrun", XCRUN),
    ("sleep", SLEEP),
)


def sign(
    tmp_path: Path,
    *entitlements: str,
    submit_fails: int = 0,
    wait_fails: int = 0,
    status: str = "Accepted",
) -> tuple[subprocess.CompletedProcess[str], list[str]]:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    for name, body in TOOLS:
        tool = bin_dir / name
        tool.write_text(body)
        tool.chmod(0o755)
    state = tmp_path / "state"
    state.mkdir()
    (state / "submit").write_text(str(submit_fails))
    (state / "wait").write_text(str(wait_fails))
    log = tmp_path / "log"
    result = subprocess.run(
        [str(SCRIPT), "binary", *entitlements],
        cwd=tmp_path,
        env={
            **os.environ,
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "LOG": str(log),
            "STATE": str(state),
            "STATUS": status,
            "KEYCHAIN_NAME": "build.keychain",
            "KEYCHAIN_PASSWORD": "keychain-password",
            "CERT_ID": "Developer ID Application",
            "APPLE_ID": "apple-id",
            "APPLE_PASSWORD": "apple-password",
            "APPLE_TEAM_ID": "team",
        },
        capture_output=True,
        text=True,
    )
    calls = log.read_text().splitlines() if log.exists() else []
    return result, calls


class TestSignMacosBinary:
    """Tests for signing and notarizing a macOS binary."""

    def test_should_sign_with_each_entitlement_and_notarize_once(
        self, tmp_path: Path
    ) -> None:
        result, calls = sign(tmp_path, "allow-jit", "disable-library-validation")
        assert result.returncode == 0
        assert calls == [
            "entitlement allow-jit",
            "entitlement disable-library-validation",
            "submit",
            "wait abc",
        ]

    def test_should_send_a_failed_upload_again(self, tmp_path: Path) -> None:
        result, calls = sign(tmp_path, submit_fails=2)
        assert result.returncode == 0
        assert calls == [
            "submit",
            "sleep 10",
            "submit",
            "sleep 10",
            "submit",
            "wait abc",
        ]

    def test_should_fail_after_three_uploads(self, tmp_path: Path) -> None:
        result, calls = sign(tmp_path, submit_fails=3)
        assert result.returncode == 1
        assert calls == ["submit", "sleep 10", "submit", "sleep 10", "submit"]
        assert "Notarization upload failed after 3 attempts" in result.stderr

    def test_should_poll_the_same_submission_after_a_failed_wait(
        self, tmp_path: Path
    ) -> None:
        result, calls = sign(tmp_path, wait_fails=2)
        assert result.returncode == 0
        assert calls == [
            "submit",
            "wait abc",
            "sleep 10",
            "wait abc",
            "sleep 10",
            "wait abc",
        ]

    def test_should_fail_after_three_waits(self, tmp_path: Path) -> None:
        result, calls = sign(tmp_path, wait_fails=3)
        assert result.returncode == 1
        assert calls == [
            "submit",
            "wait abc",
            "sleep 10",
            "wait abc",
            "sleep 10",
            "wait abc",
        ]
        assert "Notarization of submission abc did not pass" in result.stderr

    def test_should_fail_when_the_status_is_not_accepted(self, tmp_path: Path) -> None:
        result, calls = sign(tmp_path, status="Invalid")
        assert result.returncode == 1
        assert calls == ["submit", "wait abc"]
        assert "Notarization of submission abc did not pass" in result.stderr

    def test_should_require_the_credentials(self, tmp_path: Path) -> None:
        result = subprocess.run(
            [str(SCRIPT), "binary"],
            cwd=tmp_path,
            env={"PATH": os.environ["PATH"]},
            capture_output=True,
            text=True,
        )
        assert result.returncode != 0
        assert "KEYCHAIN_NAME is required" in result.stderr
