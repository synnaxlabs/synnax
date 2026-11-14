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
from collections.abc import Callable, Iterator, Sequence
from typing import Any, Literal, cast, overload, override

import numpy as np
from pydantic import BaseModel, JsonValue, PrivateAttr

from freighter import UnaryClient
from synnax import channel, framer, label, node
from synnax.exceptions import ConfigurationError, MultipleFoundError, NotFoundError
from synnax.framer.frame import K
from synnax.ranger import actions_gen, kv
from synnax.ranger.retrieve import Retriever
from synnax.ranger.types_gen import (
    Cursor,
    Key,
    Order,
    Payload,
    RetrieveRequest,
    Stage,
    Where,
    WhereInput,
    resource_id,
)
from synnax.ranger.writer import Writer
from synnax.resource import ID
from synnax.signals.signals import Registry
from synnax.state import Latest
from synnax.telem import (
    CrudeSeries,
    CrudeTimeStamp,
    DataType,
    MultiSeries,
    SampleValue,
    TimeRange,
    TimeSpan,
    TimeStamp,
)
from synnax.util import request_where, returns_one, sole
from x import color as color_, filter
from x.interop import ArrayOperators
from x.lists import normalize
from x.params import deprecate_positional, require_named_params
from x.uuid import NIL

_RANGE_SET_CHANNEL = "sy_range_set"


def _set(path: list[str], value: JsonValue) -> actions_gen.SetAction:
    return actions_gen.set(actions_gen.SetPayload(path=path, value=value))


def _remove(field: str) -> actions_gen.RemoveAction:
    return actions_gen.remove(actions_gen.RemovePayload(key=field))


# A string holds every 64-bit timestamp exactly.
def _nanoseconds(time: CrudeTimeStamp) -> str:
    return str(int(TimeStamp(time)))


# Pydantic declares BaseModel.__getattr__ only at runtime.
_model_getattr: Callable[[BaseModel, str], object] = vars(BaseModel)["__getattr__"]


class _InternalScopedChannel(ArrayOperators, channel.Payload):
    _cached_range: Range | None = PrivateAttr(None)
    """The range that this channel belongs to."""
    _cached_frame_client: framer.Client | None = PrivateAttr(None)
    """The frame client for executing read operations."""
    _cache: MultiSeries | None = PrivateAttr(None)
    """An internal cache to prevent repeated reads from the same channel."""

    # ArrayOperators clears __hash__, but writes can key data by channel payload.
    __hash__ = channel.Payload.__hash__

    def __init__(
        self,
        rng: Range,
        frame_client: framer.Client,
        payload: channel.Payload,
    ):
        super().__init__(**payload.model_dump())
        self._cached_range = rng
        self._cached_frame_client = frame_client

    @property
    def _range(self) -> Range:
        if self._cached_range is None:
            raise _RANGE_NOT_CREATED
        return self._cached_range

    @property
    def time_range(self) -> TimeRange:
        return self._range.time_range

    @override
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

    @override
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
    in the returned values.
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

    @override
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
    def leaseholder(self) -> node.Key:
        self._guard()
        return self._internal[0].leaseholder

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
    _channels: channel.Retriever | None = PrivateAttr(None)
    """For retrieving channels from the cluster."""
    _kv: kv.Client | None = PrivateAttr(None)
    """Key-value store for storing metadata about the range."""
    _cache: dict[channel.Key, _InternalScopedChannel] = PrivateAttr({})
    """A writer for creating child ranges"""
    _scoped: dict[channel.Key | str, ScopedChannel] = PrivateAttr({})
    """Resolved channel queries, so each name or key is retrieved once."""
    _cached_client: Client | None = PrivateAttr(None)
    _cached_label_client: label.Client | None = PrivateAttr(None)

    @deprecate_positional("name", "time_range", "key", "color", "parent")
    def __init__(
        self,
        *,
        name: str,
        time_range: TimeRange,
        key: Key = NIL,
        color: color_.Crude | None = None,
        labels: Sequence[label.Key] = (),
        parent: Key | Payload | None = None,
        has_children: bool = False,
        _frame_client: framer.Client | None = None,
        _channel_retriever: channel.Retriever | None = None,
        _kv: kv.Client | None = None,
        _client: Client | None = None,
        _label_client: label.Client | None = None,
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
        :param color: An optional display color. None means the range has no color.
        :param labels: The keys of the labels to attach to the range.
        :param parent: The parent range, or its key. None means the range has no parent.
        :param has_children: Whether other ranges are nested in the range. The Core
            sets it on each retrieve, and it is never stored.
        :param _frame_client: The backing client for reading and writing data to
            and from the cluster. This is provided by Synnax during calls to
            .ranges.create() and .ranges.retrieve(), and should not be set by the user.
        :param _channel_retriever: The backing client for retrieving channels to
            and from the cluster. This is provided by Synnax during calls to
            .ranges.create() and .ranges.retrieve(), and should not be set by the user.
        """
        super().__init__(
            name=name,
            time_range=time_range,
            key=key,
            color=None if color is None else color_.Color(color),
            labels=list(labels),
            parent=parent.key if isinstance(parent, Payload) else parent,
            has_children=has_children,
        )
        self._cached_frame_client = _frame_client
        self._channels = _channel_retriever
        self._kv = _kv
        self._cached_client = _client
        self._cached_label_client = _label_client

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
                )
                self._cache[pld.key] = cached
            results.append(cached)
        return results

    @property
    def resource_id(self) -> ID:
        return resource_id(self.key)

    @property
    def meta_data(self) -> kv.Client:
        if self._kv is None:
            raise _RANGE_NOT_CREATED
        return self._kv

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
    def _label_client(self) -> label.Client:
        if self._cached_label_client is None:
            raise _RANGE_NOT_CREATED
        return self._cached_label_client

    @property
    def _channel_retriever(self) -> channel.Retriever:
        if self._channels is None:
            raise _RANGE_NOT_CREATED
        return self._channels

    def to_payload(self) -> Payload:
        return Payload(
            name=self.name,
            time_range=self.time_range,
            key=self.key,
            color=self.color,
            labels=self.labels,
            parent=self.parent,
            has_children=self.has_children,
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
        color: color_.Crude | None = None,
        key: Key = NIL,
    ) -> Range:
        return self._client.create(
            Range(name=name, time_range=time_range, color=color, key=key, parent=self)
        )

    @property
    def children(self) -> list[Range]:
        """Retrieves the ranges whose parent is this range from the Core."""
        return self._client.retrieve(parents=[self.key])

    def retrieve_labels(self) -> list[label.Label]:
        """Retrieves the labels that the Core stores on the range."""
        keys = self._client.retrieve(key=self.key).labels
        # A retrieve with no keys returns every label.
        if len(keys) == 0:
            return []
        return self._label_client.retrieve(keys=keys)

    def add_label(self, *labels: label.Key) -> None:
        """Adds labels to the range.

        :param labels: The keys of the labels to add.
        """
        self._label_client.label(self.resource_id, list(labels))

    def remove_label(self, *labels: label.Key) -> None:
        """Removes labels from the range.

        :param labels: The keys of the labels to remove.
        """
        self._label_client.remove(self.resource_id, list(labels))


class Client:
    _frame_client: framer.Client
    _channels: channel.Retriever
    _retriever: Retriever
    _writer: Writer
    _unary_client: UnaryClient
    _signals: Registry
    _labels: label.Client

    def __init__(
        self,
        *,
        unary_client: UnaryClient,
        frame_client: framer.Client,
        writer: Writer,
        retriever: Retriever,
        channel_retriever: channel.Retriever,
        signals: Registry,
        labels: label.Client,
    ) -> None:
        self._unary_client = unary_client
        self._frame_client = frame_client
        self._writer = writer
        self._retriever = retriever
        self._channels = channel_retriever
        self._signals = signals
        self._labels = labels

    @overload
    def create(
        self,
        *,
        name: str,
        time_range: TimeRange,
        color: color_.Crude | None = None,
        retrieve_if_name_exists: bool = False,
        labels: Sequence[label.Key] = (),
        parent: Key | Payload | None = None,
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
        :param labels: The keys of the labels to attach to the range.
        :param parent: The parent range, or its key.
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
        color: color_.Crude | None = None,
        retrieve_if_name_exists: bool = False,
        labels: Sequence[label.Key] = (),
        parent: Key | Payload | None = None,
    ) -> Range | list[Range]:
        is_single = True
        if ranges is None:
            to_create = [
                Range(
                    key=key,
                    name=name,
                    time_range=time_range if time_range is not None else TimeRange.ZERO,
                    color=color,
                    labels=labels,
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
        *,
        key: Key,
        name: filter.StringInput = ...,
        overlaps: TimeRange = ...,
        labels: label.Key | list[label.Key] = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        labeled: bool = ...,
        stages: Stage | list[Stage] = ...,
        recent: TimeSpan = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Range: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        overlaps: TimeRange = ...,
        labels: label.Key | list[label.Key] = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        labeled: bool = ...,
        stages: Stage | list[Stage] = ...,
        recent: TimeSpan = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Range: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        overlaps: TimeRange = ...,
        labels: label.Key | list[label.Key] = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        labeled: bool = ...,
        stages: Stage | list[Stage] = ...,
        recent: TimeSpan = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        names: list[str] | tuple[str, ...] | None = ...,
        keys: list[Key] | tuple[Key, ...] | None = ...,
        overlaps_with: TimeRange | None = ...,
        has_labels: list[label.Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Range: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        overlaps: TimeRange = ...,
        labels: label.Key | list[label.Key] = ...,
        parents: Key | list[Key] = ...,
        nested: bool = ...,
        labeled: bool = ...,
        stages: Stage | list[Stage] = ...,
        recent: TimeSpan = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        names: list[str] | tuple[str, ...] | None = ...,
        keys: list[Key] | tuple[Key, ...] | None = ...,
        overlaps_with: TimeRange | None = ...,
        has_labels: list[label.Key] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        one: Literal[False] = ...,
    ) -> list[Range]: ...

    @require_named_params(example_params=("name", "My Range"))
    def retrieve(
        self,
        *,
        where: Where | None = None,
        names: list[str] | tuple[str, ...] | None = None,
        keys: list[Key] | tuple[Key, ...] | None = None,
        overlaps_with: TimeRange | None = None,
        has_labels: list[label.Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Range | list[Range]:
        """Retrieves ranges from the Core. With no arguments, retrieves every range.

        Each filter field takes a value, a list of values, or an operator, and a range
        must match every field. A plain ``key`` or ``name`` returns one range.

        :param key: Keeps ranges with this key.
        :param name: Keeps ranges with this name.
        :param overlaps: Keeps ranges whose time range overlaps this one.
        :param labels: Keeps ranges that have this label, or any label in the list.
        :param parents: Keeps the ranges nested in this range, or in any range in the
            list.
        :param nested: Keeps the ranges nested in a range, or top-level ranges when
            false.
        :param labeled: Keeps ranges that have a label, or ranges with none when false.
        :param stages: Keeps ranges in this stage, or in any stage in the list, at the
            Core's clock.
        :param recent: Keeps ranges that overlap the span ending now, or starting now if
            negative.
        :param and_: Keeps ranges that match every filter in the list.
        :param or_: Keeps ranges that match any filter in the list.
        :param not_: Keeps ranges that do not match the filter.
        :param where: A filter tree that the ranges must also match.
        :param names: Keeps ranges with one of these names.
        :param keys: Keeps ranges with one of these keys.
        :param overlaps_with: Keeps ranges whose time range overlaps this one.
        :param has_labels: Keeps ranges that have one of these labels.
        :param search_term: A fuzzy search term matched against range names.
        :param limit: The maximum number of ranges to retrieve.
        :param order: The order of the ranges. Defaults to key order.
        :param after: Retrieves the ranges past this position in the order: the key of
            the last range of the page before, and its value of the order field.
        :param one: Returns one range for any filter.
        :returns: One range for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of ranges. The list leaves out keys that do not exist.
        :raises NotFoundError: If one range is wanted and none matches.
        :raises MultipleFoundError: If one range is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one range is wanted
            beside a limit or a cursor.
        """
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=list(keys)),
            name=None if names is None else filter.String(in_=list(names)),
            overlaps=overlaps_with,
            labels=has_labels,
        )
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
        )
        ranges = self._sugar(self._retriever.retrieve(req))
        return sole("range", req, ranges) if single else ranges

    def delete(
        self,
        key: Key | list[Key] | tuple[Key, ...],
    ) -> None:
        self._writer.delete(normalize(key))

    def dispatch(
        self, key: Key, actions: actions_gen.Action | list[actions_gen.Action]
    ) -> None:
        """Applies actions to a range, in order, in one transaction.

        :param key: The key of the range.
        :param actions: The actions to apply, such as ``ranger.set`` and
            ``ranger.remove``. A timestamp in an action is a string of its exact
            nanoseconds.
        :raises NotFoundError: If no range has the given key.
        :raises ValidationError: If an action is invalid.
        """
        self._writer.dispatch(key, normalize(actions))

    def rename(self, key: Key, name: str) -> None:
        """Renames a range.

        :param key: The key of the range to rename.
        :param name: The new name for the range.
        """
        self.dispatch(key, _set(["name"], name))

    def set_end(self, key: Key, end: CrudeTimeStamp) -> None:
        """Sets the end of the time range of a range.

        :param key: The key of the range.
        :param end: The new end of the time range.
        :raises NotFoundError: If no range has the given key.
        """
        self.dispatch(key, _set(["time_range", "end"], _nanoseconds(end)))

    def set_time_range(self, key: Key, time_range: TimeRange) -> None:
        """Sets the start and end of a range.

        :param key: The key of the range.
        :param time_range: The new time range.
        :raises NotFoundError: If no range has the given key.
        :raises ValidationError: If the start is after the end.
        """
        value: JsonValue = {
            "start": _nanoseconds(time_range.start),
            "end": _nanoseconds(time_range.end),
        }
        self.dispatch(key, _set(["time_range"], value))

    def set_color(self, key: Key, color: color_.Crude | None) -> None:
        """Sets the color of a range, or clears it when color is None.

        :param key: The key of the range.
        :param color: The new color, such as ``"#FF0000"``.
        :raises NotFoundError: If no range has the given key.
        """
        if color is None:
            self.dispatch(key, _remove("color"))
        else:
            self.dispatch(key, _set(["color"], color_.Color(color).model_dump()))

    def set_parent(self, key: Key, parent: Key | Payload | None) -> None:
        """Nests a range in a parent range, or makes it a top-level range when parent
        is None.

        :param key: The key of the range.
        :param parent: The parent range, or its key.
        :raises NotFoundError: If no range has the given key.
        :raises ValidationError: If the parent is the range itself or a range nested
            in it.
        """
        if parent is None:
            self.dispatch(key, _remove("parent"))
            return
        parent_key = parent.key if isinstance(parent, Payload) else parent
        self.dispatch(key, _set(["parent"], str(parent_key)))

    def search(
        self,
        term: str,
    ) -> list[Range]:
        return self._sugar(self._retriever.retrieve(RetrieveRequest(search_term=term)))

    def _sugar(self, ranges: list[Payload]) -> list[Range]:
        result: list[Range] = []
        for r in ranges:
            rng = Range(
                **r.model_dump(),
                _frame_client=self._frame_client,
                _channel_retriever=self._channels,
                _kv=kv.Client(rng=r.key, client=self._unary_client),
                _client=self,
                _label_client=self._labels,
            )
            result.append(rng)
        return result

    def on_create(self, f: Callable[[Range], None]) -> Callable[[Latest], None]:
        @functools.wraps(f)
        def wrapper(state: Latest) -> None:
            f(self._sugar([Payload.model_validate(state[_RANGE_SET_CHANNEL])])[0])

        return self._signals.on([_RANGE_SET_CHANNEL], filter_f=lambda s: True)(wrapper)
