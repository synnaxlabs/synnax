#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, cast

import pytest

from conftest import Secrets
from logins import MAX_HOLDERS, PREFIX, Login, Logins

NOW = 1_800_000_000
YEAR = 365 * 24 * 3600


def logins(secrets: Secrets) -> Logins:
    return Logins(secrets=cast(Any, secrets), role="AROAROLE")


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

    def test_should_fail_on_a_login_without_an_expiry(self) -> None:
        secrets = Secrets()
        secrets.tags[f"{PREFIX}claude-1"] = {"agent": "claude"}
        with pytest.raises(KeyError):
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
