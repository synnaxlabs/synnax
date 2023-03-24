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
import pandas as pd

from synnax.channel.retrieve import ChannelRetriever
from synnax.framer.payload import NumpyFrame
from synnax.telem import NumpyArray, TimeRange, UnparsedTimeStamp
from synnax.transport import Transport
from synnax.framer.iterator import AUTO_SPAN, NumpyIterator
from synnax.framer.writer import DataFrameWriter


class FrameClient:
    """SegmentClient provides interfaces for reading and writing segmented
    telemetry from a Synnax Cluster. SegmentClient should not be instantiated
    directly, but rather used through the synnax.Synnax class.
    """

    _transport: Transport
    _channels: ChannelRetriever

    def __init__(self, transport: Transport, registry: ChannelRetriever):
        self._transport = transport
        self._channels = registry

    def new_writer(
        self,
        start: UnparsedTimeStamp,
        *keys_or_names: str | list[str],
        strict: bool = False,
        suppress_warnings: bool = False,
    ) -> DataFrameWriter:
        """Opens a new writer on the given channels.

        :param keys: A list of channel keys that the writer will write to. A writer
        cannot write to keys not provided in this list. See the NumpyWriter documentation
        for more.
        :returns: A NumpyWriter that can be used to write telemetry to the given channels.
        """
        return DataFrameWriter(
            self._transport.stream,
            self._channels,
            start,
            *keys_or_names,
            strict=strict,
            suppress_warnings=suppress_warnings,
        )

    def new_iterator(
        self,
        tr: TimeRange,
        *keys_or_names: str | list[str],
        aggregate: bool = False,
    ) -> NumpyIterator:
        """Opens a new iterator over the given channels within the provided time range.

        :param keys: A list of channel keys to iterator over.
        :param tr: A time range to iterate over.
        :param aggregate:  Whether to accumulate iteration results or reset them on every
        iterator method call.
        :returns: A NumpyIterator over the given channels within the provided time
        range. See the NumpyIterator documentation for more.
        """
        return NumpyIterator(
            self._transport.stream,
            self._channels,
            tr,
            *keys_or_names,
            aggregate=aggregate,
        )

    def write(
        self,
        start: UnparsedTimeStamp,
        data: ndarray,
        key_or_name: str,
        strict: bool = False,
    ):
        """Writes telemetry to the given channel starting at the given timestamp.

        :param to: The key of the channel to write to.
        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        with self.new_writer(start, key_or_name, strict=strict) as w:
            w.write(pd.DataFrame({key_or_name: data}))
            w.commit()

    @overload
    def read(
        self,
        start: UnparsedTimeStamp,
        end: UnparsedTimeStamp,
        *keys_or_name: str,
    ) -> NumpyFrame:
        ...

    @overload
    def read(
        self,
        start: UnparsedTimeStamp,
        end: UnparsedTimeStamp,
        key_or_name: str,
    ) -> tuple[ndarray, TimeRange]:
        ...


    def read(
        self,
        start: UnparsedTimeStamp,
        end: UnparsedTimeStamp,
        key_or_name: str,
        *keys_or_name: str,
    ) -> tuple[ndarray, TimeRange] | NumpyFrame:
        """Reads telemetry from the channel between the two timestamps.

        :param start: The starting timestamp of the range to read from.
        :param end: The ending timestamp of the range to read from.
        :param key_or_name: The key or name of the channel to read from.
        :returns: A tuple where the first item is a numpy array containing the telemetry
        and the second item is the time range occupied by that array.
        :raises ContiguityError: If the telemetry between start and end is non-contiguous.
        """
        arr = self.read_array(start, end, key_or_name)
        assert arr.time_range is not None
        return arr.data, arr.time_range

    def read_array(
        self,
        start: UnparsedTimeStamp,
        end: UnparsedTimeStamp,
        key_or_name: str,
    ) -> NumpyArray:
        """Reads a Segment from the given channel between the two timestamps.

        :param start: The starting timestamp of the range to read from.
        :param end: The ending timestamp of the range to read from.
        :param key_or_name: The key or name of the channel to read from.
        :returns: A NumpySegment containing the read telemetry.
        :raises ContiguityError: If the telemetry between start and end is non-contiguous.
        """
        with self.new_iterator(TimeRange(start, end), key_or_name, aggregate=True) as i:
            # exhaust the iterator
            [_ for _ in i]
            return i.value[key_or_name]
