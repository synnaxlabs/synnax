#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
import subprocess
import sys
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any

import pytest

from login_expiry import PREFIX, check

NOW = datetime(2026, 11, 15, 10, 17, tzinfo=UTC)

SCRIPT = Path(__file__).resolve().parents[1] / "login_expiry.py"

# Prints $LISTED for list-secrets and $DESCRIBED for describe-secret. Fails with $ERROR
# when it is set.
AWS = """#!/usr/bin/env bash
if [ -n "$ERROR" ]; then echo "$ERROR" >&2; exit 254; fi
case "$2" in
    list-secrets) echo "$LISTED" ;;
    describe-secret) echo "$DESCRIBED" ;;
esac
"""


def at(days: float) -> str:
    return str(int((NOW + timedelta(days=days)).timestamp()))


class FakeAws:
    """Answers the AWS CLI calls of the check from fixed logins, by name."""

    def __init__(
        self,
        logins: dict[str, dict[str, str]],
        *,
        deleted: tuple[str, ...] = (),
        purged: tuple[str, ...] = (),
        failing: tuple[str, ...] = (),
    ) -> None:
        self.logins = logins
        self.deleted = deleted
        self.purged = purged
        self.failing = failing

    def __call__(self, args: list[str]) -> Any:
        match args[:2]:
            case ["secretsmanager", "list-secrets"]:
                assert args[-1] == f"Key=name,Values={PREFIX}"
                return {"SecretList": [{"Name": PREFIX + n} for n in self.logins]}
            case ["secretsmanager", "describe-secret"]:
                login = args[-1].removeprefix(PREFIX)
                if login in self.purged:
                    raise subprocess.CalledProcessError(
                        254, args, stderr="ResourceNotFoundException"
                    )
                if login in self.failing:
                    raise subprocess.CalledProcessError(
                        254, args, stderr="AccessDeniedException"
                    )
                tags = [{"Key": k, "Value": v} for k, v in self.logins[login].items()]
                secret: dict[str, Any] = {"Tags": tags}
                if login in self.deleted:
                    secret["DeletedDate"] = "2026-11-15"
                return secret
        raise AssertionError(f"unexpected call {args}")


def test_logins_far_from_expiry_pass() -> None:
    ok, lines = check(
        FakeAws({"claude-2": {"expires": at(200)}, "claude-1": {"expires": at(31)}}),
        now=NOW,
    )
    assert ok
    assert lines == ["claude-1: expires 2026-12-16", "claude-2: expires 2027-06-03"]


def test_login_that_expires_within_the_warning_fails() -> None:
    ok, lines = check(FakeAws({"claude-1": {"expires": at(29)}}), now=NOW)
    assert not ok
    assert lines == ["claude-1: expires 2026-12-14, store a new token"]


def test_expired_login_fails() -> None:
    ok, lines = check(FakeAws({"claude-1": {"expires": at(-1)}}), now=NOW)
    assert not ok
    assert lines == ["claude-1: expired 2026-11-14, store a new token"]


def test_broken_login_fails() -> None:
    ok, lines = check(
        FakeAws({"claude-1": {"expires": at(200), "broken": ""}}), now=NOW
    )
    assert not ok
    assert lines == ["claude-1: broken, store a new token"]


def test_deleted_login_is_skipped() -> None:
    ok, lines = check(
        FakeAws(
            {"claude-1": {"expires": at(-5)}, "claude-2": {"expires": at(200)}},
            deleted=("claude-1",),
        ),
        now=NOW,
    )
    assert ok
    assert lines == ["claude-2: expires 2027-06-03"]


def test_purged_login_is_skipped() -> None:
    ok, lines = check(
        FakeAws(
            {"claude-1": {"expires": at(-5)}, "claude-2": {"expires": at(200)}},
            purged=("claude-1",),
        ),
        now=NOW,
    )
    assert ok
    assert lines == ["claude-2: expires 2027-06-03"]


def test_other_describe_error_raises() -> None:
    with pytest.raises(subprocess.CalledProcessError):
        check(
            FakeAws({"claude-1": {"expires": at(200)}}, failing=("claude-1",)), now=NOW
        )


def test_no_login_fails() -> None:
    assert check(FakeAws({}), now=NOW) == (False, ["no login is stored"])


def run(
    tmp_path: Path, *, expires: datetime, error: str = ""
) -> subprocess.CompletedProcess[str]:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    aws = bin_dir / "aws"
    aws.write_text(AWS)
    aws.chmod(0o755)
    tags = [{"Key": "expires", "Value": str(int(expires.timestamp()))}]
    return subprocess.run(
        [sys.executable, str(SCRIPT)],
        env={
            **os.environ,
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "LISTED": json.dumps({"SecretList": [{"Name": f"{PREFIX}claude-1"}]}),
            "DESCRIBED": json.dumps({"Tags": tags}),
            "ERROR": error,
        },
        capture_output=True,
        text=True,
    )


def test_script_passes_for_a_login_far_from_expiry(tmp_path: Path) -> None:
    expires = datetime.now(UTC) + timedelta(days=200)
    result = run(tmp_path, expires=expires)
    assert result.returncode == 0
    assert result.stdout == f"claude-1: expires {expires:%Y-%m-%d}\n"


def test_script_fails_for_an_expired_login(tmp_path: Path) -> None:
    expires = datetime.now(UTC) - timedelta(days=1)
    result = run(tmp_path, expires=expires)
    assert result.returncode == 1
    assert result.stdout == (
        f"claude-1: expired {expires:%Y-%m-%d}, store a new token\n"
    )


def test_script_prints_the_aws_error(tmp_path: Path) -> None:
    error = "An error occurred (AccessDeniedException) when calling ListSecrets"
    result = run(tmp_path, expires=datetime.now(UTC), error=error)
    assert result.returncode != 0
    assert error in result.stderr
