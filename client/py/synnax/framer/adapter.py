#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import warnings

from pandas import DataFrame

from synnax.channel.payload import (
    ChannelKey,
    ChannelName,
    ChannelParams,
    ChannelPayload,
    normalize_channel_params,
)
from synnax.channel.retrieve import ChannelRetriever, retrieve_required
from synnax.exceptions import PathError, ValidationError
from synnax.framer.codec import Codec
from synnax.framer.frame import CrudeFrame, Frame
from synnax.telem import DataType
from synnax.telem.series import CrudeSeries, Series


class ReadFrameAdapter:
    __adapter: dict[ChannelKey, ChannelName] | None
    retriever: ChannelRetriever
    keys: list[ChannelKey]
    codec: Codec

    def __init__(self, retriever: ChannelRetriever):
        self.retriever = retriever
        self.__adapter = None
        self.keys = list()
        self.codec = Codec()

    def update(self, channels: ChannelParams):
        normal = normalize_channel_params(channels)
        fetched = self.retriever.retrieve(normal.channels)
        self.codec.update(
            [ch.key for ch in fetched],
            [ch.data_type for ch in fetched],
        )

        if normal.variant == "keys":
            self.__adapter = None
            self.keys = normal.channels
            return

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
    _adapter: dict[ChannelName, ChannelKey] | None
    _keys: list[ChannelKey] | None
    _err_on_extra_chans: bool
    _strict_data_types: bool
    _suppress_warnings: bool

    retriever: ChannelRetriever
    codec: Codec

    def __init__(
        self,
        retriever: ChannelRetriever,
        err_on_extra_chans: bool = True,
        strict_data_types: bool = False,
        suppress_warnings: bool = True,
    ):
        self.retriever = retriever
        self._adapter = None
        self._keys = None
        self._err_on_extra_chans = err_on_extra_chans
        self._strict_data_types = strict_data_types
        self._suppress_warnings = suppress_warnings
        self.codec = Codec()

    def update(self, channels: ChannelParams):
        results = retrieve_required(self.retriever, channels)
        self._adapter = {ch.name: ch.key for ch in results}
        self._keys = [ch.key for ch in results]
        self.codec.update(
            self._keys,
            [ch.data_type for ch in results],
        )

    def adapt_dict_keys(
        self, data: dict[ChannelPayload | ChannelKey | ChannelName, any]
    ) -> dict[ChannelKey, any]:
        out = dict()
        for k in data.keys():
            out[self.__adapt_to_key(k)] = data[k]
        return out

    @property
    def keys(self):
        return self._keys

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
        channels_or_data: (
            ChannelPayload | list[ChannelPayload] | ChannelParams | CrudeFrame
        ),
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ):
        frame = self._adapt(channels_or_data, series)
        extra = set(frame.channels) - set(self.keys)
        if extra:
            raise PathError("keys", ValidationError(f"frame has extra keys {extra}"))

        for i, (col, series) in enumerate(frame.items()):
            ch = self.retriever.retrieve(col)[0]  # type: ignore
            if series.data_type != ch.data_type:
                if self._strict_data_types:
                    raise PathError(
                        str(col),
                        ValidationError(
                            f"Data type {ch.data_type} for channel {ch} does "
                            + f"not match series data type {series.data_type}.",
                        ),
                    )
                elif not self._suppress_warnings and not (
                    ch.data_type == DataType.TIMESTAMP
                    and series.data_type == DataType.INT64
                ):
                    warnings.warn(
                        f"""Series for channel {ch.name} has type {series.data_type} but
                        channel expects type {ch.data_type}. We can safely convert
                        between the two, but this can cause performance degradations
                        and is not recommended. To suppress this warning,
                        set suppress_warnings=True when constructing the writer. To
                        raise an error instead, set strict=True when constructing
                        the writer."""
                    )
                frame.series[i] = series.astype(ch.data_type)
        return frame

    def _adapt(
        self,
        channels_or_data: (
            ChannelPayload | list[ChannelPayload] | ChannelParams | CrudeFrame
        ),
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
                channels.append(pld.key)
                o_series.append(series[i])

            return Frame(channels, o_series)

        is_frame = isinstance(channels_or_data, Frame)
        is_df = isinstance(channels_or_data, DataFrame)
        if is_frame or is_df:
            cols = channels_or_data.channels if is_frame else channels_or_data.columns
            if self._adapter is None:
                return channels_or_data
            channels = list()
            series = list()
            for col in cols:
                try:
                    channels.append(
                        self._adapter[col] if isinstance(col, ChannelName) else col
                    )
                    series.append(Series(channels_or_data[col]))
                except KeyError as e:
                    if self._err_on_extra_chans:
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
                channels.append(pld.key)
                series.append(Series(v))

            return Frame(channels, series)

        raise TypeError(
            f"""Cannot construct frame from {channels_or_data} and {series}"""
        )
