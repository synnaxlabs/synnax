#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from __future__ import annotations

import functools
from collections.abc import Callable, Iterator
from typing import Literal, cast, overload
from uuid import UUID

import numpy as np
from pydantic import BaseModel, PrivateAttr

import synnax.channel.payload as channel
from freighter import UnaryClient
from synnax import framer, label, util
from synnax.channel.retrieve import Retriever as ChannelRetriever
from synnax.exceptions import ConfigurationError, MultipleFoundError, NotFoundError
from synnax.framer.frame import K
from synnax.ontology import Client as OntologyClient
from synnax.ontology.payload import ID
from synnax.ranger import alias as alias_
from synnax.ranger import builder_gen, kv
from synnax.ranger.builder_gen import Condition
from synnax.ranger.retrieve import Retriever
from synnax.ranger.types_gen import (
    Key,
    Payload,
    RetrieveRequest,
    Where,
    ontology_id,
)
from synnax.ranger.writer import Writer
from synnax.signals.signals import Registry
from synnax.state import Latest
from synnax.task import Client as TaskClient
from synnax.task import Task
from synnax.telem import (
    CrudeSeries,
    DataType,
    MultiSeries,
    SampleValue,
    TimeRange,
)
from x import filter
from x.color import Color
from x.interop import ArrayOperators
from x.lists import all_none, normalize, override
from x.uuid import NIL

_RANGE_SET_CHANNEL = "sy_range_set"

# Pydantic declares BaseModel.__getattr__ only at runtime.
_model_getattr: Callable[[BaseModel, str], object] = vars(BaseModel)["__getattr__"]


class _InternalScopedChannel(ArrayOperators, channel.Payload):
    _cached_range: Range | None = PrivateAttr(None)
    """The range that this channel belongs to."""
    _cached_frame_client: framer.Client | None = PrivateAttr(None)
    """The frame client for executing read operations."""
    _aliaser: alias_.Client | None = PrivateAttr(None)
    """An aliaser for setting the channel's alias."""
    _cache: MultiSeries | None = PrivateAttr(None)
    """An internal cache to prevent repeated reads from the same channel."""
    _tasks: TaskClient | None = PrivateAttr(None)
    _ontology: OntologyClient | None = PrivateAttr(None)

    # ArrayOperators clears __hash__, but writes can key data by channel payload.
    __hash__ = channel.Payload.__hash__

    def __init__(
        self,
        rng: Range,
        frame_client: framer.Client,
        tasks: TaskClient,
        ontology: OntologyClient,
        payload: channel.Payload,
        _aliaser: alias_.Client | None = None,
    ):
        super().__init__(**payload.model_dump())
        self._cached_range = rng
        self._cached_frame_client = frame_client
        self._aliaser = _aliaser
        self._tasks = tasks
        self._ontology = ontology

    @property
    def _range(self) -> Range:
        if self._cached_range is None:
            raise _RANGE_NOT_CREATED
        return self._cached_range

    @property
    def time_range(self) -> TimeRange:
        return self._range.time_range

    def __array__(
        self, dtype: np.dtype | None = None, copy: bool | None = None
    ) -> np.ndarray:
        """Converts the channel to a numpy array. This method is necessary
        for numpy interop."""
        return self.read().__array__(dtype, copy)

    def __getitem__(self, index: int) -> SampleValue:
        return self.read().__getitem__(index)

    def to_numpy(self) -> np.ndarray:
        """Converts the channel to a numpy array. This method is necessary
        for matplotlib interop.
        """
        return self.read().to_numpy()

    @property
    def _frame_client(self) -> framer.Client:
        if self._cached_frame_client is None:
            raise _RANGE_NOT_CREATED
        return self._cached_frame_client

    def read(self) -> MultiSeries:
        if self._cache is None:
            self._cache = self._frame_client.read(self.time_range, self.key)
        return self._cache

    def set_alias(self, alias: str) -> None:
        self._range.set_alias(self.key, alias)

    def __str__(self) -> str:
        tr = self.time_range
        return f"{super().__str__()} between {tr.start} and {tr.end}"

    def __len__(self) -> int:
        return len(self.read())


class ScopedChannel(ArrayOperators):
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

    _internal: list[_InternalScopedChannel]
    _query: str

    def __init__(
        self,
        *,
        query: str,
        internal: list[_InternalScopedChannel],
    ):
        self._internal = internal
        self._query = query

    def _guard(self) -> None:
        if len(self._internal) > 1:
            raise MultipleFoundError(
                f"""Multiple channels found for query '{self._query}':
            {[str(ch) for ch in self._internal]}
            """
            )

    def __array__(
        self, dtype: np.dtype | None = None, copy: bool | None = None
    ) -> np.ndarray:
        """Converts the scoped channel to a numpy array. This method is necessary
        for numpy interop."""
        self._guard()
        return self._internal[0].__array__(dtype, copy)

    def __getitem__(self, index: int) -> SampleValue:
        self._guard()
        return self._internal[0].__getitem__(index)

    def to_numpy(self) -> np.ndarray:
        """Converts the scoped channel to a numpy array. This method is necessary
        for matplotlib interop."""
        return self.__array__()

    @property
    def key(self) -> channel.Key:
        self._guard()
        return self._internal[0].key

    @property
    def name(self) -> str:
        self._guard()
        return self._internal[0].name

    @property
    def data_type(self) -> DataType:
        self._guard()
        return self._internal[0].data_type

    @property
    def is_index(self) -> bool:
        self._guard()
        return self._internal[0].is_index

    @property
    def index(self) -> channel.Key:
        self._guard()
        return self._internal[0].index

    @property
    def leaseholder(self) -> int:
        self._guard()
        return self._internal[0].leaseholder

    def set_alias(self, alias: str) -> None:
        self._guard()
        self._internal[0].set_alias(alias)

    def __iter__(self) -> Iterator[_InternalScopedChannel]:
        return iter(self._internal)

    def __len__(self) -> int:
        return sum(len(ch) for ch in self._internal)


_RANGE_NOT_CREATED = ConfigurationError(
    """Cannot read from a range that has not been created.
Please call client.ranges.create(range) before attempting to read from a range."""
)


class Range(Payload):
    """A range is a user-defined region of a cluster's data. It's identified by a name,
    time range, and uniquely generated key. See
    https://docs.synnaxlabs.com/reference/ranges for an introduction to ranges and how
    they work.
    """

    _cached_frame_client: framer.Client | None = PrivateAttr(None)
    """The frame client for executing read and write operations."""
    _channels: ChannelRetriever | None = PrivateAttr(None)
    """For retrieving channels from the cluster."""
    _kv: kv.Client | None = PrivateAttr(None)
    """Key-value store for storing metadata about the range."""
    _cached_aliaser: alias_.Client | None = PrivateAttr(None)
    """For setting and resolving aliases."""
    _cache: dict[channel.Key, _InternalScopedChannel] = PrivateAttr({})
    """A writer for creating child ranges"""
    _scoped: dict[channel.Key | str, ScopedChannel] = PrivateAttr({})
    """Resolved channel queries, so each name or key is retrieved once."""
    _cached_client: Client | None = PrivateAttr(None)
    _cached_tasks: TaskClient | None = PrivateAttr(None)
    _cached_ontology: OntologyClient | None = PrivateAttr(None)

    def __init__(
        self,
        *,
        name: str,
        time_range: TimeRange,
        key: UUID = NIL,
        color: str | None = "",
        parent: Range | Payload | None = None,
        _frame_client: framer.Client | None = None,
        _channel_retriever: ChannelRetriever | None = None,
        _kv: kv.Client | None = None,
        _aliaser: alias_.Client | None = None,
        _client: Client | None = None,
        _tasks: TaskClient | None = None,
        _ontology: OntologyClient | None = None,
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
        :param parent: An optional parent range. Retrieve or create the parent first,
            then pass the ``Range`` directly.
        :param _frame_client: The backing client for reading and writing data to
            and from the cluster. This is provided by Synnax during calls to
            .ranges.create() and .ranges.retrieve(), and should not be set by the user.
        :param _channel_retriever: The backing client for retrieving channels to
            and from the cluster. This is provided by Synnax during calls to
            .ranges.create() and .ranges.retrieve(), and should not be set by the user.
        """
        parent_payload: Payload | None = None
        if parent is not None:
            # parent may be a full Range or an already-resolved Payload (e.g. the
            # .parent of a created or retrieved range, which the server returns as a
            # Payload). Only Range has to_payload(); a Payload is used as-is.
            parent_payload = (
                parent.to_payload() if isinstance(parent, Range) else parent
            )
            if parent_payload.parent is not None:
                parent_payload = parent_payload.model_copy(update={"parent": None})
        super().__init__(
            name=name,
            time_range=time_range,
            key=key,
            color=None if color is None else Color(color),
            parent=parent_payload,
        )
        self._cached_frame_client = _frame_client
        self._channels = _channel_retriever
        self._kv = _kv
        self._cached_aliaser = _aliaser
        self._cached_client = _client
        self._cached_tasks = _tasks
        self._cached_ontology = _ontology

    def _get_scoped_channel(
        self, channels: list[channel.Payload], query: str
    ) -> ScopedChannel:
        if len(channels) == 0:
            raise NotFoundError(f"Channel matching {query} not found")
        return ScopedChannel(query=query, internal=self._splice_cached(channels))

    def __getattr__(self, query: str) -> ScopedChannel:
        try:
            # Private attributes resolve here. Type checkers use their declarations.
            return cast(ScopedChannel, _model_getattr(self, query))
        except AttributeError:
            pass
        scoped = self._scoped.get(query)
        if scoped is not None:
            return scoped
        channels = self._channel_retriever.retrieve(query)
        aliases = self._aliaser.resolve([query])
        channels.extend(self._channel_retriever.retrieve(list(aliases.values())))
        scoped = self._get_scoped_channel(channels, query)
        self._scoped[query] = scoped
        return scoped

    def __getitem__(self, name: str | channel.Key) -> ScopedChannel:
        if not isinstance(name, channel.Key):
            return self.__getattr__(name)
        scoped = self._scoped.get(name)
        if scoped is None:
            channels = self._channel_retriever.retrieve(name)
            scoped = self._get_scoped_channel(channels, name.__str__())
            self._scoped[name] = scoped
        return scoped

    def _splice_cached(
        self, channels: list[channel.Payload]
    ) -> list[_InternalScopedChannel]:
        results = []
        for pld in channels:
            cached = self._cache.get(pld.key, None)
            if cached is None:
                cached = _InternalScopedChannel(
                    rng=self,
                    frame_client=self._frame_client,
                    payload=pld,
                    tasks=self._tasks,
                    ontology=self._ontology,
                )
                self._cache[pld.key] = cached
            results.append(cached)
        return results

    @property
    def ontology_id(self) -> ID:
        return ontology_id(self.key)

    @property
    def meta_data(self) -> kv.Client:
        if self._kv is None:
            raise _RANGE_NOT_CREATED
        return self._kv

    @property
    def _aliaser(self) -> alias_.Client:
        if self._cached_aliaser is None:
            raise _RANGE_NOT_CREATED
        return self._cached_aliaser

    @property
    def _frame_client(self) -> framer.Client:
        if self._cached_frame_client is None:
            raise _RANGE_NOT_CREATED
        return self._cached_frame_client

    @property
    def _client(self) -> Client:
        if self._cached_client is None:
            raise _RANGE_NOT_CREATED
        return self._cached_client

    @property
    def _tasks(self) -> TaskClient:
        if self._cached_tasks is None:
            raise _RANGE_NOT_CREATED
        return self._cached_tasks

    @property
    def _ontology(self) -> OntologyClient:
        if self._cached_ontology is None:
            raise _RANGE_NOT_CREATED
        return self._cached_ontology

    @property
    def _channel_retriever(self) -> ChannelRetriever:
        if self._channels is None:
            raise _RANGE_NOT_CREATED
        return self._channels

    @overload
    def set_alias(self, channel: channel.Key | str, alias: str) -> None: ...

    @overload
    def set_alias(self, channel: dict[channel.Key | str, str]) -> None: ...

    def set_alias(
        self,
        channel: channel.Key | str | dict[channel.Key | str, str],
        alias: str | None = None,
    ) -> None:
        if not isinstance(channel, dict):
            if alias is None:
                raise ValueError("Alias must be provided if channel is not a dict")
            channel = {channel: alias}
        corrected = {}
        for ch, alias in channel.items():
            if isinstance(ch, str):
                res = self._channel_retriever.retrieve(ch)
                corrected[util.one("Channel", ch, res).key] = alias
            else:
                corrected[ch] = alias
        self._aliaser.set(corrected)
        self._scoped.clear()

    def to_payload(self) -> Payload:
        return Payload(
            name=self.name,
            time_range=self.time_range,
            key=self.key,
            color=self.color,
            parent=self.parent,
        )

    @overload
    def write(
        self, channels: channel.Params, series: CrudeSeries | list[CrudeSeries]
    ) -> None: ...

    @overload
    def write(self, channels: framer.CrudeFrame[K]) -> None: ...

    def write(
        self,
        channels: channel.Params | framer.CrudeFrame[K],
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ) -> None:
        start = self.time_range.start
        if series is None:
            self._frame_client.write(start, cast(framer.CrudeFrame[K], channels))
            return
        if not isinstance(channels, (int, str, list, tuple, channel.Payload)):
            raise TypeError(
                "channels must be a channel key, name, or list when series is provided"
            )
        self._frame_client.write(start, channels, series)

    def create_child_range(
        self,
        *,
        name: str,
        time_range: TimeRange,
        color: str = "",
        key: Key = NIL,
    ) -> Range:
        return self._client.create(
            Range(name=name, time_range=time_range, color=color, key=key, parent=self)
        )

    @property
    def children(self) -> list[Range]:
        """Returns a list of child ranges of this range."""
        res = self._ontology.retrieve_children(self.ontology_id)
        range_children = [r for r in res if r.type == "range"]
        if len(range_children) == 0:
            return []
        child_keys: list[Key] = [
            UUID(r.key) for r in range_children if r.key is not None
        ]
        return self._client.retrieve(keys=child_keys)

    def snapshots(self) -> list[Task]:
        res = self._ontology.retrieve_children(self.ontology_id)
        tasks = [t for t in res if t.type == "task"]
        return self._tasks.retrieve(
            keys=[UUID(t.key) for t in tasks if t.key is not None]
        )


class Client:
    _frame_client: framer.Client
    _channels: ChannelRetriever
    _retriever: Retriever
    _writer: Writer
    _unary_client: UnaryClient
    _signals: Registry
    _tasks: TaskClient
    _ontology: OntologyClient

    def __init__(
        self,
        *,
        unary_client: UnaryClient,
        frame_client: framer.Client,
        writer: Writer,
        retriever: Retriever,
        channel_retriever: ChannelRetriever,
        signals: Registry,
        tasks: TaskClient,
        ontology: OntologyClient,
    ) -> None:
        self._unary_client = unary_client
        self._frame_client = frame_client
        self._writer = writer
        self._retriever = retriever
        self._channels = channel_retriever
        self._signals = signals
        self._tasks = tasks
        self._ontology = ontology

    @overload
    def create(
        self,
        *,
        name: str,
        time_range: TimeRange,
        color: str = "",
        retrieve_if_name_exists: bool = False,
        parent: Range | Payload | None = None,
        key: Key = NIL,
    ) -> Range:
        """Creates a named range spanning a region of time. This range is persisted
        to the cluster and is visible to all clients.

        :param name: The name of the range
        :param time_range: The time range of the range. This time range must be valid
        i.e. its start time must be less than or equal to its end time.
        :param retrieve_if_name_exists: If true, this method will retrieve the range
        if it already exists. This method will throw a ValidationError
        if the range already exists and has a DIFFERENT time range.
        :param parent: An optional parent range. Retrieve or create the parent first,
        then pass it directly.
        """

    @overload
    def create(
        self,
        ranges: Range,
        *,
        retrieve_if_name_exists: bool = False,
    ) -> Range:
        """Creates the given range. This range is persisted to the cluster and is
        visible to all clients.

        :param ranges: The range to create
        :param retrieve_if_name_exists: If true, this method will retrieve the range
        if it already exists. This method will throw a ValidationError if the range
        already exists and has a DIFFERENT time range.
        """

    @overload
    def create(
        self,
        ranges: list[Range],
        *,
        retrieve_if_name_exists: bool = False,
    ) -> list[Range]:
        """Creates the given ranges. These ranges are persisted to the cluster and are
        visible to all clients.

        :param ranges: The ranges to create
        :param retrieve_if_name_exists: If true, this method will retrieve the range
        if it already exists. This method will throw a ValidationError if the range
        already exists and has a DIFFERENT time range.
        """

    def create(
        self,
        ranges: Range | list[Range] | None = None,
        *,
        key: Key = NIL,
        name: str = "",
        time_range: TimeRange | None = None,
        color: str = "",
        retrieve_if_name_exists: bool = False,
        parent: Range | Payload | None = None,
    ) -> Range | list[Range]:
        is_single = True
        if ranges is None:
            to_create = [
                Range(
                    key=key,
                    name=name,
                    time_range=time_range if time_range is not None else TimeRange.ZERO,
                    color=color,
                    parent=parent,
                ).to_payload()
            ]
        elif isinstance(ranges, Range):
            to_create = [ranges.to_payload()]
        else:
            is_single = False
            to_create = [r.to_payload() for r in ranges]

        res: list[Range] = []
        if retrieve_if_name_exists:
            res = self.retrieve(names=[r.name for r in to_create])
            if is_single and len(res) > 1:
                filtered = [r for r in res if r.time_range == time_range]
                if len(filtered) == 0:
                    raise NotFoundError(
                        f"""
                        retrieve_if_name_exists was set to true, but {len(res)} ranges
                        were found matching {name} but none had the same time range as
                        passed to create. Synnax can't figure out which one you want!
                        """
                    )
                if len(filtered) > 1:
                    raise MultipleFoundError(
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
            res.extend(self._sugar(self._writer.create(to_create)))
        return res if not is_single else res[0]

    @overload
    def retrieve(
        self,
        key: Key | str,
        /,
        *,
        include_labels: bool = False,
        include_parent: bool = False,
    ) -> Range: ...

    @overload
    def retrieve(
        self,
        keys: list[Key] | list[str],
        /,
        *,
        include_labels: bool = False,
        include_parent: bool = False,
    ) -> list[Range]: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        include_labels: bool = False,
        include_parent: bool = False,
    ) -> Range: ...

    @overload
    def retrieve(
        self,
        *,
        name: str,
        include_labels: bool = False,
        include_parent: bool = False,
    ) -> Range: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[True],
        names: list[str] | tuple[str, ...] | None = None,
        keys: list[Key] | tuple[Key, ...] | None = None,
        search_term: str = "",
        overlaps_with: TimeRange | None = None,
        has_labels: list[label.Key] | None = None,
        limit: int | None = None,
        offset: int = 0,
        include_labels: bool = False,
        include_parent: bool = False,
    ) -> Range: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[False] = False,
        names: list[str] | tuple[str, ...] | None = None,
        keys: list[Key] | tuple[Key, ...] | None = None,
        search_term: str = "",
        overlaps_with: TimeRange | None = None,
        has_labels: list[label.Key] | None = None,
        limit: int | None = None,
        offset: int = 0,
        include_labels: bool = False,
        include_parent: bool = False,
    ) -> list[Range]: ...

    def retrieve(
        self,
        *args: Key | str | list[Key] | list[str] | Condition,
        key: Key | None = None,
        name: str | None = None,
        one: bool = False,
        names: list[str] | tuple[str, ...] | None = None,
        keys: list[Key] | tuple[Key, ...] | None = None,
        search_term: str = "",
        overlaps_with: TimeRange | None = None,
        has_labels: list[label.Key] | None = None,
        limit: int | None = None,
        offset: int = 0,
        include_labels: bool = False,
        include_parent: bool = False,
    ) -> Range | list[Range]:
        """Retrieves one range by key or name, or a list of ranges that match every
        condition and filter. A filter left at its default does not filter.

        :param args: Conditions from the ranger builder, such as
            ``ranger.name.contains("Hotfire")``, or one key, name, or list of either. A
            key or a name returns one range.
        :param key: The key of one range to retrieve.
        :param name: The name of one range to retrieve.
        :param one: Returns the one range that matches, instead of a list.
        :param names: Keeps ranges with one of these names.
        :param keys: Keeps ranges with one of these keys.
        :param search_term: Fuzzy matches range names.
        :param overlaps_with: Keeps ranges whose time range overlaps this one.
        :param has_labels: Keeps ranges that have one of these labels.
        :param limit: The maximum number of ranges to return.
        :param offset: The number of matching ranges to skip.
        :param include_labels: Attaches the labels of each range.
        :param include_parent: Attaches the parent of each range.
        :returns: One range for a key, a name, or one, else a list of the matching
            ranges.
        :raises NotFoundError: If no range matches a key, a name, or one.
        :raises MultipleFoundError: If more than one range matches a name or one.
        :raises ValueError: If key and name are both given, or either is combined with
            a list filter.
        :raises TypeError: If args are neither range conditions nor a shortcut.
        """
        conditions, query = util.shortcut(
            "Range", args, Condition, UUID, builder_gen.key, builder_gen.name
        )
        if key is not None and name is not None:
            raise ValueError("key and name cannot be combined; use keys or names")
        if key is not None or name is not None:
            if (
                args
                or one
                or search_term
                or offset
                or not all_none(keys, names, overlaps_with, has_labels, limit)
            ):
                raise ValueError(
                    "key and name cannot be combined with list filters; use keys or "
                    "names"
                )
            query = key or name
        keys = override(key, keys)
        names = override(name, names)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
            overlaps=overlaps_with,
            labels=has_labels,
        )
        req = RetrieveRequest(
            where=filter.merge(filter.join(Where, conditions), flat),
            search_term=search_term,
            limit=limit,
            offset=offset,
            include_labels=include_labels,
            include_parent=include_parent,
        )
        sug = self._sugar(self._retriever.retrieve(req))
        if query is None and not one:
            return sug
        return util.one("Range", req.where if query is None else query, sug)

    def delete(
        self,
        key: Key | list[Key] | tuple[Key, ...],
    ) -> None:
        self._writer.delete(normalize(key))

    def rename(self, key: Key, name: str) -> None:
        """Renames a range.

        :param key: The key of the range to rename.
        :param name: The new name for the range.
        """
        self._writer.rename(key, name)

    def search(
        self,
        term: str,
    ) -> list[Range]:
        return self._sugar(self._retriever.retrieve(RetrieveRequest(search_term=term)))

    def _sugar(self, ranges: list[Payload]) -> list[Range]:
        result: list[Range] = []
        for r in ranges:
            rng = Range(
                **r.model_dump(exclude={"labels", "parent"}),
                _frame_client=self._frame_client,
                _channel_retriever=self._channels,
                _kv=kv.Client(rng=r.key, client=self._unary_client),
                _aliaser=alias_.Client(rng=r.key, client=self._unary_client),
                _client=self,
                _ontology=self._ontology,
                _tasks=self._tasks,
            )
            rng.labels = r.labels
            if r.parent is not None:
                rng.parent = r.parent
            result.append(rng)
        return result

    def on_create(self, f: Callable[[Range], None]) -> Callable[[Latest], None]:
        @functools.wraps(f)
        def wrapper(state: Latest) -> None:
            f(self._sugar([Payload.model_validate(state[_RANGE_SET_CHANNEL])])[0])

        return self._signals.on([_RANGE_SET_CHANNEL], filter_f=lambda s: True)(wrapper)
