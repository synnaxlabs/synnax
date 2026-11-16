#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
from datetime import UTC, datetime, timedelta
from typing import Any

import pytest

from login_expiry import PREFIX, check

NOW = datetime(2026, 11, 15, 10, 17, tzinfo=UTC)


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
