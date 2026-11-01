#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import warnings
from collections.abc import Mapping
from typing import Any, cast

from pandas import DataFrame

import synnax.channel.payload as channel
from synnax.channel.retrieve import Retriever as ChannelRetriever
from synnax.channel.retrieve import retrieve_required as retrieve_required_channel
from synnax.exceptions import PathError, ValidationError
from synnax.framer.codec import Codec
from synnax.framer.frame import CrudeFrame, Frame, FramePayload, K
from synnax.telem import CrudeSeries, DataType, Series


class ReadFrameAdapter:
    _by_name: bool
    _names: dict[channel.Key, str]
    _rank: dict[channel.Key | str, int]
    retriever: ChannelRetriever
    keys: list[channel.Key]
    codec: Codec

    def __init__(self, retriever: ChannelRetriever):
        self.retriever = retriever
        self._by_name = False
        self._names = {}
        self._rank = {}
        self.keys = []
        self.codec = Codec()

    def update(self, channels: channel.Params) -> None:
        normal = channel.normalize_params(channels)
        fetched = self.retriever.retrieve(normal.channels)
        new_keys = [ch.key for ch in fetched]
        # The Core counts every update that carries keys, and both sides must agree on
        # the sequence number of each codec state.
        if len(new_keys) > 0:
            self.codec.update(
                keys=new_keys, data_types=[ch.data_type for ch in fetched]
            )
        self._names = {ch.key: ch.name for ch in fetched}
        self._by_name = isinstance(normal, channel.NormalizedNameResult)
        if isinstance(normal, channel.NormalizedKeyResult):
            self.keys = list(normal.channels)
        else:
            keys_by_name = {name: key for key, name in self._names.items()}
            for name in normal.channels:
                if name not in keys_by_name:
                    raise KeyError(f"Channel {name} not found.")
            self.keys = [keys_by_name[name] for name in normal.channels]
        self._rank = {}
        for i, key in enumerate(self.keys):
            self._rank.setdefault(key, i)
            if key in self._names:
                self._rank.setdefault(self._names[key], i)

    def adapt(self, fr: Frame) -> Frame:
        """Sorts the frame's series into the order of the requested channels and
        records the channel names in the frame. When the channels were requested by
        name, the frame is keyed by name.
        """
        # The Core can send channels the client has not adapted yet, so purge them.
        items = sorted(
            ((k, s) for k, s in fr.items() if k in self._rank),
            key=lambda item: self._rank[item[0]],
        )
        fr.channels = [
            self._names[k] if self._by_name and isinstance(k, channel.Key) else k
            for k, _ in items
        ]
        fr.series = [s for _, s in items]
        fr.names.update(self._names)
        return fr


class WriteFrameAdapter:
    _adapter: dict[str, channel.Key] | None
    _channels: dict[channel.Key, channel.Payload]
    _keys: list[channel.Key]
    _err_on_extra_chans: bool
    _strict_data_types: bool
    _suppress_warnings: bool

    retriever: ChannelRetriever
    codec: Codec

    def __init__(
        self,
        *,
        retriever: ChannelRetriever,
        err_on_extra_chans: bool = True,
        strict_data_types: bool = False,
        suppress_warnings: bool = True,
    ):
        self.retriever = retriever
        self._adapter = None
        self._channels = {}
        self._keys = []
        self._err_on_extra_chans = err_on_extra_chans
        self._strict_data_types = strict_data_types
        self._suppress_warnings = suppress_warnings
        self.codec = Codec()

    def update(self, channels: channel.Params) -> None:
        results = retrieve_required_channel(self.retriever, channels)
        self._adapter = {ch.name: ch.key for ch in results}
        self._channels = {ch.key: ch for ch in results}
        self._keys = [ch.key for ch in results]
        self.codec.update(
            keys=self._keys,
            data_types=[ch.data_type for ch in results],
        )

    def adapt_dict_keys(
        self, data: dict[channel.Payload | channel.Key | str, Any]
    ) -> dict[channel.Key, Any]:
        out = {}
        for k in data:
            out[self._adapt_to_key(k)] = data[k]
        return out

    @property
    def keys(self) -> list[channel.Key]:
        return self._keys

    def _adapt_to_key(self, ch: channel.Payload | channel.Key | str) -> channel.Key:
        if isinstance(ch, channel.Key):
            return ch
        if isinstance(ch, channel.Payload):
            return ch.key
        key = None if self._adapter is None else self._adapter.get(ch)
        return key if key is not None else self.retriever.retrieve_one(ch).key

    def adapt(
        self,
        channels_or_data: (
            channel.Payload | list[channel.Payload] | channel.Params | CrudeFrame[K]
        ),
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ) -> Frame:
        frame = self._adapt(channels_or_data, series)
        extra = set(frame.channels) - set(self.keys)
        if extra:
            raise PathError("keys", ValidationError(f"frame has extra keys {extra}"))

        for i, (col, series) in enumerate(frame.items()):
            ch = self._channels[cast(channel.Key, col)]
            if series.data_type != ch.data_type:
                if self._strict_data_types:
                    raise PathError(
                        str(col),
                        ValidationError(
                            f"Data type {ch.data_type} for channel {ch} does "
                            + f"not match series data type {series.data_type}.",
                        ),
                    )
                if not self._suppress_warnings and not (
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
                        the writer.""",
                        stacklevel=3,
                    )
                frame.series[i] = series.astype(ch.data_type)
        return frame

    def _adapt(
        self,
        channels_or_data: (
            channel.Payload | list[channel.Payload] | channel.Params | CrudeFrame[K]
        ),
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ) -> Frame:
        if isinstance(channels_or_data, (str, channel.Key, channel.Payload)):
            kind = "name" if isinstance(channels_or_data, str) else "key"
            if series is None:
                raise ValidationError(
                    f"""
                Received a single channel {kind}
                but no data.
                """
                )
            if isinstance(series, list) and len(series) > 1:
                first = series[0]
                if not isinstance(first, (float, int)):
                    raise ValidationError(
                        f"""
                    Received a single channel {kind}
                    but multiple series.
                    """
                    )

            key = self._adapt_to_key(channels_or_data)
            return Frame([key], [Series(cast(CrudeSeries, series))])

        if isinstance(channels_or_data, (list, tuple)):
            if series is None:
                raise ValidationError(
                    f"""
                Received {len(channels_or_data)} channels but no series.
                """
                )
            series_list: list[CrudeSeries] = (
                [series]
                if not isinstance(series, list)
                else cast(list[CrudeSeries], series)
            )
            channels: list[channel.Key] = []
            o_series: list[Series] = []
            for i, ch in enumerate(channels_or_data):
                key = self._adapt_to_key(ch)
                if i >= len(series_list):
                    n_channels, n_series = len(channels_or_data), len(series_list)
                    raise ValidationError(
                        f"""
                    Received {n_channels} channels but only {n_series} series.
                    """
                    )
                channels.append(key)
                o_series.append(Series(series_list[i]))

            return Frame(channels, o_series)

        if isinstance(channels_or_data, FramePayload):
            channels_or_data = Frame(channels_or_data)

        if isinstance(channels_or_data, (Frame, DataFrame)):
            if isinstance(channels_or_data, Frame):
                cols: list[channel.Key | str] = channels_or_data.channels
            else:
                cols = list(channels_or_data.columns)
            if self._adapter is None:
                return (
                    Frame(channels_or_data)
                    if isinstance(channels_or_data, DataFrame)
                    else channels_or_data
                )
            adapted_channels: list[channel.Key] = []
            adapted_series: list[Series] = []
            for col in cols:
                try:
                    adapted_channels.append(
                        self._adapter[col] if isinstance(col, str) else col
                    )
                    adapted_series.append(Series(channels_or_data[col]))
                except KeyError as e:
                    if self._err_on_extra_chans:
                        raise ValidationError(
                            f"Channel {e} was not provided in the list of "
                            f"channels when the writer was opened."
                        ) from e
            return Frame(channels=adapted_channels, series=adapted_series)

        if isinstance(channels_or_data, Mapping):
            dict_channels: list[channel.Key] = []
            dict_series: list[Series] = []
            for k, v in channels_or_data.items():
                dict_channels.append(self._adapt_to_key(k))
                dict_series.append(Series(v))

            return Frame(dict_channels, dict_series)

        raise TypeError(
            f"""Cannot construct frame from {channels_or_data} and {series}"""
        )
