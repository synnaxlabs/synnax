#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from __future__ import annotations

import functools
from typing import Callable, overload

from freighter import UnaryClient

from synnax.channel.retrieve import ChannelRetriever
from synnax.exceptions import QueryError
from synnax.framer.client import Client
from synnax.framer.frame import CrudeFrame
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
from synnax.ranger.retrieve import RangeRetriever
from synnax.ranger.writer import RangeWriter
from synnax.signals.signals import Registry
from synnax.state import LatestState
from synnax.telem import (
    TimeRange,
    Series,
    SampleValue,
    DataType,
    Rate,
    CrudeSeries,
)
from synnax.ontology.id import OntologyID
from synnax.channel.payload import (
    ChannelKey,
    ChannelName,
    ChannelPayload,
    ChannelKeys,
    ChannelNames,
    ChannelParams,
)
from synnax.util.interop import overload_comparison_operators

from uuid import UUID
from typing import overload

import numpy as np
from pydantic import PrivateAttr

RANGE_SET_CHANNEL = "sy_range_set"


class _InternalScopedChannel(ChannelPayload):
    __range: Range | None = PrivateAttr(None)
    """The range that this channel belongs to."""
    __frame_client: Client | None = PrivateAttr(None)
    """The frame client for executing read operations."""
    __aliaser: Aliaser | None = PrivateAttr(None)
    """An aliaser for setting the channel's alias."""
    __cache: Series | None = PrivateAttr(None)
    """An internal cache to prevent repeated reads from the same channel."""

    def __new__(cls, *args, **kwargs):
        cls = overload_comparison_operators(cls, "__array__")
        return super().__new__(cls)

    class Config:
        arbitrary_types_allowed = True

    def __init__(
        self,
        rng: Range,
        frame_client: Client,
        payload: ChannelPayload,
        aliaser: Aliaser | None = None,
    ):
        super().__init__(**payload.dict())
        self.__range = rng
        self.__frame_client = frame_client
        self.__aliaser = aliaser

    @property
    def time_range(self) -> TimeRange:
        return self.__range.time_range

    def __array__(self, *args, **kwargs) -> np.ndarray:
        """Converts the channel to a numpy array. This method is necessary
        for numpy interop."""
        return self.read().__array__(*args, **kwargs)

    def __getitem__(self, index: int) -> SampleValue:
        return self.read().__getitem__(index)

    def to_numpy(self) -> np.ndarray:
        """Converts the channel to a numpy array. This method is necessary
        for matplotlib interop.
        """
        return self.read().to_numpy()

    def read(self) -> Series:
        if self.__cache is None:
            self.__cache = self.__frame_client.read(self.time_range, self.key)
        return self.__cache

    def set_alias(self, alias: str):
        self.__range.set_alias(self.key, alias)

    def __str__(self) -> str:
        return f"{super().__str__()} between {self.time_range.start} and {self.time_range.end}"

    def __len__(self):
        return len(self.read())


class ScopedChannel:
    """A channel that is scoped to a particular range. This class is returned when
    accessing the channel as a key or property on a range. This channel has direct
    interoperability with numpy arrays, meaning that it can be passed as an argument
    to any function/method that accepts a numpy array.

    It's very important to note that if the property accessor matches multiple channels,
    this class will contain all of them, and, as a result, single channel operations
    will fail. However, you can use the __iter__ method to iterate over the channels
    in the returned values. This is particularly relevant when using regex to match
    multiple channels.
    """

    __internal: list[_InternalScopedChannel]
    __query: str

    def __new__(cls, *args, **kwargs):
        cls = overload_comparison_operators(cls, "__array__")
        return super().__new__(cls)

    def __init__(
        self,
        query: str,
        internal: list[_InternalScopedChannel],
    ):
        self.__internal = internal
        self.__query = query

    def __guard(self):
        if len(self.__internal) > 1:
            raise QueryError(
                f"""Multiple channels found for query '{self.__query}':
            {[str(ch) for ch in self.__internal]}
            """
            )

    def __array__(self, *args, **kwargs):
        """Converts the scoped channel to a numpy array. This method is necessary
        for numpy interop."""
        self.__guard()
        return self.__internal[0].__array__(*args, **kwargs)

    def __getitem__(self, index: int) -> SampleValue:
        self.__guard()
        return self.__internal[0].__getitem__(index)

    def to_numpy(self) -> np.ndarray:
        """Converts the scoped channel to a numpy array. This method is necessary
        for matplotlib interop."""
        return self.__array__()

    @property
    def key(self) -> ChannelKey:
        self.__guard()
        return self.__internal[0].key

    @property
    def name(self) -> str:
        self.__guard()
        return self.__internal[0].name

    @property
    def data_type(self) -> DataType:
        self.__guard()
        return self.__internal[0].data_type

    @property
    def is_index(self) -> bool:
        self.__guard()
        return self.__internal[0].is_index

    @property
    def index(self) -> ChannelKey:
        self.__guard()
        return self.__internal[0].index

    @property
    def leaseholder(self) -> int:
        self.__guard()
        return self.__internal[0].leaseholder

    @property
    def rate(self) -> Rate:
        self.__guard()
        return self.__internal[0].rate

    def set_alias(self, alias: str):
        self.__guard()
        self.__internal[0].set_alias(alias)

    def __iter__(self):
        return iter(self.__internal)

    def __len__(self):
        return sum(len(ch) for ch in self.__internal)


_RANGE_NOT_CREATED = QueryError(
    """Cannot read from a range that has not been created.
Please call client.ranges.create(range) before attempting to read from a range."""
)


class Range(RangePayload):
    """A range is a user-defined region of a cluster's data. It's identified by a name,
    time range, and uniquely generated key. See
    https://docs.synnaxlabs.com/concepts/read-ranges for an introduction to ranges and
    how they work.
    """

    __frame_client: Client | None = PrivateAttr(None)
    """The frame client for executing read and write operations."""
    __channel_retriever: ChannelRetriever | None = PrivateAttr(None)
    """For retrieving channels from the cluster."""
    __kv: KV | None = PrivateAttr(None)
    """Key-value store for storing metadata about the range."""
    __aliaser: Aliaser | None = PrivateAttr(None)
    """For setting and resolving aliases."""
    __cache: dict[ChannelKey, _InternalScopedChannel] = PrivateAttr(dict())
    """A writer for creating child ranges"""
    __client: RangeClient | None = PrivateAttr(None)

    class Config:
        arbitrary_types_allowed = True

    def __init__(
        self,
        name: str,
        time_range: TimeRange,
        key: UUID = UUID(int=0),
        color: str = "",
        *,
        _frame_client: Client | None = None,
        _channel_retriever: ChannelRetriever | None = None,
        _kv: KV | None = None,
        _aliaser: Aliaser | None = None,
        _client: RangeClient | None = None,
    ):
        """Initializes a new Range using the given parameters. It's important to note
        that this does not create the Range in the cluster. To create the range, call
        client.ranges.create(range).

        :param name: A human-readable name for the range. This should represent the data
            that the range contains i.e. "Hotfire 1", "Print 22", or "Tank Burst Test.".
        :param time_range: The time region spanned by the range. Note that this range
            is end inclusive and end exclusive i.e. the start represents the timestamp
            of just before or at the first data point in the range, and the end
            represents the timestamp of the just after the last data point in the range.
        :param key: A UUID that uniquely identifies the range. This is typically not
            set by the user, and is generated by the cluster upon creating the range.
        :param _frame_client: The backing client for reading and writing data to
            and from the cluster. This is provided by Synnax during calls to
            .ranges.create() and .ranges.retrieve(), and should not be set by the user.
        :param _channel_retriever: The backing client for retrieving channels to
            and from the cluster. This is provided by Synnax during calls to
            .ranges.create() and .ranges.retrieve(), and should not be set by the user.
        """
        super().__init__(name=name, time_range=time_range, key=key, color=color)
        self.__frame_client = _frame_client
        self.__channel_retriever = _channel_retriever
        self.__kv = _kv
        self.__aliaser = _aliaser
        self.__client = _client

    def __getattr__(self, query: str) -> ScopedChannel:
        channels = self._channel_retriever.retrieve(query)
        aliases = self._aliaser.resolve([query])
        channels.extend(self._channel_retriever.retrieve(list(aliases.values())))
        if len(channels) == 0:
            raise QueryError(f"Channel matching {query} not found")

        return ScopedChannel(query, self.__splice_cached(channels))

    def __getitem__(self, name: str | ChannelKey) -> ScopedChannel:
        return self.__getattr__(name)

    def __splice_cached(
        self, channels: list[ChannelPayload]
    ) -> list[_InternalScopedChannel]:
        results = list()
        for pld in channels:
            cached = self.__cache.get(pld.key, None)
            if cached is None:
                cached = _InternalScopedChannel(
                    rng=self,
                    frame_client=self._frame_client,
                    payload=pld,
                )
                self.__cache[pld.key] = cached
            results.append(cached)
        return results

    @property
    def meta_data(self):
        if self.__kv is None:
            raise _RANGE_NOT_CREATED
        return self.__kv

    @property
    def _aliaser(self):
        if self.__aliaser is None:
            raise _RANGE_NOT_CREATED
        return self.__aliaser

    @property
    def _frame_client(self) -> Client:
        if self.__frame_client is None:
            raise _RANGE_NOT_CREATED
        return self.__frame_client

    @property
    def _channel_retriever(self) -> ChannelRetriever:
        if self.__channel_retriever is None:
            raise _RANGE_NOT_CREATED
        return self.__channel_retriever

    def set_alias(self, channel: ChannelKey | ChannelName, alias: str):
        ...

    def set_alias(self, channel: dict[ChannelKey | ChannelName, str]):
        ...

    def set_alias(
        self,
        channel: ChannelKey | ChannelName | dict[ChannelKey | ChannelName, str],
        alias: str = None,
    ):
        if not isinstance(channel, dict):
            if alias is None:
                raise ValueError("Alias must be provided if channel is not a dict")
            channel = {channel: alias}

        corrected = {}
        for ch, alias in channel.items():
            if isinstance(ch, ChannelName):
                res = self._channel_retriever.retrieve(ch)
                if len(res) == 0:
                    raise QueryError(f"Channel {ch} not found")
                corrected[res[0].key] = alias
            else:
                corrected[ch] = alias
        self._aliaser.set(corrected)

    def to_payload(self) -> RangePayload:
        return RangePayload(name=self.name, time_range=self.time_range, key=self.key)

    @overload
    def write(self, to: ChannelKey | ChannelName | ChannelPayload, data: CrudeSeries):
        ...

    @overload
    def write(
        self,
        to: ChannelKeys | ChannelNames | list[ChannelPayload],
        series: list[CrudeSeries],
    ):
        ...

    @overload
    def write(self, frame: CrudeFrame):
        ...

    def write(
        self,
        to: ChannelParams | ChannelPayload | list[ChannelPayload] | CrudeFrame,
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ):
        self.__frame_client.write(self.time_range.start, to, series)

    def create_sub_range(
        self,
        *,
        name: str,
        time_range: TimeRange,
        color: str = "",
        key: RangeKey = UUID(int=0),
    ) -> Range:
        return self.__client.create(
            name=name,
            time_range=time_range,
            color=color,
            key=key,
            parent=OntologyID(type="range", key=str(self.key)),
        )


class RangeClient:
    __frame_client: Client
    __channel_retriever: ChannelRetriever
    __retriever: RangeRetriever
    __writer: RangeWriter
    __unary_client: UnaryClient
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
        self.__signals = signals

    @overload
    def create(
        self,
        *,
        name: str,
        time_range: TimeRange,
        color: str = "",
        retrieve_if_name_exists: bool = False,
        parent: OntologyID | None = None,
        key: RangeKey = RangeKey(int=0),
    ) -> Range:
        """Creates a named range spanning a region of time. This range is persisted
        to the cluster and is visible to all clients.

        :param name: The name of the range
        :param time_range: The time range of the range. This time range must be valid
        i.e. its start time must be less than or equal to its end time.
        :param retrieve_if_name_exists: If true, this method will retrieve the range
        if it already exists. This method will throw a ValidationError
        if the range already exists and has a DIFFERENT time range.
        :param parent: An optional parent ontology item to set as the parent of the
        range.
        """
        ...

    @overload
    def create(
        self,
        ranges: Range,
        retrieve_if_name_exists: bool = False,
        parent: OntologyID | None = None,
    ) -> Range:
        """Creates the given range. This range is persisted to the cluster and is
        visible to all clients.

        :param ranges: The range to create
        :param retrieve_if_name_exists: If true, this method will retrieve the range
        if it already exists. This method will throw a ValidationError if the range
        already exists and has a DIFFERENT time range.
        :param parent: An optional parent ontology item to set as the parent of the
        range.
        """
        ...

    @overload
    def create(
        self,
        ranges: list[Range],
        retrieve_if_name_exists: bool = False,
        parent: OntologyID | None = None,
    ) -> list[Range]:
        """Creates the given ranges. These ranges are persisted to the cluster and are
        visible to all clients.

        :param ranges: The ranges to create
        :param retrieve_if_name_exists: If true, this method will retrieve the range
        if it already exists. This method will throw a ValidationError if the range
        already exists and has a DIFFERENT time range.
        :param parent: An optional parent ontology item to set as the parent of the
        """
        ...

    def create(
        self,
        ranges: Range | list[Range] = None,
        *,
        key: RangeKey = RangeKey(int=0),
        name: str = "",
        time_range: TimeRange | None = None,
        color: str = "",
        retrieve_if_name_exists: bool = False,
        parent: OntologyID | None = None,
    ) -> Range | list[Range]:
        is_single = True
        if ranges is None:
            to_create = [
                RangePayload(key=key, name=name, time_range=time_range, color=color)
            ]
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
            res.extend(self.__sugar(self.__writer.create(to_create, parent=parent)))
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
                _client=self,
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
