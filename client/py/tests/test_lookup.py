#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import unicodedata
from typing import Any

import numpy as np
import pytest

import synnax as sy
from synnax.channel.types_gen import RetrieveRequest
from x.strings import random_name
from x.telem import seconds_linspace


def channel_name() -> str:
    return f"test_{random_name()}"


def virtual(client: sy.Synnax, name: str | None = None, **kw: Any) -> sy.Channel:
    return client.channels.create(
        name=channel_name() if name is None else name,
        data_type=sy.DataType.FLOAT64,
        virtual=True,
        **kw,
    )


def in_group(client: sy.Synnax, **kw: Any) -> tuple[sy.Channel, str]:
    """Creates a channel in a new folder and returns it with its path."""
    g = client.groups.create(channel_name(), scope="channel")
    ch = client.channels.create(name=channel_name(), group=g.key, **kw)
    return ch, f"{g.name}.{ch.name}"


def indexed(client: sy.Synnax) -> tuple[sy.Channel, sy.Channel]:
    index = client.channels.create(
        name=channel_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
    )
    data = client.channels.create(
        name=channel_name(), data_type=sy.DataType.FLOAT64, index=index.key
    )
    return index, data


@pytest.fixture
def requests(
    client: sy.Synnax, monkeypatch: pytest.MonkeyPatch
) -> list[RetrieveRequest]:
    """Records each channel retrieve request that the client's retriever sends."""
    sent: list[RetrieveRequest] = []
    retriever = client.channels._retriever
    send = retriever.filter

    def record(req: RetrieveRequest) -> list[sy.channel.Payload]:
        sent.append(req)
        return send(req)

    monkeypatch.setattr(retriever, "filter", record)
    return sent


@pytest.mark.channel
class TestChannelLookups:
    def test_retrieve_by_path(self, client: sy.Synnax) -> None:
        """Should read a string that holds a '.' as a path."""
        ch, path = in_group(client, data_type=sy.DataType.FLOAT64, virtual=True)
        assert client.channels.retrieve(path).key == ch.key

    def test_retrieve_a_mixed_list(self, client: sy.Synnax) -> None:
        """Should retrieve a list that mixes a key, a name, and a path."""
        by_key, by_name = virtual(client), virtual(client)
        by_path, path = in_group(client, data_type=sy.DataType.FLOAT64, virtual=True)
        lookups: list[int | str] = [by_key.key, by_name.name, path]
        got = client.channels.retrieve(lookups)
        assert sorted(c.key for c in got) == sorted(
            [by_key.key, by_name.key, by_path.key]
        )

    def test_retrieve_one_name_strictly(
        self, client: sy.Synnax, requests: list[RetrieveRequest]
    ) -> None:
        """Should send one name strictly and a list of names not strictly."""
        ch = virtual(client)
        client.channels.retrieve(ch.name)
        client.channels.retrieve([ch.name])
        assert [r.strict for r in requests] == [True, False]

    def test_refuse_a_key_past_the_uint32_range(self, client: sy.Synnax) -> None:
        """Should refuse an integer that does not fit a channel key."""
        with pytest.raises(sy.ValidationError, match="4294967296 is not a channel key"):
            client.channels.retrieve(2**32)

    def test_delete_by_path(self, client: sy.Synnax) -> None:
        """Should delete the channel at a path."""
        ch, path = in_group(client, data_type=sy.DataType.FLOAT64, virtual=True)
        client.channels.delete(path)
        assert client.channels.retrieve([ch.key]) == []

    def test_refuse_a_delete_of_an_unknown_path(self, client: sy.Synnax) -> None:
        """Should refuse a delete of a path that names no channel, naming the path."""
        path = f"{channel_name()}.{channel_name()}"
        with pytest.raises(
            sy.NotFoundError, match=f'channels not found matching \\["{path}"\\]'
        ):
            client.channels.delete([path])

    def test_delete_digit_text_as_a_name(self, client: sy.Synnax) -> None:
        """Should delete the channel named by digit text in a mixed list, not the
        channel with that key.
        """
        other, victim = virtual(client), virtual(client)
        named = virtual(client, name=str(victim.key))
        client.channels.delete([other.key, named.name])
        assert client.channels.retrieve(victim.key).key == victim.key
        assert client.channels.retrieve([other.key, named.key]) == []

    def test_description(self, client: sy.Synnax) -> None:
        """Should keep the description of a channel."""
        ch = virtual(client, description="the tank pressure")
        assert client.channels.retrieve(ch.key).description == "the tank pressure"


@pytest.mark.framer
class TestFramerLookups:
    def test_write_and_read_by_path(self, client: sy.Synnax) -> None:
        """Should open a writer and read by the paths of channels."""
        index, index_path = in_group(
            client, data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data, data_path = in_group(
            client, data_type=sy.DataType.FLOAT64, index=index.key
        )
        client.write(
            sy.TimeStamp.now(),
            {
                index_path: seconds_linspace(1, count=3),
                data_path: np.array([1.0, 2.0, 3.0]),
            },
        )
        series = client.read(sy.TimeRange.MAX, data_path)
        assert series.to_numpy().tolist() == [1.0, 2.0, 3.0]
        fr = client.read(sy.TimeRange.MAX, [index_path, data_path])
        assert data_path in fr.channels
        assert data.key not in fr.channels

    def test_label_a_mixed_read_by_key(self, client: sy.Synnax) -> None:
        """Should key a read that mixes a key and a name by key."""
        index, data = indexed(client)
        client.write(
            sy.TimeStamp.now(),
            {
                index.key: seconds_linspace(1, count=3),
                data.key: np.array([1.0, 2.0, 3.0]),
            },
        )
        mixed: list[int | str] = [index.key, data.name]
        fr = client.read(sy.TimeRange.MAX, mixed)
        assert set(fr.channels) == {index.key, data.key}

    def test_refuse_two_references_to_one_channel(self, client: sy.Synnax) -> None:
        """Should refuse an open that names one channel by its key and its name."""
        index, data = indexed(client)
        with pytest.raises(
            sy.ValidationError, match=f'{data.key} and "{data.name}" name one channel'
        ):
            client.open_writer(
                start=sy.TimeStamp.now(), channels=[index.key, data.key, data.name]
            )

    def test_dedupe_an_nfc_and_nfd_pair_on_open(self, client: sy.Synnax) -> None:
        """Should open one channel named in NFC and in NFD as one channel."""
        name = f"{channel_name()}_e\u0301"
        ch = virtual(client, name=name)
        nfc = unicodedata.normalize("NFC", name)
        assert nfc != name
        with client.open_streamer([nfc, name]) as streamer:
            assert streamer._adapter.keys == [ch.key]

    def test_refuse_a_record_with_an_nfc_and_nfd_pair(self, client: sy.Synnax) -> None:
        """Should refuse a record that names one channel in NFC and in NFD, since a
        dedupe would drop one entry's data.
        """
        name = f"{channel_name()}_e\u0301"
        ch = virtual(client, name=name)
        nfc = unicodedata.normalize("NFC", name)
        with client.open_writer(start=sy.TimeStamp.now(), channels=[ch.key]) as w:
            with pytest.raises(sy.ValidationError, match="name one channel"):
                w.write({nfc: np.array([1.0]), name: np.array([2.0])})

    def test_delete_the_data_of_a_key_and_a_name(self, client: sy.Synnax) -> None:
        """Should delete the data of each channel that a key or a name gives."""
        a, b = indexed(client)[0], indexed(client)[0]
        for ch in (a, b):
            client.write(sy.TimeStamp.now(), ch.key, seconds_linspace(1, count=3))
        client.delete([a.key, b.name], sy.TimeRange.MAX)
        assert len(client.read(sy.TimeRange.MAX, a.key)) == 0
        assert len(client.read(sy.TimeRange.MAX, b.key)) == 0

    def test_delete_the_data_of_a_path(self, client: sy.Synnax) -> None:
        """Should delete the data of the channel at a path."""
        index, path = in_group(client, data_type=sy.DataType.TIMESTAMP, is_index=True)
        client.write(sy.TimeStamp.now(), index.key, seconds_linspace(1, count=3))
        client.delete(path, sy.TimeRange.MAX)
        assert len(client.read(sy.TimeRange.MAX, index.key)) == 0
