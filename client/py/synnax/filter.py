#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Shared comparison structs for the typed query protocol."""

from __future__ import annotations

from pydantic import BaseModel, Field


class StringFilter(BaseModel):
    eq: str | None = None
    in_: list[str] | None = Field(default=None, alias="in")
    contains: str | None = None
    regex: str | None = None

    model_config = {"populate_by_name": True}


class BoolFilter(BaseModel):
    eq: bool | None = None


class NumericFilter(BaseModel):
    eq: int | float | None = None
    in_: list[int | float] | None = Field(default=None, alias="in")
    gt: int | float | None = None
    lt: int | float | None = None
    gte: int | float | None = None
    lte: int | float | None = None

    model_config = {"populate_by_name": True}


class TimeFilter(BaseModel):
    eq: int | None = None
    gt: int | None = None
    lt: int | None = None
    gte: int | None = None
    lte: int | None = None


class StringOrderBy(BaseModel):
    direction: str = "asc"
    after: str | None = None


class NumericOrderBy(BaseModel):
    direction: str = "asc"
    after: int | float | None = None


class TimestampOrderBy(BaseModel):
    direction: str = "asc"
    after: int | None = None
