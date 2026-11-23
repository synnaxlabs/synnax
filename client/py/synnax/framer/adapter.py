#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import unicodedata
import warnings
from collections.abc import Mapping
from typing import Any

import synnax.channel.payload as channel
from synnax.channel.retrieve import (
    Retriever,
    match,
    not_found,
    refuse_shared,
    resolve_open,
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
    _labeled: bool
    _names: dict[channel.Key, str]
    _indexes: dict[channel.Key, channel.Key]
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
        self._labeled = False
        self._names = {}
        self._indexes = {}
        self._rank = {}
        self.keys = []
        self.codec = Codec()

    def update(self, channels: channel.Params) -> None:
        resolved = resolve_open(self.retriever, channels)
        requested = resolved.channels
        fetched = list(requested)
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
        # A frame holds keys or names, never both, so a list that holds any key labels
        # by key. Otherwise the frame names each channel by the name or path that first
        # looked it up.
        lookups = resolved.lookups
        self._labeled = len(lookups) > 0 and all(isinstance(lk, str) for lk in lookups)
        if self._labeled:
            labels: dict[channel.Key, str] = {}
            for lookup, ch in zip(lookups, resolved.each, strict=True):
                if isinstance(lookup, str):
                    labels.setdefault(ch.key, lookup)
            self._names.update(labels)
        self._indexes = {
            ch.key: ch.key if ch.is_index else ch.index
            for ch in fetched
            if self._time_indexed(ch) and (ch.is_index or ch.index != 0)
        }
        self.keys = [ch.key for ch in requested]
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
        records the channel names and indexes in the frame. When every channel was
        requested by name or path, the frame is keyed by that text.
        """
        # The Core can send channels the client has not adapted yet, so purge them.
        items = sorted(
            ((k, s) for k, s in fr.items() if k in self._rank),
            key=lambda item: self._rank[item[0]],
        )
        fr.channels = [
            self._names[k] if self._labeled and isinstance(k, channel.Key) else k
            for k, _ in items
        ]
        fr.series = [s for _, s in items]
        fr.names.update(self._names)
        fr.indexes.update(self._indexes)
        return fr


class WriteFrameAdapter:
    _by_text: dict[str, list[channel.Payload]]
    _channels: dict[channel.Key, channel.Payload]
    _keys: list[channel.Key]
    _last_entries: tuple[list[Any], list[channel.Key]] | None
    """The entries of the last record or list and their checked channel keys, until an
    update. Only entries of held references count, since a path can move between
    records.
    """
    _err_on_extra_chans: bool
    _strict_data_types: bool
    _suppress_warnings: bool

    retriever: Retriever
    codec: Codec

    def __init__(
        self,
        *,
        retriever: Retriever,
        err_on_extra_chans: bool = True,
        strict_data_types: bool = False,
        suppress_warnings: bool = True,
    ):
        self.retriever = retriever
        self._by_text = {}
        self._channels = {}
        self._keys = []
        self._last_entries = None
        self._err_on_extra_chans = err_on_extra_chans
        self._strict_data_types = strict_data_types
        self._suppress_warnings = suppress_warnings
        self.codec = Codec()

    def update(self, channels: channel.Params) -> None:
        resolved = resolve_open(self.retriever, channels)
        results = resolved.channels
        self._channels = {ch.key: ch for ch in results}
        self._by_text = {}
        for ch in results:
            self._hold(ch.name, ch)
        for lookup, ch in zip(resolved.lookups, resolved.each, strict=True):
            if isinstance(lookup, str):
                self._hold(lookup, ch)
        self._last_entries = None
        self._keys = [ch.key for ch in results]
        self.codec.update(
            keys=self._keys,
            data_types=[ch.data_type for ch in results],
        )

    def adapt_dict_keys(
        self, data: dict[channel.Payload | channel.Key | str, Any]
    ) -> dict[channel.Key, Any]:
        """Returns data keyed by the key of each channel that its keys name.

        :raises NotFoundError: Naming every key that finds no channel.
        :raises ValidationError: If two keys name one channel.
        """
        keys = self._entry_keys(list(data))
        return dict(zip(keys, data.values(), strict=True))

    @property
    def keys(self) -> list[channel.Key]:
        return self._keys

    def _hold(self, text: str, ch: channel.Payload) -> None:
        held = self._by_text.setdefault(unicodedata.normalize("NFC", text), [])
        if all(c.key != ch.key for c in held):
            held.append(ch)

    def _held(self, text: str) -> channel.Payload | None:
        """Returns the held channel that text names, or None.

        :raises ValidationError: If the writer holds several channels with that name.
        """
        held = self._by_text.get(unicodedata.normalize("NFC", text))
        if held is None:
            return None
        if len(held) > 1:
            raise ValidationError(
                f'The writer holds {len(held)} channels named "{text}"'
            )
        return held[0]

    def _match_keys(
        self, refs: list[channel.Lookup], *, names_fetched: bool = True
    ) -> tuple[list[channel.Key | None], bool]:
        """Returns the key of each reference, or None for one that finds no channel,
        and whether every reference resolved with no request. A key is itself. A name
        or path answers from the held channels, by NFC name or the NFC path that opened
        them. The other paths, and the other names when names_fetched, go in one
        retrieve.
        """
        texts: dict[str, channel.Payload | None] = {}
        for r in refs:
            if isinstance(r, str) and r not in texts:
                texts[r] = self._held(r)
        unheld = [
            t
            for t, ch in texts.items()
            if ch is None and (names_fetched or channel.is_path(t))
        ]
        texts.update(zip(unheld, match(self.retriever, unheld), strict=True))
        keys: list[channel.Key | None] = []
        for r in refs:
            if isinstance(r, str):
                ch = texts[r]
                keys.append(None if ch is None else ch.key)
            else:
                keys.append(r)
        return keys, len(unheld) == 0

    def _resolve_keys(
        self, refs: list[channel.Lookup]
    ) -> tuple[list[channel.Key], bool]:
        """Returns the key of each reference, and whether every one resolved with no
        request, as _match_keys does.

        :raises NotFoundError: Naming every reference that finds no channel.
        """
        keys, held = self._match_keys(refs)
        missing = [r for r, key in zip(refs, keys, strict=True) if key is None]
        if len(missing) > 0:
            raise not_found(missing)
        return [key for key in keys if key is not None], held

    def _entry_keys(self, entries: list[Any]) -> list[channel.Key]:
        """Returns the key of each channel that the entries of a record or a list name.
        Entries equal to the last ones reuse their checked keys.

        :raises NotFoundError: Naming every entry that finds no channel.
        :raises ValidationError: If two entries name one channel.
        """
        last = self._last_entries
        if last is not None and last[0] == entries:
            return last[1]
        refs = channel.to_lookups(entries)
        keys, held = self._resolve_keys(refs)
        refuse_shared(refs, keys)
        self._last_entries = (entries, keys) if held else None
        return keys

    def adapt(
        self,
        channels_or_data: (
            channel.Payload | list[channel.Payload] | channel.Params | CrudeFrame[K]
        ),
        series: CrudeSeries | list[CrudeSeries] | None = None,
    ) -> Frame:
        frame = self._adapt(channels_or_data, series)
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
            if not is_crude_series(series):
                raise ValidationError(
                    f"""
                Received a single channel {kind}
                but multiple series.
                """
                )
            [key], _ = self._resolve_keys(channel.to_lookups(channels_or_data))
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
            if len(series_list) < len(channels_or_data):
                n_channels, n_series = len(channels_or_data), len(series_list)
                raise ValidationError(
                    f"""
                Received {n_channels} channels but only {n_series} series.
                """
                )
            keys = self._entry_keys(list(channels_or_data))
            return Frame(keys, [Series(series_list[i]) for i in range(len(keys))])

        if isinstance(channels_or_data, FramePayload):
            channels_or_data = Frame(channels_or_data)

        if isinstance(channels_or_data, Frame) or is_dataframe(channels_or_data):
            if isinstance(channels_or_data, Frame):
                cols: list[channel.Key | str] = channels_or_data.channels
            else:
                cols = list(channels_or_data.columns)
            adapted_refs: list[channel.Lookup] = []
            adapted_channels: list[channel.Key] = []
            adapted_series: list[Series] = []
            # The writer holds the name of each of its channels, so a name it does not
            # hold names no channel of the writer.
            matched, _ = self._match_keys(channel.to_lookups(cols), names_fetched=False)
            for col, found in zip(cols, matched, strict=True):
                if found is None or (
                    isinstance(col, str) and found not in self._channels
                ):
                    if self._err_on_extra_chans:
                        raise ValidationError(
                            f"Channel {col} was not provided in the list of "
                            f"channels when the writer was opened."
                        )
                    continue
                adapted_refs.append(col)
                adapted_channels.append(found)
                adapted_series.append(Series(channels_or_data[col]))
            refuse_shared(adapted_refs, adapted_channels)
            return Frame(channels=adapted_channels, series=adapted_series)

        if isinstance(channels_or_data, Mapping):
            keys = self._entry_keys(list(channels_or_data))
            return Frame(keys, [Series(v) for v in channels_or_data.values()])

        raise TypeError(
            f"""Cannot construct frame from {channels_or_data} and {series}"""
        )
