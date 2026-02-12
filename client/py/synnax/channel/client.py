#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import overload

from pydantic import PrivateAttr

from synnax import framer
from synnax.channel.payload import (
    Key,
    Operation,
    Params,
    Payload,
    normalize_params,
    ontology_id,
)
from synnax.channel.retrieve import Retriever
from synnax.channel.writer import Writer
from synnax.exceptions import MultipleFoundError, NotFoundError, ValidationError
from synnax.ontology.payload import ID
from synnax.telem import (
    CrudeDataType,
    CrudeSeries,
    CrudeTimeStamp,
    DataType,
    MultiSeries,
    TimeRange,
)
from synnax.util.normalize import normalize


class Channel(Payload):
    """A channel is a logical collection of samples emitted by or representing the
    values of a single source. See
    https://docs.synnaxlabs.com/reference/concepts/channels for an introduction to
    channels and how they work.
    """

    ___frame_client: framer.Client | None = PrivateAttr(None)
    __client: Client | None = PrivateAttr(None)

    def __init__(
        self,
        *,
        name: str,
        data_type: CrudeDataType,
        is_index: bool = False,
        index: Key = 0,
        leaseholder: int = 0,
        key: Key = 0,
        virtual: bool | None = None,
        internal: bool = False,
        expression: str = "",
        operations: list[Operation] | None = None,
        _frame_client: framer.Client | None = None,
        _client: Client | None = None,
    ) -> None:
        """Initializes a new Channel using the given parameters. It's important to note
        that this does not create the Channel in the cluster. To create the channel,
        call client.channels.create(channel).

        :param name: A name for the channel.
        :param data_type: The data type of the samples in the channel. For example, `"float32"`.
        :param is_index: Boolean indicating whether the channel is an index. Index
        channels should have a data type of synnax.TIMESTAMP.
        :param index: The key of the channel that indexes this channel.
        :param leaseholder: The node that holds the lease for this channel. If you don't know
        what this is, leave it at the default value of 0.
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
        :returns: The created channel.
        :param _frame_client: The backing client for reading and writing data to and
        from the channel. This is provided by the Synnax py during calls to
        .channels.create() and .channels.retrieve() and should not be set by the caller.
        """
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
        )
        self.___frame_client = _frame_client
        self.__client = _client

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
        :raises ContiguityError: If the telemetry between start and end is non-contiguous.
        """
        tr = TimeRange(start_or_range, end)
        return self.__frame_client.read(tr, self.key)

    def write(self, start: CrudeTimeStamp, data: CrudeSeries) -> None:
        """Writes telemetry to the channel starting at the given timestamp.

        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        self.__frame_client.write(start, self.key, data)

    def rename(self, name: str) -> None:
        """Renames the channel.

        :param name: The new name for the channel.
        :returns: None.
        """
        self.__client.rename(self.key, name)

    @property
    def ontology_id(self) -> ID:
        return ontology_id(self.key)

    @property
    def __frame_client(self) -> framer.Client:
        if self.___frame_client is None:
            raise ValidationError(
                "Cannot read from or write to channel that has not been created."
            )
        return self.___frame_client

    def __hash__(self) -> int:
        return hash(self.key)

    def __eq__(self, other) -> bool:
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
        )


class Client:
    """The core py class for executing channel operations against a Synnax cluster."""

    _frame_client: framer.Client
    _retriever: Retriever
    _creator: Writer

    def __init__(
        self,
        frame_client: framer.Client,
        retriever: Retriever,
        creator: Writer,
    ):
        self._frame_client = frame_client
        self._retriever = retriever
        self._creator = creator

    def delete(self, channels: Params) -> None:
        """Deletes on or more channels from the cluster"""
        self._creator.delete(channels)

    @overload
    def create(
        self,
        *,
        data_type: CrudeDataType = DataType.UNKNOWN,
        name: str = "",
        index: Key = 0,
        is_index: bool = False,
        leaseholder: int = 0,
        virtual: bool | None = None,
        expression: str | None = None,
        operations: list[Operation] | None = None,
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
        leaseholder: int = 0,
        virtual: bool | None = None,
        expression: str = "",
        operations: list[Operation] | None = None,
        retrieve_if_name_exists: bool = False,
    ) -> Channel | list[Channel]:
        """Creates new channel(s) in the Synnax cluster.

        Overload 1:
        :param data_type: The data type of the samples in the channel. For example, `"float32"`.
        :param name: A name for the channel.
        :param is_index: Boolean indicating whether the channel is an index. Index
        channels should have a data type of synnax.TIMESTAMP.
        :param index: The key of the channel that indexes this channel.
        :param leaseholder: The node that holds the lease for this channel. If you don't know
        what this is, leave it at the default value of 0.
        :param virtual: Boolean indicating whether the channel is virtual. Virtual
        channels do not store any data, and are used for streaming purposes only.
        :param expression: An optional expression that defines the channel as a
        calculation of another channel. If this is set, the channel will be
        automatically configured as virtual.
        set if expression is not an empty string. If expression is not an empty string,
        this should have at least one channel.
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

        if channels is None:
            if is_index and data_type == DataType.UNKNOWN:
                data_type = DataType.TIMESTAMP
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
                )
            ]
        elif isinstance(channels, Channel):
            _channels = [channels.to_payload()]
        else:
            _channels = [c.to_payload() for c in channels]

        created = list()
        if retrieve_if_name_exists:
            created = self.__sugar(
                self._retriever.retrieve([ch.name for ch in _channels])
            )
            _channels = [
                c for c in _channels if c.name not in [ch.name for ch in created]
            ]

        created.extend(self.__sugar(self._creator.create(_channels)))
        return created if isinstance(channels, list) else created[0]

    @overload
    def retrieve(self, channel: Key | str) -> Channel: ...

    @overload
    def retrieve(
        self,
        channel: list[Key] | tuple[Key] | list[str] | tuple[str],
    ) -> list[Channel]: ...

    def retrieve(self, channel: Params) -> Channel | list[Channel]:
        """Retrieves a channel or set of channels from the cluster.

        Overload 1:

        :param channel: The key or name of the channel to retrieve. If this is
        :returns: The associated channel.
        :raises QueryError: If the channel is not found.

        Overload 2 + 3:
        :param channel: The list of keys or the list of names for the channels to retrieve.
        :returns: The retrieved channels.
        """
        normal = normalize_params(channel)
        res = self._retriever.retrieve(channel)
        sug = self.__sugar(res)
        if not normal.single:
            return sug
        if len(res) == 1:
            return sug[0]
        if len(res) > 1:
            raise _multiple_results_error(channel, res)
        raise NotFoundError(f"Channel matching '{channel}' not found.")

    @overload
    def rename(self, keys: Key, names: str) -> None:
        """Renames a channel in the cluster.

        :param keys: The key of the channel to rename.
        :param names: The new name for the channel.
        :returns: None.
        """
        ...

    @overload
    def rename(
        self, keys: list[Key] | tuple[Key], names: list[str] | tuple[str]
    ) -> None:
        """Renames one or more channels in the cluster.

        :param keys: The keys of the channels to rename.
        :param names: The new names for the channels.
        :returns: None.
        """

    def rename(
        self, keys: list[Key] | tuple[Key], names: list[str] | tuple[str]
    ) -> None:
        """Renames one or more channels in the cluster.

        :param keys: The keys of the channels to rename.
        :param names: The new names for the channels.
        :returns: None.
        """
        self._creator.rename(normalize(keys), normalize(names))

    def __sugar(self, channels: list[Payload]) -> list[Channel]:
        return [
            Channel(**c.model_dump(), _frame_client=self._frame_client)
            for c in channels
        ]


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
        msg += f"{str(results[i])}, "

    if len(results) > 5:
        msg += f"and {len(results) - 5} more."

    return MultipleFoundError(msg)
