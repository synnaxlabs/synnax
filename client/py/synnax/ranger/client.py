#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
import functools
from typing import Callable, overload

from freighter import UnaryClient

from synnax.channel.retrieve import ChannelRetriever
from synnax.exceptions import QueryError
from synnax.framer.client import Client
from synnax.ranger.active import Active
from synnax.ranger.alias import Aliaser
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
from synnax.ranger.writer import RangeWriter
from synnax.signals.signals import Registry
from synnax.state import LatestState
from synnax.telem import TimeRange

RANGE_SET_CHANNEL = "sy_range_set"


class RangeClient:
    __frame_client: Client
    __channel_retriever: ChannelRetriever
    __retriever: RangeRetriever
    __writer: RangeWriter
    __unary_client: UnaryClient
    __active: Active
    __signals: Registry

    def __init__(
        self,
        unary_client: UnaryClient,
        frame_client: Client,
        writer: RangeWriter,
        retriever: RangeRetriever,
        channel_retriever: ChannelRetriever,
        signals: Registry,
    ) -> None:
        self.__unary_client = unary_client
        self.__frame_client = frame_client
        self.__writer = writer
        self.__retriever = retriever
        self.__channel_retriever = channel_retriever
        self.__active = Active(self.__unary_client)
        self.__signals = signals

    @overload
    def create(
        self,
        *,
        name: str,
        time_range: TimeRange,
        retrieve_if_name_exists: bool = False,
    ) -> Range:
        """Creates a named range spanning a region of time. This range is persisted
        to the cluster and is visible to all clients.

        :param name: The name of the range
        :param time_range: The time range of the range. This time range must be valid
        i.e. its start time must be less than or equal to its end time.
        :param retrieve_if_name_exists: If true, this method will retrieve the range
        if it already exists. This method will throw a ValidationError
        if the range already exists and has a DIFFERENT time range.
        """
        ...

    @overload
    def create(
        self,
        ranges: Range,
        retrieve_if_name_exists: bool = False,
    ) -> Range:
        """Creates the given range. This range is persisted to the cluster and is
        visible to all clients.

        :param ranges: The range to create
        :param retrieve_if_name_exists: If true, this method will retrieve the range
        if it already exists. This method will throw a ValidationError if the range
        already exists and has a DIFFERENT time range.
        """
        ...

    @overload
    def create(
        self,
        ranges: list[Range],
        retrieve_if_name_exists: bool = False,
    ) -> list[Range]:
        """Creates the given ranges. These ranges are persisted to the cluster and are
        visible to all clients.

        :param ranges: The ranges to create
        :param retrieve_if_name_exists: If true, this method will retrieve the range
        if it already exists. This method will throw a ValidationError if the range
        already exists and has a DIFFERENT time range.
        """
        ...

    def create(
        self,
        ranges: Range | list[Range] = None,
        *,
        name: str = "",
        time_range: TimeRange | None = None,
        retrieve_if_name_exists: bool = False,
    ) -> Range | list[Range]:
        is_single = True
        if ranges is None:
            to_create = [RangePayload(name=name, time_range=time_range)]
        elif isinstance(ranges, Range):
            to_create = [ranges.to_payload()]
        else:
            is_single = False
            to_create = [r.to_payload() for r in ranges]

        res: list[Range] = list()
        if retrieve_if_name_exists:
            res = self.retrieve([r.name for r in to_create])
            if is_single and len(res) > 1:
                filtered = [r for r in res if r.time_range == time_range]
                if len(filtered) == 0:
                    raise QueryError(
                        f"""
                        retrieve_if_name_exists was set to true, but {len(res)} ranges
                        were found matching {name} but none had the same time range as
                        passed to create. Synnax can't figure out which one you want!
                        """
                    )
                if len(filtered) > 1:
                    raise QueryError(
                        f"""
                    retrieve_if_name_exists was set to true, but {len(res)} ranges were
                    found that matched {name} and had time range {time_range}. Synnax
                    can't figure out which one you want!
                    """
                    )
                res = [filtered[0]]
            existing_names = {r.name for r in res}
            to_create = [r for r in to_create if r.name not in existing_names]

        if len(to_create) > 0:
            res.extend(self.__sugar(self.__writer.create(to_create)))

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

    def delete(
        self,
        params: RangeParams,
    ) -> None:
        _ranges = self.__retriever.retrieve(params)
        self.__writer.delete([r.key for r in _ranges])

    def set_active(self, key: RangeKey):
        self.__active.set(key)

    def clear_active(self):
        self.__active.clear()

    def retrieve_active(self) -> Range | None:
        rng = self.__active.retrieve()
        return None if rng is None else self.__sugar([rng])[0]

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

    def on_create(self, f: Callable[[Range], None]):
        @functools.wraps(f)
        def wrapper(state: LatestState):
            d = state[RANGE_SET_CHANNEL]
            f(
                Range(
                    name=d["name"],
                    key=d["key"],
                    time_range=TimeRange(
                        start=d["time_range"]["start"],
                        end=d["time_range"]["end"],
                    ),
                    _frame_client=self.__frame_client,
                    _channel_retriever=self.__channel_retriever,
                    _kv=KV(d["key"], self.__unary_client),
                    _aliaser=Aliaser(d["key"], self.__unary_client),
                )
            )

        return self.__signals.on([RANGE_SET_CHANNEL], lambda s: True)(wrapper)
