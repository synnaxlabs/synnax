#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Checks for pydantic AfterValidator that pydantic's own constraints lack."""

from collections.abc import Callable, Sequence
from typing import Any
from uuid import UUID


def max_bytes(limit: int) -> Callable[[str], str]:
    """Returns a check that refuses a string whose UTF-8 encoding holds more than
    limit bytes."""

    def check(value: str) -> str:
        if len(value.encode("utf-8")) > limit:
            raise ValueError(f"must be at most {limit} bytes long")
        return value

    return check


def unique[T: Sequence[Any]](key: str) -> Callable[[T], T]:
    """Returns a check that refuses a list in which two items share the attribute
    key. The error names each item whose key an earlier item holds."""

    def check(items: T) -> T:
        seen: set[Any] = set()
        duplicates: list[str] = []
        for i, item in enumerate(items):
            value = getattr(item, key)
            if value in seen:
                duplicates.append(f"{i}.{key}")
            seen.add(value)
        if duplicates:
            raise ValueError(f"{', '.join(duplicates)}: must be unique")
        return items

    return check


def multiple_of[T: int](step: int) -> Callable[[T], T]:
    """Returns a check that refuses an integer, such as a telem.TimeSpan, that is not a
    whole multiple of step."""

    def check(value: T) -> T:
        if int(value) % step != 0:
            raise ValueError(f"must be a multiple of {step}")
        return value

    return check


def non_nil_uuid(value: UUID) -> UUID:
    """Refuses the nil UUID, which a required UUID field cannot hold."""
    if value.int == 0:
        raise ValueError("must be non-zero")
    return value
