#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

import pandas as pd
from numpy import ndarray
from freighter import StreamClient

from alamos import trace, Instrumentation, NOOP
from synnax.channel.payload import ChannelParams, ChannelKey, ChannelName
from synnax.channel.retrieve import ChannelRetriever
from synnax.framer.iterator import NumpyIterator
from synnax.framer.payload import NumpyFrame
from synnax.framer.streamer import NumpyStreamer
from synnax.framer.writer import DataFrameWriter
from synnax.telem import TimeRange, UnparsedTimeStamp


class FrameClient:
    """SegmentClient provides interfaces for reading and writing segmented
    telemetry from a Synnax Cluster. SegmentClient should not be instantiated
    directly, but rather used through the synnax.Synnax class.
    """

    __client: StreamClient
    __channels: ChannelRetriever
    instrumentation: Instrumentation

    def __init__(
        self,
        client: StreamClient,
        registry: ChannelRetriever,
        instrumentation: Instrumentation = NOOP,
    ):
        self.__client = client
        self.__channels = registry
        self.instrumentation = instrumentation

    def new_writer(
        self,
        start: UnparsedTimeStamp,
        params: ChannelParams,
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
            self.__client,
            self.__channels,
            start,
            params,
            strict=strict,
            suppress_warnings=suppress_warnings,
        )

    def new_iterator(
        self,
        tr: TimeRange,
        params: ChannelParams,
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
            self.__client,
            self.__channels,
            tr,
            params,
            aggregate=aggregate,
        )

    def write(
        self,
        start: UnparsedTimeStamp,
        data: ndarray,
        to: ChannelKey | ChannelName,
        strict: bool = False,
    ):
        """Writes telemetry to the given channel starting at the given timestamp.

        :param to: The key of the channel to write to.
        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        with self.new_writer(start, to, strict=strict) as w:
            w.write(pd.DataFrame({to: data}))
            w.commit()

    @overload
    def read(
        self,
        start: UnparsedTimeStamp,
        end: UnparsedTimeStamp,
        key_or_name: str,
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
        params: ChannelParams,
    ) -> tuple[ndarray, TimeRange] | NumpyFrame:
        """Reads telemetry from the channel between the two timestamps.

        :param start: The starting timestamp of the range to read from.
        :param end: The ending timestamp of the range to read from.
        :param params: The key or name of the channel to read from.
        :returns: A tuple where the first item is a numpy array containing the telemetry
        and the second item is the time range occupied by that array.
        :raises ContiguityError: If the telemetry between start and end is non-contiguous.
        """
        frame = self.read_frame(start, end, params)
        if len(params) > 1:
            return frame
        arr = frame[params]
        assert arr.time_range is not None
        return arr.data, arr.time_range

    @trace("prod")
    def read_frame(
        self,
        start: UnparsedTimeStamp,
        end: UnparsedTimeStamp,
        params: ChannelParams,
    ) -> NumpyFrame:
        """Reads a Segment from the given channel between the two timestamps.

        :param start: The starting timestamp of the range to read from.
        :param end: The ending timestamp of the range to read from.
        :param key_or_name: The key or name of the channel to read from.
        :returns: A NumpySegment containing the read telemetry.
        :raises ContiguityError: If the telemetry between start and end is non-contiguous.
        """
        tr = TimeRange(start, end)
        with self.new_iterator(tr, params, aggregate=True) as i:
            # exhaust the iterator
            _ = [value for value in i]
            return i.value

    def stream(
        self,
        start: UnparsedTimeStamp,
        *keys_or_names: ChannelParams,
    ) -> NumpyStreamer:
        return NumpyStreamer(
            self.__client,
            self.__channels,
            start,
            keys_or_names,
        )
