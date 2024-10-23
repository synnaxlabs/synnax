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
from alamos import NOOP, Instrumentation
from freighter import AsyncStreamClient, StreamClient, UnaryClient

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
from synnax.exceptions import QueryError
from synnax.framer.adapter import ReadFrameAdapter, WriteFrameAdapter
from synnax.framer.frame import Frame, CrudeFrame
from synnax.framer.iterator import Iterator
from synnax.framer.streamer import AsyncStreamer, Streamer
from synnax.framer.writer import Writer, WriterMode, CrudeWriterMode
from synnax.framer.deleter import Deleter
from synnax.ontology import ID
from synnax.telem import (
    CrudeTimeStamp,
    TimeRange,
    TimeSpan,
    CrudeSeries,
    MultiSeries,
)
from synnax.telem.control import Authority, CrudeAuthority

ontology_type = ID(type="framer")


class Client:
    """FramerClient provides interfaces for reading and writing segmented
    telemetry from a Synnax Cluster. SegmentClient should not be instantiated
    directly, but rather used through the synnax.Synnax class.
    """

    __stream_client: StreamClient
    __async_client: AsyncStreamClient
    __unary_client: UnaryClient
    __channels: ChannelRetriever
    __deleter: Deleter
    instrumentation: Instrumentation

    def __init__(
        self,
        stream_client: StreamClient,
        async_client: AsyncStreamClient,
        unary_client: UnaryClient,
        retriever: ChannelRetriever,
        deleter: Deleter,
        instrumentation: Instrumentation = NOOP,
    ):
        self.__stream_client = stream_client
        self.__async_client = async_client
        self.__unary_client = unary_client
        self.__channels = retriever
        self.__deleter = deleter
        self.instrumentation = instrumentation

    def open_writer(
        self,
        start: CrudeTimeStamp,
        channels: ChannelParams,
        authorities: CrudeAuthority | list[CrudeAuthority] = Authority.ABSOLUTE,
        *,
        name: str = "",
        strict: bool = False,
        suppress_warnings: bool = False,
        mode: CrudeWriterMode = WriterMode.PERSIST_STREAM,
        err_on_unauthorized: bool = False,
        enable_auto_commit: bool = False,
        auto_index_persist_interval: TimeSpan = 1 * TimeSpan.SECOND,
        err_on_extra_chans: bool = True,
    ) -> Writer:
        """Opens a new writer on the given channels.

        :param start: Sets the starting timestamp for the first sample in the writer. If
        this timestamp overlaps with existing data for ANY of the provided channels,
        the writer will fail to open.
        :param channels: The channels to write to. This can be a single channel name,
        a list of channel names, a single channel key, or a list of channel keys.
        :param authorities: The control authority to set for each channel on the writer.
        Defaults to absolute authority. If not working with concurrent control,
        it's best to leave this as the default.
        :param name: The name of the writer used in control subject.
        :param strict: Sets whether the writer will fail to write if the data for a
        particular channel does not exactly match this data type. When False,
        the default, the writer will automatically convert the data to the correct
        type if possible.
        :param suppress_warnings: Suppress various print warnings that may be emitted
        by the writer.
        :param mode: sets the persistence and streaming mode of the writer. The default
        mode is WriterModePersistStream. See the WriterMode documentation for more.
        :param err_on_unauthorized: sets whether the writer should return an error if
        it attempts to write to a channel it does not have control over.
        :param enable_auto_commit: determines whether the writer will automatically
        commit. If enable_auto_commit is true, then the writer will commit after each
        write, and will flush that commit to index after the specified
        auto_index_persist_interval.
        :param auto_index_persist_interval: interval at which commits to the index will
        be persisted. To persist every commit to guarantee minimal loss of data, set
        auto_index_persist_interval to AlwaysAutoIndexPersist.
        """
        adapter = WriteFrameAdapter(self.__channels, err_on_extra_chans)
        adapter.update(channels)
        return Writer(
            start=start,
            adapter=adapter,
            client=self.__stream_client,
            strict=strict,
            suppress_warnings=suppress_warnings,
            authorities=authorities,
            name=name,
            mode=mode,
            err_on_unauthorized=err_on_unauthorized,
            enable_auto_commit=enable_auto_commit,
            auto_index_persist_interval=auto_index_persist_interval,
        )

    def open_iterator(
        self,
        tr: TimeRange,
        channels: ChannelParams,
        chunk_size: int = 1e5,
    ) -> Iterator:
        """Opens a new iterator over the given channels within the provided time range.

        :param channels: A list of channel keys to iterator over.
        :param tr: A time range to iterate over.
        :param chunk_size: The number of samples to read in a chunk with AutoSpan. Defaults to 100000
        :returns: An Iterator over the given channels within the provided time
        range. See the Iterator documentation for more.
        """
        adapter = ReadFrameAdapter(self.__channels)
        adapter.update(channels)
        return Iterator(
            tr=tr,
            adapter=adapter,
            client=self.__stream_client,
            chunk_size=chunk_size,
            instrumentation=self.instrumentation,
        )

    @overload
    def write(
        self,
        start: CrudeTimeStamp,
        frame: CrudeFrame,
        strict: bool = False,
    ):
        ...

    @overload
    def write(
        self,
        start: CrudeTimeStamp,
        channel: ChannelKey | ChannelName | ChannelPayload,
        data: CrudeSeries,
        strict: bool = False,
    ):
        """Writes telemetry to the given channel starting at the given timestamp.

        :param channel: The key of the channel to write to.
        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        ...

    @overload
    def write(
        self,
        start: CrudeTimeStamp,
        channel: ChannelKeys | ChannelNames | list[ChannelPayload],
        series: list[CrudeSeries],
        strict: bool = False,
    ):
        ...

    def write(
        self,
        start: CrudeTimeStamp,
        channels: ChannelParams | ChannelPayload | list[ChannelPayload] | CrudeFrame,
        series: CrudeSeries | list[CrudeSeries] | None = None,
        strict: bool = False,
    ):
        parsed_channels = list()
        if isinstance(channels, (list, ChannelKey, ChannelPayload, ChannelName)):
            parsed_channels = channels
        elif isinstance(channels, dict):
            parsed_channels = list(channels.keys())
        elif isinstance(channels, Frame):
            parsed_channels = channels.channels
        elif isinstance(channels, pd.DataFrame):
            parsed_channels = list(channels.columns)
        with self.open_writer(
            start=start,
            channels=parsed_channels,
            strict=strict,
            mode=WriterMode.PERSIST,
            err_on_unauthorized=True,
            enable_auto_commit=True,
            auto_index_persist_interval=TimeSpan.MAX,
        ) as w:
            w.write(channels, series)

    @overload
    def read(
        self,
        tr: TimeRange,
        channels: ChannelKeys | ChannelNames,
    ) -> Frame:
        ...

    @overload
    def read(
        self,
        tr: TimeRange,
        channels: ChannelKey | ChannelName,
    ) -> MultiSeries:
        ...

    def read(
        self,
        tr: TimeRange,
        channels: ChannelParams,
    ) -> MultiSeries | Frame:
        """
        Reads telemetry from the channel between the two timestamps.

        :param tr: The time range to read from.
        :param channels: The key or name of the channel to read from.

        :returns: A tuple where the first item is a numpy array containing the telemetry
        and the second item is the time range occupied by that array.
        """
        normal = normalize_channel_params(channels)
        frame = self._read_frame(tr, channels)
        if len(normal.channels) > 1:
            return frame
        series = frame.get(normal.channels[0], None)
        if series is None:
            raise QueryError(
                f"""No data found for channel {normal.channels[0]} between {tr}"""
            )
        return series

    def open_streamer(
        self, channels: ChannelParams, downsample_factor: int = 1
    ) -> Streamer:
        """Opens a new streamer on the given channels. The streamer will immediately
        being receiving frames of data from the given channels.

        :param channels: The channels to stream from. This can be a single channel name,
        a list of channel names, a single channel key, or a list of channel keys.

        :param downsample_factor: The downsample factor to use for the streamer.
        """
        adapter = ReadFrameAdapter(self.__channels)
        adapter.update(channels)
        return Streamer(
            adapter=adapter,
            client=self.__stream_client,
            downsample_factor=downsample_factor,
        )

    async def open_async_streamer(
        self, channels: ChannelParams, downsample_factor: int = 1
    ) -> AsyncStreamer:
        adapter = ReadFrameAdapter(self.__channels)
        adapter.update(channels)
        s = AsyncStreamer(
            adapter=adapter,
            client=self.__async_client,
            downsample_factor=downsample_factor,
        )
        await s._open()
        return s

    def delete(self, channels: ChannelParams, tr: TimeRange) -> None:
        """
        delete deletes data in the specified channels in the specified time range.
        Note that the time range is start-inclusive and end-exclusive.
        Also note that deleting all data in a channel does not delete the channel; to
        delete a channel, use client.channels.delete().
        :param channels: channels to delete data from.
        :param tr: time range to delete data from.
        """
        self.__deleter.delete(channels, tr)

    def _read_frame(
        self,
        tr: TimeRange,
        channels: ChannelParams,
    ) -> Frame:
        aggregate = Frame()
        with self.open_iterator(tr, channels) as it:
            for fr in it:
                aggregate.append(fr)
        return aggregate
