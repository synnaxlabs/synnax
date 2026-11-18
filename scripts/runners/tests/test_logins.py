#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

import pytest

from conftest import Secrets
from logins import LIFETIME, MAX_HOLDERS, PREFIX, Login, Logins

NOW = 1_800_000_000
YEAR = 365 * 24 * 3600


def logins(secrets: Secrets) -> Logins:
    return Logins(secrets=secrets, role="AROAROLE")


def login(**fields: Any) -> Login:
    return Login(**{"name": "claude-1", "arn": "a", "agent": "claude"} | fields)


class TestLogin:
    """Tests for whether a seat may start on a login."""

    def test_should_be_usable_before_it_expires(self) -> None:
        assert login(expires=NOW + 1).usable(now=NOW)

    def test_should_not_be_usable_once_it_expires(self) -> None:
        assert not login(expires=NOW).usable(now=NOW)

    def test_should_not_be_usable_while_limited(self) -> None:
        limited = login(expires=NOW + YEAR, limited_until=NOW + 1)
        assert not limited.usable(now=NOW)
        assert limited.usable(now=NOW + 1)

    def test_should_not_be_usable_when_broken(self) -> None:
        assert not login(expires=NOW + YEAR, broken=True).usable(now=NOW)

    def test_should_not_be_usable_with_the_most_holders(self) -> None:
        full = login(expires=NOW + YEAR, holders=["h"] * MAX_HOLDERS)
        assert not full.usable(now=NOW)

    def test_should_be_stored_a_lifetime_before_it_expires(self) -> None:
        assert login(expires=NOW + LIFETIME).stored == NOW


class TestLogins:
    """Tests for reading logins and writing their tags."""

    def test_should_list_the_logins_from_their_tags(self) -> None:
        secrets = Secrets()
        secrets.tags[f"{PREFIX}claude-1"] = {
            "agent": "claude",
            "expires": str(NOW),
            "limited-until": "5",
            "broken": "",
            "holders": "AROAROLE:i-1 AROAROLE:i-2",
        }
        secrets.tags[f"{PREFIX}claude-2"] = {"agent": "claude", "expires": str(NOW)}
        secrets.tags["other"] = {}
        assert logins(secrets).read() == {
            "claude-1": Login(
                name="claude-1",
                arn=f"arn:{PREFIX}claude-1",
                agent="claude",
                expires=NOW,
                limited_until=5,
                broken=True,
                holders=["AROAROLE:i-1", "AROAROLE:i-2"],
            ),
            "claude-2": Login(
                name="claude-2",
                arn=f"arn:{PREFIX}claude-2",
                agent="claude",
                expires=NOW,
            ),
        }

    def test_should_skip_a_login_deleted_after_the_list(self) -> None:
        secrets = Secrets()
        for name in ("claude-1", "claude-2", "claude-3"):
            secrets.tags[f"{PREFIX}{name}"] = {"agent": "claude", "expires": str(NOW)}
        secrets.deleted.add(f"{PREFIX}claude-1")
        secrets.gone.add(f"{PREFIX}claude-2")
        assert list(logins(secrets).read()) == ["claude-3"]

    def test_should_fail_on_a_login_without_an_expiry(self) -> None:
        secrets = Secrets()
        secrets.tags[f"{PREFIX}claude-1"] = {"agent": "claude"}
        with pytest.raises(KeyError):
            logins(secrets).read()

    def test_should_fail_on_a_login_with_an_expiry_that_is_not_whole(self) -> None:
        secrets = Secrets()
        secrets.tags[f"{PREFIX}claude-1"] = {"agent": "claude", "expires": "1.5"}
        with pytest.raises(ValueError, match="invalid literal"):
            logins(secrets).read()

    def test_should_name_a_holder_by_the_session_of_its_seat(self) -> None:
        assert logins(Secrets()).holder("i-1") == "AROAROLE:i-1"

    def test_should_write_and_clear_the_holders(self) -> None:
        secrets = Secrets()
        name = f"{PREFIX}claude-1"
        secrets.tags[name] = {"agent": "claude", "expires": str(NOW)}
        logs = logins(secrets)
        found = logs.read()["claude-1"]
        logs.hold(found, holders=["AROAROLE:i-1", "AROAROLE:i-2"])
        assert secrets.tags[name]["holders"] == "AROAROLE:i-1 AROAROLE:i-2"
        logs.hold(found, holders=[])
        assert "holders" not in secrets.tags[name]
        assert found.holders == []

    def test_should_record_a_usage_limit(self) -> None:
        secrets = Secrets()
        name = f"{PREFIX}claude-1"
        secrets.tags[name] = {"agent": "claude", "expires": str(NOW)}
        logs = logins(secrets)
        found = logs.read()["claude-1"]
        logs.limit(found, until=NOW + 60)
        assert secrets.tags[name]["limited-until"] == str(NOW + 60)
        assert found.limited_until == NOW + 60

    def test_should_record_a_rejected_token(self) -> None:
        secrets = Secrets()
        name = f"{PREFIX}claude-1"
        secrets.tags[name] = {"agent": "claude", "expires": str(NOW)}
        logs = logins(secrets)
        found = logs.read()["claude-1"]
        logs.reject(found, at=NOW - 60)
        assert secrets.tags[name]["broken"] == str(NOW - 60)
        assert found.broken

    def test_should_keep_the_login_when_a_write_fails(self) -> None:
        secrets = Secrets()
        secrets.tags[f"{PREFIX}claude-1"] = {"agent": "claude", "expires": str(NOW)}
        logs = logins(secrets)
        found = logs.read()["claude-1"]
        secrets.failing = True
        with pytest.raises(PermissionError):
            logs.hold(found, holders=["AROAROLE:i-1"])
        with pytest.raises(PermissionError):
            logs.limit(found, until=NOW + 60)
        with pytest.raises(PermissionError):
            logs.reject(found, at=NOW - 60)
        assert found.holders == []
        assert found.limited_until == 0
        assert not found.broken
