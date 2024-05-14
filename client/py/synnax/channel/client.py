#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from typing import overload

from numpy import ndarray
from pydantic import PrivateAttr

from synnax.channel.payload import (
    ChannelKey,
    ChannelKeys,
    ChannelName,
    ChannelNames,
    ChannelParams,
    ChannelPayload,
    normalize_channel_params,
)
from synnax.channel.retrieve import ChannelRetriever
from synnax.channel.writer import ChannelWriter
from synnax.exceptions import MultipleFoundError, NotFoundError, ValidationError
from synnax.framer.client import Client
from synnax.telem import (
    CrudeDataType,
    CrudeRate,
    CrudeTimeStamp,
    DataType,
    Rate,
    Series,
    TimeRange,
)


class Channel(ChannelPayload):
    """A channel is a logical collection of samples emitted by or representing the
    values of a single source. See https://docs.synnaxlabs.com/concepts/channels for an
    introduction to channels and how they work.
    """

    ___frame_client: Client | None = PrivateAttr(None)

    class Config:
        arbitrary_types_allowed = True

    def __init__(
        self,
        *,
        name: str,
        data_type: CrudeDataType,
        rate: CrudeRate = 0,
        is_index: bool = False,
        index: ChannelKey = 0,
        leaseholder: int = 0,
        key: ChannelKey = 0,
        _frame_client: Client | None = None,
    ) -> None:
        """Initializes a new Channel using the given parameters. It's important to note
        that this does not create the Channel in the cluster. To create the channel,
        call client.channels.create(channel).

        :param data_type: The data type of the samples in the channel e.g. np.int64
        :param rate: Rate sets the rate at which the channels values are written. If
        this parameter is non-zero, is_index must be false and index must be an empty
        string or unspecified.
        :param name: A human-readable name for the channel.
        :param key: Is auto-assigned by the cluster, and should not be set by the
        caller.
        :param is_index: Boolean indicating whether the channel is an index. Index
        channels should have ax data type of synnax.TIMESTAMP.
        :param index: The key of the channel that indexes this channel.
        :param leaseholder: The node that holds the lease for this channel. If you
        don't know what this is, leave it at the default value of 0.
        :param _frame_client: The backing client for reading and writing data to and
        from the channel. This is provided by the Synnax py during calls to
        .channels.create() and .channels.retrieve() and should not be set by the caller.
        """
        super().__init__(
            data_type=DataType(data_type),
            rate=Rate(rate),
            name=name,
            leaseholder=leaseholder,
            key=key,
            is_index=is_index,
            index=index,
        )
        self.___frame_client = _frame_client

    @overload
    def read(
        self,
        start_or_range: TimeRange,
    ) -> Series:
        ...

    @overload
    def read(
        self,
        start_or_range: CrudeTimeStamp,
        end: CrudeTimeStamp,
    ) -> Series:
        ...

    def read(
        self,
        start_or_range: CrudeTimeStamp | TimeRange,
        end: CrudeTimeStamp | None = None,
    ) -> Series:
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

    def write(self, start: CrudeTimeStamp, data: ndarray | Series) -> None:
        """Writes telemetry to the channel starting at the given timestamp.

        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        self.__frame_client.write(start, data, self.key)

    @property
    def __frame_client(self) -> Client:
        if self.___frame_client is None:
            raise ValidationError(
                "Cannot read from or write to channel that has not been created."
            )
        return self.___frame_client

    def __hash__(self) -> int:
        return hash(self.key)

    def __eq__(self, other) -> bool:
        return self.key == other.key

    def to_payload(self) -> ChannelPayload:
        return ChannelPayload(
            data_type=self.data_type,
            rate=self.rate,
            name=self.name,
            leaseholder=self.leaseholder,
            key=self.key,
            index=self.index,
            is_index=self.is_index,
        )


class ChannelClient:
    """The core py class for executing channel operations against a Synnax cluster."""

    _frame_client: Client
    _retriever: ChannelRetriever
    _creator: ChannelWriter

    def __init__(
        self,
        frame_client: Client,
        retriever: ChannelRetriever,
        creator: ChannelWriter,
    ):
        self._frame_client = frame_client
        self._retriever = retriever
        self._creator = creator

    def delete(self, channels: ChannelParams) -> None:
        """Deletes on or more channels from the cluster"""
        self._creator.delete(channels)

    @overload
    def create(
        self,
        *,
        data_type: CrudeDataType = DataType.UNKNOWN,
        name: ChannelName = "",
        rate: CrudeRate = Rate(0),
        index: ChannelKey = 0,
        is_index: bool = False,
        leaseholder: int = 0,
        retrieve_if_name_exists: bool = False,
    ) -> Channel:
        ...

    @overload
    def create(
        self, channels: Channel, *, retrieve_if_name_exists: bool = False
    ) -> Channel:
        ...

    @overload
    def create(
        self, channels: list[Channel], *, retrieve_if_name_exists: bool = False
    ) -> list[Channel]:
        ...

    def create(
        self,
        channels: Channel | list[Channel] | None = None,
        *,
        data_type: CrudeDataType = DataType.UNKNOWN,
        name: ChannelName = "",
        rate: CrudeRate = Rate(0),
        is_index: bool = False,
        index: ChannelKey = 0,
        leaseholder: int = 0,
        retrieve_if_name_exists: bool = False,
    ) -> Channel | list[Channel]:
        """Creates a new channel or set of channels in the cluster. Possible arguments
        are as follows:

        Overload 1:
        :param data_type: The data type of the samples in the channel e.g np.int64
        :param rate: Rate sets the rate at which the channels values are written. If this
        parameter is non-zero, is_index must be false and index must be an empty string or
        unspecified.
        :param name: A human-readable name for the channel.
        :param is_index: Boolean indicating whether the channel is an index. Index
        channels should have ax data type of synnax.TIMESTAMP.
        :param index: The key or channel that indexes this channel.
        :param leaseholder: The node that holds the lease for this channel. If you don't know
        what this is, leave it at the default value of 0.
        :returns: The created channel.

        Overload 2:

        :param channels: A single channel to create.
        :returns: The created channel.

        Overload 3:

        :param channels: A list of channels to create.
        :returns: The created channels.
        """

        if channels is None:
            _channels = [
                ChannelPayload(
                    name=name,
                    leaseholder=leaseholder,
                    rate=Rate(rate),
                    data_type=DataType(data_type),
                    index=index,
                    is_index=is_index,
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
    def retrieve(self, channel: ChannelKey | ChannelName) -> Channel:
        ...

    @overload
    def retrieve(
        self,
        channel: ChannelKeys | ChannelNames,
    ) -> list[Channel]:
        ...

    def retrieve(self, channel: ChannelParams) -> Channel | list[Channel]:
        """Retrieves a channel or set of channels from the cluster.

        Overload 1:

        :param key: The key of the channel to retrieve. If this is specified, the name
        parameter is ignored.
        :param name: The name of the channel to retrieve. If key is specified, this is
        ignored.
        Only one of key or name must be specified.
        :returns: The associated channel.
        :raises QueryError: If the channel is not found.

        Overload 2 + 3:
        :param keys: The keys of the channels to retrieve. If this is specified, the names
        parameter is ignored.
        :param names: The names of the channels to retrieve. If keys are specified, this is
        ignored.
        Only one of keys or names may be specified.
        :param leaseholder: The node that holds the lease for the channels to retrieve. If you
        don't know what this is, don't specify it.
        :param include_not_found: Boolean indicating whether or not to include the keys or
        names of the channels that were not found in the result.
        :returns: The retrieved channels if include_not_found is False, otherwise a tuple
        containing the retrieved channels and the keys or names of the channels that were
        not found.
        """
        normal = normalize_channel_params(channel)
        res = self._retriever.retrieve(channel)
        sug = self.__sugar(res)
        if not normal.single:
            return sug

        if len(res) == 1:
            return sug[0]

        if len(res) > 1:
            raise _multiple_results_error(channel, res)

        raise NotFoundError(f"Channel matching '{channel}' not found.")

    def __sugar(self, channels: list[ChannelPayload]) -> list[Channel]:
        return [Channel(**c.dict(), _frame_client=self._frame_client) for c in channels]


def _multiple_results_error(
    channel: ChannelParams,
    results: list[ChannelPayload],
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
