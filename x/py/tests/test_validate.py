#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from typing import Annotated
from uuid import UUID, uuid4

import pytest
from pydantic import AfterValidator, BaseModel, ValidationError

from x import telem, validate


class Item(BaseModel):
    channel: int


class Holder(BaseModel):
    node: Annotated[str, AfterValidator(validate.max_bytes(4))] = ""
    items: Annotated[list[Item], AfterValidator(validate.unique("channel"))] = []


class UUIDHolder(BaseModel):
    owner: Annotated[UUID, AfterValidator(validate.non_nil_uuid)]


class TestMaxBytes:
    @pytest.mark.parametrize("value", ["abcd", "éé", "😀"])
    def test_accepts_four_bytes(self, value: str) -> None:
        """Should accept a string of 4 UTF-8 bytes."""
        assert Holder(node=value).node == value

    @pytest.mark.parametrize("value", ["abcde", "ééé"])
    def test_rejects_more_bytes(self, value: str) -> None:
        """Should refuse a string of more than 4 UTF-8 bytes."""
        with pytest.raises(ValidationError, match="must be at most 4 bytes long"):
            Holder(node=value)


class TestUnique:
    def test_accepts_distinct_keys(self) -> None:
        """Should accept items whose keys differ."""
        items = [Item(channel=1), Item(channel=2)]
        assert Holder(items=items).items == items

    def test_rejects_shared_keys(self) -> None:
        """Should name each item whose key an earlier item holds."""
        items = [Item(channel=1), Item(channel=2), Item(channel=1), Item(channel=2)]
        with pytest.raises(
            ValidationError, match=re.escape("2.channel, 3.channel: must be unique")
        ):
            Holder(items=items)


class TestMultipleOf:
    def test_accepts_a_multiple(self) -> None:
        """Should accept a TimeSpan that is a whole number of milliseconds."""
        span = telem.TimeSpan.SECOND
        assert validate.multiple_of(1_000_000)(span) == span

    def test_rejects_a_remainder(self) -> None:
        """Should refuse a TimeSpan with a remainder below the step."""
        with pytest.raises(ValueError, match="must be a multiple of 1000000"):
            validate.multiple_of(1_000_000)(telem.TimeSpan(1_500))


class TestNonNilUUID:
    def test_accepts_a_uuid(self) -> None:
        """Should accept a UUID that is not nil."""
        key = uuid4()
        assert UUIDHolder(owner=key).owner == key

    def test_rejects_the_nil_uuid(self) -> None:
        """Should refuse the nil UUID."""
        with pytest.raises(ValidationError, match="must be non-zero"):
            UUIDHolder(owner=UUID(int=0))
