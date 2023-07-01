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

from synnax.telem import (
    UnparsedDataType,
    UnparsedRate,
    DataType,
    Rate,
    TimeRange,
    Series,
    UnparsedTimeStamp,
)
from synnax.exceptions import ValidationError
from synnax.channel.payload import ChannelPayload, ChannelKey
from synnax.framer import FrameClient


class Channel(ChannelPayload):
    """Represents a Channel in a Synnax database."""

    ___frame_client: FrameClient | None = PrivateAttr(None)

    class Config:
        arbitrary_types_allowed = True

    def __init__(
        self,
        *,
        name: str,
        data_type: UnparsedDataType,
        rate: UnparsedRate = 0,
        is_index: bool = False,
        index: ChannelKey = 0,
        leaseholder: int = 0,
        key: ChannelKey = 0,
        _frame_client: FrameClient | None = None,
    ):
        """Initializes a new Channel using the given parameters. It's important to note
        that this does not create the Channel in the cluster. To create the channel,
        call .channels.create().

        :param data_type: The data type of the samples in the channel e.g. np.int64
        :param rate: Rate sets the rate at which the channels values are written. If
        this parameter is non-zero, is_index must be false and index must be an empty
        string or unspecified.
        :param name: A human-readable name for the channel.
        :param key: Is auto-assigned by the cluster, and should not be set by the
        caller.
        :param is_index: Boolean indicating whether the channel is an index. Index
        channels should have ax data type of synnax.TIMESTAMP.
        :param index: The key or channel that indexes this channel.
        :param leaseholder: The node that holds the lease for this channel. If you
        don't know what this is, leave it at the default value of 0.
        :param _frame_client: The backing py for reading and writing data to and
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
        start_or_range: UnparsedTimeStamp,
        end: UnparsedTimeStamp,
    ) -> Series:
        ...

    def read(
        self,
        start_or_range: UnparsedTimeStamp | TimeRange,
        end: UnparsedTimeStamp | None = None,
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

    def write(self, start: UnparsedTimeStamp, data: ndarray | Series):
        """Writes telemetry to the channel starting at the given timestamp.

        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        self.__frame_client.write(start, data, self.key)

    @property
    def __frame_client(self) -> FrameClient:
        if self.___frame_client is None:
            raise ValidationError(
                "Cannot read from or write to channel that has not been created."
            )
        return self.___frame_client

    def __hash__(self):
        return hash(self.key)

    def __eq__(self, other):
        return self.key == other.key

    def __str__(self):
        base = f"{self.name} ({self.data_type})"
        if self.rate != 0:
            base += f" @ {self.rate}Hz"
        return base

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
