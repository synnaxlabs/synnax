#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from numpy import ndarray
import pandas as pd

from synnax.channel.retrieve import ChannelRetriever
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
        keys: list[str] | None = None,
        names: list[str] | None = None,
        strict: bool = False,
        suppress_warnings: bool = False,
    ) -> DataFrameWriter:
        """Opens a new writer on the given channels.

        :param keys: A list of channel keys that the writer will write to. A writer
        cannot write to keys not provided in this list. See the NumpyWriter documentation
        for more.
        :returns: A NumpyWriter that can be used to write telemetry to the given channels.
        """
        w = DataFrameWriter(
            client=self._transport.stream,
            channels=self._channels,
            strict=strict,
            suppress_warnings=suppress_warnings,
        )
        w.open(start, keys, names)
        return w

    def new_iterator(
        self,
        tr: TimeRange,
        keys: list[str] | None = None,
        names: list[str] | None = None,
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
        npi = NumpyIterator(
            transport=self._transport.stream,
            channels=self._channels,
            aggregate=aggregate,
        )
        npi.open(tr, keys, names)
        return npi

    def write(
        self,
        start: UnparsedTimeStamp,
        data: ndarray,
        key: str | None = None,
        name: str | None = None,
        strict: bool = False,
    ):
        """Writes telemetry to the given channel starting at the given timestamp.

        :param to: The key of the channel to write to.
        :param start: The starting timestamp of the first sample in data.
        :param data: The telemetry to write to the channel.
        :returns: None.
        """
        to = key if key else name
        w = self.new_writer(
            start=start,
            keys=[key] if key else None,
            names=[name] if name else None,
            strict=strict,
        )
        try:
            w.write(pd.DataFrame({to: data}))
            w.commit()
        finally:
            w.close()

    def read(
        self,
        start: UnparsedTimeStamp,
        end: UnparsedTimeStamp,
        key: str | None = None,
        name: str | None = None,
    ) -> tuple[ndarray, TimeRange]:
        """Reads telemetry from the channel between the two timestamps.

        :param from_: THe key of the channel to read from.
        :param start: The starting timestamp of the range to read from.
        :param end: The ending timestamp of the range to read from.
        :returns: A numpy array containing the retrieved telemetry.
        :raises ContiguityError: If the telemetry between start and end is non-contiguous.
        """
        arr = self.read_array(start, end, key, name)
        assert arr.time_range is not None
        return arr.data, arr.time_range

    def read_array(
        self,
        start: UnparsedTimeStamp,
        end: UnparsedTimeStamp,
        key: str | None = None,
        name: str | None = None,
    ) -> NumpyArray:
        """Reads a Segment from the given channel between the two timestamps.

        :param from_: The key of the channel to read from.
        :param start: The starting timestamp of the range to read from.
        :param end: The ending timestamp of the range to read from.
        :returns: A NumpySegment containing the read telemetry.
        :raises ContiguityError: If the telemetry between start and end is non-contiguous.
        """
        from_ = key if key else name
        i = self.new_iterator(
            tr=TimeRange(start, end),
            aggregate=True,
            keys=[key] if key else None,
            names=[name] if name else None,
        )
        try:
            i.seek_first()
            while i.next(AUTO_SPAN):
                pass
        finally:
            i.close()
        return i.value[from_]
