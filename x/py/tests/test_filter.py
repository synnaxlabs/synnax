#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel, ConfigDict, Field

from x import filter


class Where(BaseModel):
    model_config = ConfigDict(populate_by_name=True, extra="forbid")

    and_: list["Where"] | None = Field(default=None, alias="and")
    key: filter.Set[str] | None = None


class TestMerge:
    def test_no_where(self) -> None:
        flat = Where(key=filter.Set[str](in_=["a"]))
        assert filter.merge(None, flat) == flat

    def test_empty_flat(self) -> None:
        where = Where(key=filter.Set[str](eq="a"))
        assert filter.merge(where, Where()) == where

    def test_both(self) -> None:
        where = Where(key=filter.Set[str](eq="a"))
        flat = Where(key=filter.Set[str](in_=["b"]))
        assert filter.merge(where, flat) == Where(and_=[where, flat])
