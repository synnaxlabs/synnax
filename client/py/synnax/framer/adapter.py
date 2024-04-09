#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pandas import DataFrame

from synnax.channel.payload import (
    ChannelKey,
    ChannelKeys,
    ChannelName,
    ChannelNames,
    ChannelParams,
    ChannelPayload,
    normalize_channel_params,
)
from synnax.channel.retrieve import ChannelRetriever, retrieve_required
from synnax.exceptions import QueryError, ValidationError
from synnax.framer.frame import Frame
from synnax.telem.series import CrudeSeries, Series


class ReadFrameAdapter:
    __adapter: dict[ChannelKey, ChannelName] | None
    retriever: ChannelRetriever
    keys: list[ChannelKey]

    def __init__(self, retriever: ChannelRetriever):
        self.retriever = retriever
        self.__adapter = None
        self.keys = list()

    def update(self, channels: ChannelParams):
        normal = normalize_channel_params(channels)
        if normal.variant == "keys":
            self.__adapter = None
            self.keys = normal.params
            return

        fetched = self.retriever.retrieve(normal.params)
        self.__adapter = dict[int, str]()
        for name in normal.params:
            ch = next((c for c in fetched if c.name == name), None)
            if ch is None:
                raise KeyError(f"Channel {name} not found.")
            self.__adapter[ch.key] = ch.name
        self.keys = list(self.__adapter.keys())

    def adapt(self, fr: Frame):
        if self.__adapter is None:
            return fr
        keys = [
            self.__adapter[k] if isinstance(k, ChannelKey) else k for k in fr.channels
        ]
        return Frame(channels=keys, series=fr.series)


class WriteFrameAdapter:
    __adapter: dict[ChannelName, ChannelKey] | None
    retriever: ChannelRetriever
    __keys: list[ChannelKey] | None

    def __init__(self, retriever: ChannelRetriever):
        self.retriever = retriever
        self.__adapter = None
        self.__keys = None

    def update(self, channels: ChannelParams):
        results = retrieve_required(self.retriever, channels)
        self.__adapter = {ch.name: ch.key for ch in results}
        self.__keys = [ch.key for ch in results]

    @property
    def keys(self):
        return self.__keys

    def __adapt_ch(
        self, ch: ChannelKey | ChannelName | ChannelPayload
    ) -> ChannelPayload:
        if not isinstance(ch, (ChannelKey, ChannelName)):
            return ch
        return retrieve_required(self.retriever, ch)[0]

    def adapt(
        self,
        channels_or_data: ChannelPayload
        | ChannelParams
        | Frame
        | dict[ChannelKey | ChannelName, CrudeSeries]
        | DataFrame,
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ) -> Frame:
        if isinstance(channels_or_data, (ChannelName, ChannelKey)):
            if series is None:
                raise ValidationError(
                    f"""
                Received a single channel {'name' if isinstance(channels_or_data, ChannelName) else 'key'}
                but no series.
                """
                )
            if isinstance(series, list) and len(list) > 1:
                raise ValidationError(
                    f"""
                Received a single channel {'name' if isinstance(channels_or_data, ChannelName) else 'key'}
                but multiple series.
                """
                )

            pld = self.__adapt_ch(channels_or_data)
            series = Series(data_type=pld.data_type, data=series)
            return Frame([pld.key], [series])

        if isinstance(channels_or_data, list):
            if series is None:
                raise ValidationError(
                    f"""
                Received {len(channels_or_data)} channels but no series.
                """
                )
            channels = list()
            o_series = list()
            for i, ch in enumerate(channels_or_data):
                pld = self.__adapt_ch(ch)
                if i >= len(series):
                    raise ValidationError(
                        f"""
                    Received {len(channels_or_data)} channels but only {len(series)} series.
                    """
                    )
                s = Series(data_type=pld.data_type, data=series[i])
                channels.append(pld.key)
                o_series.append(s)

            return Frame(channels, o_series)

        is_frame = isinstance(channels_or_data, Frame)
        is_df = isinstance(channels_or_data, DataFrame)
        if is_frame or is_df:
            if is_df:
                channels_or_data = Frame(channels_or_data)
            if self.__adapter is None:
                return channels_or_data
            try:
                channels = [
                    self.__adapter[col] if isinstance(col, ChannelName) else col
                    for col in channels_or_data.channels
                ]
            except KeyError as e:
                raise ValidationError(
                    f"Channel {e} was not provided in the list of "
                    f"channels when the writer was opened."
                )
            return Frame(channels=channels, series=channels_or_data.series)

        if isinstance(channels_or_data, dict):
            channels = list()
            series = list()
            for k, v in channels_or_data.items():
                pld = self.__adapt_ch(k)
                s = Series(data_type=pld.data_type, data=v)
                channels.append(pld.key)
                series.append(s)

            return Frame(channels, series)

        raise TypeError(
            f"""Cannot construct frame from {channels_or_data} and {series}"""
        )
