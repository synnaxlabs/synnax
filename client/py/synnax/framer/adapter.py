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
    ChannelName,
    ChannelParams,
    ChannelPayload,
    normalize_channel_params,
)
from synnax.channel.retrieve import ChannelRetriever, retrieve_required
from synnax.exceptions import ValidationError
from synnax.framer.frame import Frame, CrudeFrame
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
            self.keys = normal.channels
            return

        fetched = self.retriever.retrieve(normal.channels)
        self.__adapter = dict[int, str]()
        for name in normal.channels:
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
    __err_on_extra_chans: bool

    def __init__(self, retriever: ChannelRetriever, err_on_extra_chans: bool = True):
        self.retriever = retriever
        self.__adapter = None
        self.__keys = None
        self.__err_on_extra_chans = err_on_extra_chans

    def update(self, channels: ChannelParams):
        results = retrieve_required(self.retriever, channels)
        self.__adapter = {ch.name: ch.key for ch in results}
        self.__keys = [ch.key for ch in results]

    def adapt_dict_keys(
        self, data: dict[ChannelPayload | ChannelKey | ChannelName, any]
    ) -> dict[ChannelKey, any]:
        out = dict()
        for k in data.keys():
            out[self.__adapt_to_key(k)] = data[k]
        return out

    @property
    def keys(self):
        return self.__keys

    def __adapt_to_key(
        self, ch: ChannelPayload | ChannelKey | ChannelName
    ) -> ChannelKey:
        if isinstance(ch, ChannelKey):
            return ch
        if isinstance(ch, ChannelPayload):
            return ch.key
        # If it's not a payload or key already, it has to be a name,
        # which means we need to resolve the key from a remote source
        # (either cache or server)
        return self.__adapt_ch(ch).key

    def __adapt_ch(
        self, ch: ChannelKey | ChannelName | ChannelPayload
    ) -> ChannelPayload:
        if isinstance(ch, (ChannelKey, ChannelName)):
            return self.retriever.retrieve_one(ch)
        return ch

    def adapt(
        self,
        channels_or_data: ChannelPayload
        | list[ChannelPayload]
        | ChannelParams
        | CrudeFrame,
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ) -> Frame:
        if isinstance(channels_or_data, (ChannelName, ChannelKey, ChannelPayload)):
            if series is None:
                raise ValidationError(
                    f"""
                Received a single channel {'name' if isinstance(channels_or_data, ChannelName) else 'key'}
                but no data.
                """
                )
            if isinstance(series, list) and len(series) > 1:
                first = series[0]
                if not isinstance(first, (float, int)):
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
            cols = channels_or_data.channels if is_frame else channels_or_data.columns
            if self.__adapter is None:
                return channels_or_data
            channels = list()
            series = list()
            for col in cols:
                try:
                    channels.append(
                        self.__adapter[col] if isinstance(col, ChannelName) else col
                    )
                    series.append(Series(channels_or_data[col]))
                except KeyError as e:
                    if self.__err_on_extra_chans:
                        raise ValidationError(
                            f"Channel {e} was not provided in the list of "
                            f"channels when the writer was opened."
                        )
            return Frame(channels=channels, series=series)

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
