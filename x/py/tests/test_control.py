#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from x.control import (
    Authority,
    Concurrency,
    CrudeAuthority,
    State,
    Subject,
    Transfer,
    Update,
)


class TestAuthority:
    def test_absolute_value(self) -> None:
        assert Authority.ABSOLUTE == 255

    def test_default_value(self) -> None:
        assert Authority.DEFAULT == 1

    def test_is_int(self) -> None:
        assert isinstance(Authority.ABSOLUTE, int)

    def test_crude_authority_accepts_int(self) -> None:
        v: CrudeAuthority = 100
        assert v == 100


class TestConcurrency:
    def test_exclusive(self) -> None:
        assert Concurrency.exclusive.value == 0

    def test_shared(self) -> None:
        assert Concurrency.shared.value == 1


class TestSubject:
    def test_create(self) -> None:
        s = Subject(key="test-key", name="Test")
        assert s.key == "test-key"
        assert s.name == "Test"
        assert s.group is None

    def test_with_group(self) -> None:
        s = Subject(key="k", name="n", group=42)
        assert s.group == 42


class TestState:
    def test_create(self) -> None:
        s = State(
            subject=Subject(key="k", name="n"),
            resource="res-1",
            authority=100,
        )
        assert s.authority == 100
        assert s.resource == "res-1"


class TestTransfer:
    def test_empty_transfer(self) -> None:
        t = Transfer[str]()
        assert t.from_ is None
        assert t.to is None

    def test_acquire_transfer(self) -> None:
        state = State(
            subject=Subject(key="k", name="n"),
            resource="ch-1",
            authority=255,
        )
        t = Transfer[str](to=state)
        assert t.from_ is None
        assert t.to is not None
        assert t.to.authority == 255


class TestUpdate:
    def test_empty_update(self) -> None:
        u = Update[str](transfers=[])
        assert len(u.transfers) == 0

    def test_with_transfers(self) -> None:
        state = State(
            subject=Subject(key="k", name="n"),
            resource="ch-1",
            authority=1,
        )
        t = Transfer[str](to=state)
        u = Update[str](transfers=[t])
        assert len(u.transfers) == 1
