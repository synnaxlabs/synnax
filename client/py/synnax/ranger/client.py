#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from synnax.framer.client import FrameClient
from synnax.ranger.retrieve import RangeRetriever
from synnax.ranger.create import RangeCreator
from synnax.ranger.range import Range
from synnax.telem import TimeRange
from synnax.ranger.payload import (
    RangePayload,
    RangeKey,
    RangeName,
    RangeKeys,
    RangeNames,
    RangeParams,
    normalize_range_params,
)
from synnax.exceptions import QueryError


class RangeClient:
    __frame_client: FrameClient
    __retriever: RangeRetriever
    __creator: RangeCreator

    def __init__(
        self,
        frame_client: FrameClient,
        creator: RangeCreator,
        retriever: RangeRetriever,
    ) -> None:
        self.__frame_client = frame_client
        self.__creator = creator
        self.__retriever = retriever

    @overload
    def create(
        self,
        *,
        name: str,
        time_range: TimeRange,
    ) -> Range:
        ...

    @overload
    def create(
        self,
        ranges: Range,
    ) -> Range:
        ...

    @overload
    def create(
        self,
        ranges: list[Range],
    ) -> list[Range]:
        ...

    def create(
        self,
        ranges: Range | list[Range],
        *,
        name: str = "",
        time_range: TimeRange | None = None,
    ) -> Range | list[Range]:
        if ranges is None:
            _ranges = [RangePayload(name=name, time_range=time_range)]
        elif isinstance(ranges, Range):
            _ranges = [ranges.to_payload()]
        else:
            _ranges = [r.to_payload() for r in ranges]
        _ranges = self.__creator.create(_ranges)
        return self.__sugar(_ranges)

    @overload
    def retrieve(
        self,
        key_or_name: RangeKey | RangeName,
    ) -> Range:
        ...

    @overload
    def retrieve(
        self,
        keys_or_names: RangeKeys | RangeNames,
    ) -> list[Range]:
        ...

    def retrieve(
        self,
        params: RangeParams,
    ) -> Range | list[Range]:
        normal = normalize_range_params(params)
        _ranges = self.__retriever.retrieve(normal)
        sug = self.__sugar(_ranges)
        if not normal.single:
            return sug
        if len(sug) == 0:
            raise QueryError(f"Range {normal} not found")
        elif len(sug) > 1:
            raise QueryError(f"Multiple ranges matching {normal} found")
        return sug[0]

    def __sugar(self, ranges: list[RangePayload]):
        return [Range(**r.dict(), frame_client=self.__frame_client) for r in ranges]
