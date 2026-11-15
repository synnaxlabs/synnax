#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Any, Literal, overload, override

from pydantic import PrivateAttr

from freighter import Empty, UnaryClient
from synnax import framer, group as group_, node
from synnax.channel.payload import (
    Key,
    Params,
    normalize_params,
)
from synnax.channel.retrieve import Retriever, params_where
from synnax.channel.types_gen import (
    CountLimitedResponse,
    Cursor,
    Operation,
    Order,
    Payload,
    RetrieveRequest,
    Status,
    Where,
    WhereInput,
    resource_id,
)
from synnax.channel.writer import Writer
from synnax.exceptions import MultipleFoundError, ValidationError
from synnax.resource import ID
from synnax.telem import (
    CrudeDataType,
    CrudeSeries,
    CrudeTimeStamp,
    DataType,
    MultiSeries,
    TimeRange,
)
from synnax.util import request_where, returns_one, sole
from synnax.util.send_required import send_required
from x import control, filter
from x.lists import normalize


class Channel(Payload):
    """A channel is a logical collection of samples emitted by or representing the
    values of a single source. See https://docs.synnaxlabs.com/reference/channels for an
    introduction to channels and how they work.
    """

    _cached_frame_client: framer.Client | None = PrivateAttr(None)
    _client: Client | None = PrivateAttr(None)

    def __init__(
        self,
        *,
        name: str,
        data_type: CrudeDataType,
        is_index: bool = False,
        index: Key = 0,
        leaseholder: node.Key = 0,
        key: Key = 0,
        virtual: bool | None = None,
        internal: bool = False,
        expression: str = "",
        operations: list[Operation] | None = None,
        concurrency: control.Concurrency = control.Concurrency.exclusive,
        status: Status | None = None,
        group: group_.Key | None = None,
        _frame_client: framer.Client | None = None,
        _client: Client | None = None,
    ) -> None:
        """Initializes a new Channel using the given parameters. It's important to note
        that this does not create the Channel in the cluster. To create the channel,
        call client.channels.create(channel).

        :param name: A name for the channel.
        :param data_type: The data type of the samples in the channel. For example,
            `"float32"`.
        :param is_index: Boolean indicating whether the channel is an index. Index
        channels should have a data type of synnax.TIMESTAMP.
        :param index: The key of the channel that indexes this channel.
        :param leaseholder: The node that holds the lease for this channel. If you don't
        know what this is, leave it at the default value of 0.
        :param virtual: Boolean indicating whether the channel is virtual. Virtual
        channels do not store any data, and are used for streaming purposes only.
        :param expression: An optional Lua expression that defines the channel as a
        calculation of another channel. If this is set, the channel will be
        automatically configured as virtual.
        :param internal: Boolean indicating whether the channel is internal. Internal
        channels are not visible to the user and are used for internal purposes only.
        :param operations: A list of operations to apply to the channel. Operations
        include aggregations like min, max, avg over a time duration or triggered by
        a reset channel.
        :param group: The group the channel sits in, if any.
        :returns: The created channel.
        :param _frame_client: The backing client for reading and writing data to and
        from the channel. This is provided by the Synnax py during calls to
        .channels.create() and .channels.retrieve() and should not be set by the caller.
        """
        if operations is None:
            operations = []
        if virtual is None:
            virtual = len(expression) > 0
        super().__init__(
            data_type=DataType(data_type),
            name=name,
            leaseholder=leaseholder,
            key=key,
            is_index=is_index,
            index=index,
            internal=internal,
            virtual=virtual,
            expression=expression,
            operations=operations,
            concurrency=concurrency,
            status=status,
            group=group,
        )
        self._cached_frame_client = _frame_client
        self._client = _client

    @overload
    def read(
        self,
        start_or_range: TimeRange,
    ) -> MultiSeries: ...

    @overload
    def read(
        self,
        start_or_range: CrudeTimeStamp,
        end: CrudeTimeStamp,
    ) -> MultiSeries: ...

    def read(
        self,
        start_or_range: CrudeTimeStamp | TimeRange,
        end: CrudeTimeStamp | None = None,
    ) -> MultiSeries:
        """Reads telemetry from the channel between the two timestamps.

        :param start_or_range: The starting timestamp of the range to read from.
        :param end: The ending timestamp of the range to read from.
        :returns: A tuple containing a numpy array of the telemetry and a TimeRange
        representing the range of telemetry. The start of the time range represents
        the timestamp of the first sample in the array.
        :raises ContiguityError: If the telemetry between start and end is
            non-contiguous.
        """
        tr = TimeRange(start_or_range, end)
        return self._frame_client.read(tr, self.key)

    def write(self, start: CrudeTimeStamp, data: CrudeSeries) -> None:
        """Writes telemetry to the channel starting at the given timestamp.

        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        self._frame_client.write(start, self.key, data)

    def rename(self, name: str) -> None:
        """Renames the channel.

        :param name: The new name for the channel.
        :returns: None.
        """
        if self._client is None:
            raise ValidationError("Cannot rename a channel that has not been created.")
        self._client.rename(self.key, name)

    @property
    def resource_id(self) -> ID:
        return resource_id(self.key)

    @property
    def _frame_client(self) -> framer.Client:
        if self._cached_frame_client is None:
            raise ValidationError(
                "Cannot read from or write to channel that has not been created."
            )
        return self._cached_frame_client

    @override
    def __hash__(self) -> int:
        return hash(self.key)

    @override
    def __eq__(self, other: object) -> bool:
        if not isinstance(other, Channel):
            return NotImplemented
        return self.key == other.key

    def to_payload(self) -> Payload:
        return Payload(
            data_type=self.data_type,
            name=self.name,
            leaseholder=self.leaseholder,
            key=self.key,
            index=self.index,
            is_index=self.is_index,
            virtual=self.virtual,
            internal=self.internal,
            expression=self.expression,
            operations=self.operations,
            group=self.group,
        )


class Client:
    """The core py class for executing channel operations against a Synnax cluster."""

    _client: UnaryClient
    _frame_client: framer.Client
    _retriever: Retriever
    _creator: Writer

    def __init__(
        self,
        *,
        client: UnaryClient,
        frame_client: framer.Client,
        retriever: Retriever,
        creator: Writer,
    ):
        self._client = client
        self._frame_client = frame_client
        self._retriever = retriever
        self._creator = creator

    def delete(self, channels: Params) -> None:
        """Deletes on or more channels from the cluster"""
        self._creator.delete(channels)

    def count_limited(self) -> int:
        """Retrieves how many channels count against the license channel limit (not
        internal, not virtual).

        :returns: The number of channels.
        :raises AccessDenied: If the caller cannot retrieve every channel.
        """
        return send_required(
            self._client, "/channel/count-limited", Empty(), CountLimitedResponse
        ).count

    @overload
    def create(
        self,
        *,
        data_type: CrudeDataType = DataType.UNKNOWN,
        name: str = "",
        index: Key = 0,
        is_index: bool = False,
        leaseholder: node.Key = 0,
        virtual: bool | None = None,
        expression: str = "",
        operations: list[Operation] | None = None,
        group: group_.Key | None = None,
        retrieve_if_name_exists: bool = False,
    ) -> Channel: ...

    @overload
    def create(
        self, channels: Channel, *, retrieve_if_name_exists: bool = False
    ) -> Channel: ...

    @overload
    def create(
        self, channels: list[Channel], *, retrieve_if_name_exists: bool = False
    ) -> list[Channel]: ...

    def create(
        self,
        channels: Channel | list[Channel] | None = None,
        *,
        data_type: CrudeDataType = DataType.UNKNOWN,
        name: str = "",
        is_index: bool = False,
        index: Key = 0,
        leaseholder: node.Key = 0,
        virtual: bool | None = None,
        expression: str = "",
        operations: list[Operation] | None = None,
        group: group_.Key | None = None,
        retrieve_if_name_exists: bool = False,
    ) -> Channel | list[Channel]:
        """Creates new channel(s) in the Synnax cluster.

        Overload 1:
        :param data_type: The data type of the samples in the channel. For example,
            `"float32"`.
        :param name: A name for the channel.
        :param is_index: Boolean indicating whether the channel is an index. Index
        channels should have a data type of synnax.TIMESTAMP.
        :param index: The key of the channel that indexes this channel.
        :param leaseholder: The node that holds the lease for this channel. If you don't
        know what this is, leave it at the default value of 0.
        :param virtual: Boolean indicating whether the channel is virtual. Virtual
        channels do not store any data, and are used for streaming purposes only.
        :param expression: An optional expression that defines the channel as a
        calculation of another channel. If this is set, the channel will be
        automatically configured as virtual.
        set if expression is not an empty string. If expression is not an empty string,
        this should have at least one channel.
        :param group: The group to create the channel in, if any.
        :param retrieve_if_name_exists: Boolean indicating whether to retrieve channels
        with the same name if they already exist in the cluster.
        :returns: The created channel.

        Overload 2:

        :param channels: A single channel to create.
        :param retrieve_if_name_exists: Boolean indicating whether to retrieve channels
        with the same name if they already exist in the cluster.
        :returns: The created channel.

        Overload 3:

        :param channels: A list of channels to create.
        :param retrieve_if_name_exists: Boolean indicating whether to retrieve channels
        with the same name if they already exist in the cluster.
        :returns: The created channels.
        """

        _channels: list[Payload]
        if channels is None:
            if is_index and data_type == DataType.UNKNOWN:
                data_type = DataType.TIMESTAMP
            if operations is None:
                operations = []
            _channels = [
                Payload(
                    name=name,
                    leaseholder=leaseholder,
                    data_type=DataType(data_type),
                    index=index,
                    is_index=is_index,
                    virtual=virtual if virtual is not None else len(expression) > 0,
                    expression=expression,
                    operations=operations,
                    group=group,
                )
            ]
        elif isinstance(channels, Channel):
            _channels = [channels.to_payload()]
        else:
            _channels = [c.to_payload() for c in channels]

        created = self._sugar(self._creator.create(_channels, retrieve_if_name_exists))
        return created if isinstance(channels, list) else created[0]

    @overload
    def retrieve(
        self,
        channel: Key | str,
        *,
        leaseholder: filter.SetInput[node.Key] = ...,
        data_type: filter.SetInput[DataType] = ...,
        is_index: filter.BoolInput = ...,
        internal: filter.BoolInput = ...,
        virtual: bool = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        data_types: list[CrudeDataType] | None = ...,
        not_data_types: list[CrudeDataType] | None = ...,
        node_key: node.Key | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Channel: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        leaseholder: filter.SetInput[node.Key] = ...,
        data_type: filter.SetInput[DataType] = ...,
        is_index: filter.BoolInput = ...,
        internal: filter.BoolInput = ...,
        virtual: bool = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Channel: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        leaseholder: filter.SetInput[node.Key] = ...,
        data_type: filter.SetInput[DataType] = ...,
        is_index: filter.BoolInput = ...,
        internal: filter.BoolInput = ...,
        virtual: bool = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Channel: ...

    @overload
    def retrieve(
        self,
        channel: list[Key] | tuple[Key, ...] | list[str] | tuple[str, ...] | None = ...,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        leaseholder: filter.SetInput[node.Key] = ...,
        data_type: filter.SetInput[DataType] = ...,
        is_index: filter.BoolInput = ...,
        internal: filter.BoolInput = ...,
        virtual: bool = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        data_types: list[CrudeDataType] | None = ...,
        not_data_types: list[CrudeDataType] | None = ...,
        node_key: node.Key | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: Literal[True],
    ) -> Channel: ...

    @overload
    def retrieve(
        self,
        channel: list[Key] | tuple[Key, ...] | list[str] | tuple[str, ...] | None = ...,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        leaseholder: filter.SetInput[node.Key] = ...,
        data_type: filter.SetInput[DataType] = ...,
        is_index: filter.BoolInput = ...,
        internal: filter.BoolInput = ...,
        virtual: bool = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        data_types: list[CrudeDataType] | None = ...,
        not_data_types: list[CrudeDataType] | None = ...,
        node_key: node.Key | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        include_status: bool = ...,
        one: Literal[False] = ...,
    ) -> list[Channel]: ...

    def retrieve(
        self,
        channel: Params | None = None,
        *,
        where: Where | None = None,
        data_types: list[CrudeDataType] | None = None,
        not_data_types: list[CrudeDataType] | None = None,
        node_key: node.Key | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        include_status: bool = False,
        one: bool = False,
        **fields: Any,
    ) -> Channel | list[Channel]:
        """Retrieves channels from the Core. With no arguments, retrieves every
        channel.

        Each filter field takes a value, a list of values, or an operator, and a
        channel must match every field. A plain ``key`` or ``name``, or a single key
        or name as ``channel``, returns one channel.

        :param channel: The key or name of one channel, or a list of keys or names. A
            name matches exactly. To match a pattern, use where with a name regex.
        :param key: Keeps channels with this key.
        :param name: Keeps channels with this name.
        :param leaseholder: Keeps channels whose leaseholder is this node.
        :param data_type: Keeps channels with this data type.
        :param is_index: Keeps only index or only non-index channels.
        :param internal: Keeps only internal or only non-internal channels.
        :param virtual: Keeps only virtual channels that are not calculated, or only
            the other channels.
        :param groups: Keeps channels in any of these groups.
        :param grouped: Keeps only channels in a group when True, and only channels in
            no group when False.
        :param and_: Keeps channels that match every filter in the list.
        :param or_: Keeps channels that match any filter in the list.
        :param not_: Keeps channels that do not match the filter.
        :param where: A filter tree that the channels must also match.
        :param data_types: Keeps channels with one of these data types.
        :param not_data_types: Drops channels with one of these data types.
        :param node_key: Keeps channels whose leaseholder is this node.
        :param search_term: Fuzzy matches channel names.
        :param limit: The maximum number of channels to retrieve.
        :param order: The order of the channels. Defaults to key order.
        :param after: Retrieves the channels past this position in the order: the key of
            the last channel of the page before, and its value of the order field.
        :param include_status: Attaches each channel's status. A channel with no
            status gets None.
        :param one: Returns one channel for any filter.
        :returns: One channel when one is wanted, otherwise a list of channels. The
            list leaves out keys that do not exist.
        :raises NotFoundError: If one channel is wanted and none matches.
        :raises MultipleFoundError: If one channel is wanted and more than one
            matches.
        :raises ValidationError: If a filter field is None, or if one channel is
            wanted beside a limit or a cursor.
        """
        selected = Where()
        positional = False
        if channel is not None:
            selected = params_where(channel)
            positional = normalize_params(channel).single
        single = returns_one(fields, one or positional, limit=limit, after=after)
        flat = Where(
            key=selected.key,
            name=selected.name,
            leaseholder=(
                None if node_key is None else filter.Set[node.Key](eq=node_key)
            ),
            data_type=(
                None
                if data_types is None
                else filter.Set[DataType](in_=_data_types(data_types))
            ),
            not_=(
                None
                if not_data_types is None
                else Where(
                    data_type=filter.Set[DataType](in_=_data_types(not_data_types))
                )
            ),
        )
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
            include_status=include_status,
        )
        res = self._retriever.filter(req)
        sug = self._sugar(res)
        if not single:
            return sug
        if channel is not None and positional and len(res) > 1:
            raise _multiple_results_error(channel, res)
        return sole("channel", req, sug)

    @overload
    def rename(self, keys: Key, names: str) -> None:
        """Renames a channel in the cluster.

        :param keys: The key of the channel to rename.
        :param names: The new name for the channel.
        :returns: None.
        """

    @overload
    def rename(
        self, keys: list[Key] | tuple[Key, ...], names: list[str] | tuple[str, ...]
    ) -> None:
        """Renames one or more channels in the cluster.

        :param keys: The keys of the channels to rename.
        :param names: The new names for the channels.
        :returns: None.
        """

    def rename(
        self,
        keys: Key | list[Key] | tuple[Key, ...],
        names: str | list[str] | tuple[str, ...],
    ) -> None:
        """Renames one or more channels in the cluster.

        :param keys: The keys of the channels to rename.
        :param names: The new names for the channels.
        :returns: None.
        """
        self._creator.rename(normalize(keys), normalize(names))

    def set_group(self, keys: Key | list[Key], group: group_.Key | None = None) -> None:
        """Moves channels into a group, or out of their group when group is None.

        :param keys: The key of the channel to move, or a list of keys.
        :param group: The key of the group to move the channels into.
        """
        self._creator.set_group(normalize(keys), group)

    def set_expression(self, key: Key, expression: str) -> None:
        """Sets the expression of a calculated channel. The Core infers the data type
        of the channel from the new expression.

        :param key: The key of the calculated channel.
        :param expression: The new expression.
        :raises NotFoundError: If no channel has the given key.
        :raises ValidationError: If the channel is not calculated.
        """
        self._creator.set_expression(key, expression)

    def _sugar(self, channels: list[Payload]) -> list[Channel]:
        return [
            Channel(**c.model_dump(), _frame_client=self._frame_client)
            for c in channels
        ]


def _data_types(data_types: list[CrudeDataType]) -> list[DataType]:
    return [DataType(dt) for dt in data_types]


def _multiple_results_error(
    channel: Params,
    results: list[Payload],
) -> MultipleFoundError:
    msg = f"""

{len(results)} channels matching '{channel}' found. If you'd like to retrieve all
of them, pass in '{channel}' as an array i.e. ['{channel}'] instead of {channel}.

The channels found were:
    """

    # append a max of five results to the error message. If we have more than five,
    # we'll just say "and x more" at the end.
    for i in range(min(5, len(results))):
        msg += f"{results[i]!s}, "

    if len(results) > 5:
        msg += f"and {len(results) - 5} more."

    return MultipleFoundError(msg)
