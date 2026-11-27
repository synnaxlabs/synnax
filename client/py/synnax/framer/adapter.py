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
from typing import Any

import synnax.channel.payload as channel
from synnax.channel.retrieve import (
    Retriever,
    retrieve_required as retrieve_required_channel,
)
from synnax.exceptions import PathError, ValidationError
from synnax.framer.codec import Codec
from synnax.framer.frame import CrudeFrame, Frame, FramePayload, K, is_dataframe
from synnax.telem import CrudeSeries, DataType, LimitReduction, Reduction, Series
from x.telem import is_crude_series


class ReadFrameAdapter:
    _aligned: bool
    _limited: bool
    _timestamps_included: bool
    _by_name: bool
    _names: dict[channel.Key, str]
    # _paired holds the channels whose samples pair with the time of their clock.
    _paired: set[channel.Key]
    _records: dict[channel.Key, channel.Payload]
    _rank: dict[channel.Key | str, int]
    retriever: Retriever
    keys: list[channel.Key]
    codec: Codec

    def __init__(
        self,
        retriever: Retriever,
        *,
        reduction: Reduction | None = None,
        timestamps_included: bool = False,
    ):
        """
        :param timestamps_included: Also reads the index channel of each requested
            channel that the request leaves out. Frames hold the index series after
            the requested ones.
        """
        self.retriever = retriever
        # A min_max reduction picks the extremes of each channel on its own, so a data
        # sample does not keep the time of the index sample at its position.
        self._aligned = not (
            isinstance(reduction, LimitReduction) and reduction.aggregation == "min_max"
        )
        self._limited = isinstance(reduction, LimitReduction)
        self._timestamps_included = timestamps_included
        self._by_name = False
        self._names = {}
        self._paired = set()
        self._records = {}
        self._rank = {}
        self.keys = []
        self.codec = Codec()

    def update(self, channels: channel.Params) -> None:
        normal = channel.normalize_params(channels)
        fetched = retrieve_required_channel(self.retriever, normal.channels)
        if self._timestamps_included:
            fetched += self._retrieve_missing_indexes(fetched)
        new_keys = [ch.key for ch in fetched]
        # The Core counts every update that carries keys, and both sides must agree on
        # the sequence number of each codec state.
        if len(new_keys) > 0:
            self.codec.update(
                keys=new_keys, data_types=[ch.data_type for ch in fetched]
            )
        self._names = {ch.key: ch.name for ch in fetched}
        self._records = {ch.key: ch for ch in fetched}
        self._paired = {ch.key for ch in fetched if self._time_indexed(ch)}
        self._by_name = isinstance(normal, channel.NormalizedNameResult)
        if isinstance(normal, channel.NormalizedKeyResult):
            self.keys = list(normal.channels)
        else:
            keys_by_name = {name: key for key, name in self._names.items()}
            self.keys = [keys_by_name[name] for name in normal.channels]
        self.keys += [ch.key for ch in fetched if ch.key not in self.keys]
        self._rank = {}
        for i, key in enumerate(self.keys):
            self._rank.setdefault(key, i)
            self._rank.setdefault(self._names[key], i)

    def _time_indexed(self, ch: channel.Payload) -> bool:
        # A limit reduces the index but keeps every sample of a non-numeric channel.
        return self._aligned and not (self._limited and not ch.data_type.has_np)

    def _retrieve_missing_indexes(
        self, fetched: list[channel.Payload]
    ) -> list[channel.Payload]:
        present = {ch.key for ch in fetched}
        missing = {
            ch.index: None
            for ch in fetched
            if self._time_indexed(ch) and ch.index != 0 and ch.index not in present
        }
        if len(missing) == 0:
            return []
        return self.retriever.retrieve(list(missing))

    def adapt(self, fr: Frame) -> Frame:
        """Sorts the frame's series into the order of the requested channels and
        records the channel names in the frame, the index of each channel (the clock of
        its series), and the time series of each clock that a kept series names. When
        the channels were requested by name, the frame is keyed by name.
        """
        # The Core can send channels the client has not adapted yet, so purge them.
        items = sorted(
            ((k, s) for k, s in fr.items() if k in self._rank),
            key=lambda item: self._rank[item[0]],
        )
        named = {s.clock for _, s in items if s.clock != 0}
        times: dict[int, list[Series]] = {}
        for k, s in fr.items():
            if isinstance(k, channel.Key) and s.clock == k and k in named:
                times.setdefault(k, []).append(s)
        hidden = self._hidden(items)
        items = [(k, s) for k, s in items if not (s.clock == k and k in hidden)]
        fr.channels = [
            self._names[k] if self._by_name and isinstance(k, channel.Key) else k
            for k, _ in items
        ]
        fr.series = [s for _, s in items]
        fr.names.update(self._names)
        indexes: dict[channel.Key, int] = {}
        for k, s in items:
            if not isinstance(k, channel.Key) or k not in self._paired or s.clock == 0:
                continue
            held = indexes.setdefault(k, s.clock)
            if held != s.clock:
                raise ValueError(
                    f"channel {k} has series on clocks {held} and {s.clock}, so its "
                    "samples do not pair with one index"
                )
        fr.indexes.update(indexes)
        fr.times = times
        return fr

    def _hidden(self, items: list[tuple[channel.Key | str, Series]]) -> set[int]:
        """Returns the requested index channels whose stamps in the frame time only a
        calculation, as a calculation's frame holds its primary's stamps under that
        index's key. Those stamps are time, not samples of the index channel.
        """
        calculated: set[int] = set()
        recorded: set[int] = set()
        for k, s in items:
            record = self._records.get(k) if isinstance(k, channel.Key) else None
            if record is None or s.clock == k:
                continue
            if record.expression:
                calculated.add(s.clock)
            recorded.add(record.index)
        return calculated - recorded


# _TIME marks the series of a write's "time" key, which names no channel of the writer.
_TIME = channel.Key(0)

_TIME_AND_NO_CHANNELS = (
    'The write holds "time" and no channels. An empty series, time or data, makes '
    "nothing present"
)


class WriteFrameAdapter:
    """
    :ivar clocks: The index channel of each group of the writer's channels, from the
        writer's open response. A write's "time" goes to the clock of each group that
        has a channel in the write.
    """

    _adapter: dict[str, channel.Key] | None
    _channels: dict[channel.Key, channel.Payload]
    _keys: list[channel.Key]
    _err_on_extra_chans: bool
    _strict_data_types: bool
    _suppress_warnings: bool

    retriever: Retriever
    codec: Codec
    clocks: list[channel.Key]

    def __init__(
        self,
        *,
        retriever: Retriever,
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
        self.clocks = []

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
        if key is None and ch == "time":
            return _TIME
        return key if key is not None else self.retriever.retrieve_one(ch).key

    def _adapt_name(self, name: str) -> channel.Key:
        if self._adapter is not None and name in self._adapter:
            return self._adapter[name]
        if name == "time":
            return _TIME
        raise KeyError(name)

    def adapt(
        self,
        channels_or_data: (
            channel.Payload | list[channel.Payload] | channel.Params | CrudeFrame[K]
        ),
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ) -> Frame:
        frame = self._adapt(channels_or_data, series)
        time = self._take_time(frame)
        channels: list[channel.Payload] = []
        extra: set[channel.Key | str] = set()
        for col in frame.channels:
            ch = self._channels.get(col) if isinstance(col, channel.Key) else None
            if ch is None:
                extra.add(col)
            else:
                channels.append(ch)
        if extra:
            raise PathError("keys", ValidationError(f"frame has extra keys {extra}"))

        for i, (ch, series) in enumerate(zip(channels, frame.series, strict=True)):
            if series.data_type != ch.data_type:
                if self._strict_data_types:
                    raise PathError(
                        str(ch.key),
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
        if time is not None:
            self._place_time(frame, time)
        return frame

    def _take_time(self, frame: Frame) -> Series | None:
        """Removes the series of the write's "time" key from frame and returns it."""
        if _TIME not in frame.channels:
            return None
        i = frame.channels.index(_TIME)
        del frame.channels[i]
        return frame.series.pop(i)

    def _place_time(self, frame: Frame, time: Series) -> None:
        """Adds time to frame under the clock of each group that has a channel in it,
        and sets the clock of each series on it.
        """
        if len(frame.channels) == 0:
            raise ValidationError(_TIME_AND_NO_CHANNELS)
        if time.data_type not in (DataType.TIMESTAMP, DataType.INT64):
            raise ValidationError(f'"time" holds {time.data_type}, not timestamps')
        placed: list[channel.Key] = []
        for col, s in frame.items():
            if not isinstance(col, channel.Key):
                continue
            ch = self._channels[col]
            index = ch.index
            if ch.virtual or index == 0:
                continue
            if index not in self.clocks:
                raise ValidationError(
                    f"{ch.name} takes its time from the stamps that its index already "
                    'stores, so the write cannot hold "time" for it'
                )
            if index in frame.channels:
                raise ValidationError(
                    f'The write holds "time" and the index channel {index}, which '
                    "both give the time of one group"
                )
            s.clock = index
            if index not in placed:
                placed.append(index)
        if len(placed) == 0:
            raise ValidationError(
                'The write holds "time" and no channel that stores it'
            )
        for index in placed:
            frame.append(index, Series(time, data_type=DataType.TIMESTAMP, clock=index))

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
            if not is_crude_series(series):
                raise ValidationError(
                    f"""
                Received a single channel {kind}
                but multiple series.
                """
                )
            key = self._adapt_to_key(channels_or_data)
            return Frame([key], [Series(series)])

        if isinstance(channels_or_data, (list, tuple)):
            if series is None:
                raise ValidationError(
                    f"""
                Received {len(channels_or_data)} channels but no series.
                """
                )
            series_list: list[CrudeSeries] = (
                list(series) if isinstance(series, list) else [series]
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

        if isinstance(channels_or_data, Frame) or is_dataframe(channels_or_data):
            if isinstance(channels_or_data, Frame):
                cols: list[channel.Key | str] = channels_or_data.channels
                columns = channels_or_data.series
            else:
                cols = list(channels_or_data.columns)
                columns = [Series(channels_or_data[col]) for col in cols]
            if self._adapter is None:
                return (
                    Frame(channels_or_data)
                    if is_dataframe(channels_or_data)
                    else channels_or_data
                )
            adapted_channels: list[channel.Key] = []
            adapted_series: list[Series] = []
            for col, column in zip(cols, columns, strict=True):
                try:
                    adapted_channels.append(
                        self._adapt_name(col) if isinstance(col, str) else col
                    )
                    adapted_series.append(column)
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
