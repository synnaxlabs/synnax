#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from freighter import UnaryClient

from synnax.channel.retrieve import ChannelRetriever
from synnax.exceptions import QueryError
from synnax.framer.client import Client
from synnax.ranger.alias import Aliaser
from synnax.ranger.create import RangeCreator
from synnax.ranger.kv import KV
from synnax.ranger.payload import (
    RangeKey,
    RangeKeys,
    RangeName,
    RangeNames,
    RangeParams,
    RangePayload,
    normalize_range_params,
)
from synnax.ranger.range import Range
from synnax.ranger.retrieve import RangeRetriever
from synnax.telem import TimeRange


class RangeClient:
    __frame_client: Client
    __channel_retriever: ChannelRetriever
    __retriever: RangeRetriever
    __creator: RangeCreator
    __unary_client: UnaryClient

    def __init__(
        self,
        unary_client: UnaryClient,
        frame_client: Client,
        creator: RangeCreator,
        retriever: RangeRetriever,
        channel_retriever: ChannelRetriever,
    ) -> None:
        self.__unary_client = unary_client
        self.__frame_client = frame_client
        self.__creator = creator
        self.__retriever = retriever
        self.__channel_retriever = channel_retriever

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
        ranges: Range | list[Range] = None,
        *,
        name: str = "",
        time_range: TimeRange | None = None,
    ) -> Range | list[Range]:
        is_single = True
        if ranges is None:
            _ranges = [RangePayload(name=name, time_range=time_range)]
        elif isinstance(ranges, Range):
            _ranges = [ranges.to_payload()]
        else:
            is_single = False
            _ranges = [r.to_payload() for r in ranges]
        res = self.__sugar(self.__creator.create(_ranges))
        return res if not is_single else res[0]

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
        _ranges = self.__retriever.retrieve(normal.params)
        sug = self.__sugar(_ranges)
        if not normal.single:
            return sug
        if len(sug) == 0:
            raise QueryError(f"Range {normal} not found")
        elif len(sug) > 1:
            raise QueryError(f"Multiple ranges matching {normal} found")
        return sug[0]

    def search(
        self,
        term: str,
    ) -> list[Range]:
        _ranges = self.__retriever.search(term)
        return self.__sugar(_ranges)

    def __sugar(self, ranges: list[RangePayload]):
        return [
            Range(
                **r.dict(),
                _frame_client=self.__frame_client,
                _channel_retriever=self.__channel_retriever,
                _kv=KV(r.key, self.__unary_client),
                _aliaser=Aliaser(r.key, self.__unary_client),
            )
            for r in ranges
        ]
