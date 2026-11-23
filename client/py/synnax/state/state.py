#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import unicodedata
from collections.abc import Mapping

from synnax import channel
from synnax.framer import Frame
from synnax.telem import MultiSeries, SampleValue


class State:
    value: dict[channel.Key, MultiSeries]
    _keys: dict[str, channel.Key]

    def __init__(
        self,
        channels: list[channel.Payload],
        texts: Mapping[str, channel.Key] | None = None,
    ):
        """
        :param channels: The channels whose values the state holds.
        :param texts: The key of each other name or path that reads a channel, such as
            the path that a handler registered.
        """
        self._keys = {ch.name: ch.key for ch in channels}
        if texts is not None:
            for text, key in texts.items():
                self._keys[unicodedata.normalize("NFC", text)] = key
        self.value = {}

    def _key(self, ch: channel.Key | str) -> channel.Key:
        if isinstance(ch, str):
            return self._keys[unicodedata.normalize("NFC", ch)]
        return ch

    def update(self, frame: Frame) -> None:
        for ch in frame.channels:
            self.value[self._key(ch)] = frame[ch]

    def __getitem__(self, ch: channel.Key | str) -> MultiSeries:
        return self.value[self._key(ch)]

    def __getattr__(self, name: str) -> MultiSeries:
        try:
            return self[name]
        except KeyError as e:
            raise AttributeError(name) from e


class Latest:
    _state: State

    def __init__(self, state: State) -> None:
        self._state = state

    def __getitem__(self, ch: channel.Key | str) -> SampleValue:
        return self._state[ch][-1]

    def __getattr__(self, name: str) -> SampleValue:
        try:
            return self[name]
        except KeyError as e:
            raise AttributeError(name) from e
