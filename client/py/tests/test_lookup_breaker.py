#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import contextlib
import re
import unicodedata
from collections.abc import Iterator
from typing import Any

import numpy as np
import pytest

import synnax as sy
from synnax.channel.payload import lookup_where
from synnax.state import Latest, State
from synnax.telem import SampleValue
from x.strings import random_name


def channel_name() -> str:
    return f"brk_{random_name()}"


def virtual(client: sy.Synnax, name: str | None = None, **kw: Any) -> sy.Channel:
    return client.channels.create(
        name=channel_name() if name is None else name,
        data_type=sy.DataType.FLOAT64,
        virtual=True,
        **kw,
    )


def nfd_name() -> tuple[str, str]:
    """Returns one name in NFC and in NFD."""
    name = f"{channel_name()}_\u00e9"
    nfd = unicodedata.normalize("NFD", name)
    assert nfd != name
    return name, nfd


@pytest.fixture
def fresh_client(login_info: tuple[str, int, str, str]) -> Iterator[sy.Synnax]:
    host, port, username, password = login_info
    c = sy.Synnax(
        host=host, port=port, username=username, password=password, secure=False
    )
    yield c
    c.close()


def drain(streamer: sy.Streamer, seconds: float = 1.5) -> dict[int, list[float]]:
    """Returns every sample of each key that the streamer reads within seconds."""
    got: dict[int, list[float]] = {}
    deadline = sy.TimeStamp.now() + sy.TimeSpan.from_seconds(seconds)
    while sy.TimeStamp.now() < deadline:
        fr = streamer.read(timeout=0.2)
        if fr is None:
            continue
        for k in fr.channels:
            assert isinstance(k, int)
            got.setdefault(k, []).extend(fr[k].to_numpy().tolist())
    return got


@pytest.mark.framer
class TestWriterLookupBreaker:
    def test_a_record_path_after_a_rename_names_the_new_holder(
        self, client: sy.Synnax
    ) -> None:
        """Should write a record path to the channel that holds the path now."""
        g = client.groups.create(channel_name(), scope="channel")
        a = virtual(client, group=g.key)
        b = virtual(client, group=g.key)
        path = f"{g.name}.{a.name}"
        with client.open_streamer([a.key, b.key]) as s:
            with client.open_writer(
                start=sy.TimeStamp.now(), channels=[a.key, b.key]
            ) as w:
                w.write({path: np.array([1.0])})
                assert drain(s) == {a.key: [1.0]}
                old = a.name
                client.channels.rename(a.key, channel_name())
                client.channels.rename(b.key, old)
                assert client.channels.retrieve(path).key == b.key
                w.write({path: np.array([2.0])})
                assert drain(s) == {b.key: [2.0]}

    def test_not_found_names_a_non_ascii_name_as_given(self, client: sy.Synnax) -> None:
        """Should name a missing non-ASCII name as written, as the TS client does."""
        name = f"temp\u00e9rature_{random_name()}"
        with pytest.raises(sy.NotFoundError, match=re.escape(f'["{name}"]')):
            client.open_writer(start=sy.TimeStamp.now(), channels=[name])


def in_group(
    client: sy.Synnax, name: str | None = None, **kw: Any
) -> tuple[sy.Channel, str]:
    g = client.groups.create(channel_name(), scope="channel")
    ch = client.channels.create(
        name=channel_name() if name is None else name, group=g.key, **kw
    )
    return ch, f"{g.name}.{ch.name}"


def indexed_in_groups(
    client: sy.Synnax, data_name: str | None = None
) -> tuple[sy.Channel, str, sy.Channel, str]:
    """Creates an index and a data channel, each in its own folder, and writes three
    samples to them.
    """
    index, index_path = in_group(client, data_type=sy.DataType.TIMESTAMP, is_index=True)
    data, data_path = in_group(
        client, name=data_name, data_type=sy.DataType.FLOAT64, index=index.key
    )
    start = sy.TimeStamp.now()
    client.write(
        start,
        {
            index.key: np.array(
                [start + i * sy.TimeSpan.SECOND for i in range(3)], dtype=np.int64
            ),
            data.key: np.array([1.0, 2.0, 3.0]),
        },
    )
    return index, index_path, data, data_path


@pytest.mark.framer
class TestReadLabelBreaker:
    def test_a_frame_read_by_path_gives_the_series_by_key(
        self, client: sy.Synnax
    ) -> None:
        """Should give a channel's series by key from a frame read by path."""
        index, index_path, data, data_path = indexed_in_groups(client)
        by_name = client.read(sy.TimeRange.MAX, [index.name, data.name])
        assert by_name[data.key].to_numpy().tolist() == [1.0, 2.0, 3.0]
        by_path = client.read(sy.TimeRange.MAX, [index_path, data_path])
        assert by_path[data.key].to_numpy().tolist() == [1.0, 2.0, 3.0]

    def test_a_frame_read_by_path_keeps_its_time_index(self, client: sy.Synnax) -> None:
        """Should index the DataFrame of a frame read by path by time."""
        index, index_path, data, data_path = indexed_in_groups(client)
        by_name = client.read(sy.TimeRange.MAX, [index.name, data.name]).to_df()
        assert by_name.index.name == index.name
        by_path = client.read(sy.TimeRange.MAX, [index_path, data_path]).to_df()
        assert len(by_path.columns) == 1
        assert by_path.index.dtype == by_name.index.dtype

    def test_a_frame_read_by_an_nfd_name_gives_the_series_by_key(
        self, client: sy.Synnax
    ) -> None:
        """Should give a channel's series by key from a frame read by NFD name."""
        name, nfd = nfd_name()
        index, _, data, _ = indexed_in_groups(client, data_name=name)
        fr = client.read(sy.TimeRange.MAX, [index.name, nfd])
        assert fr[data.key].to_numpy().tolist() == [1.0, 2.0, 3.0]


@pytest.mark.control
class TestControllerLookupBreaker:
    def test_refuse_a_set_of_one_channel_by_key_and_name(
        self, client: sy.Synnax
    ) -> None:
        """Should refuse a set that names one channel by its key and its name."""
        ch = virtual(client)
        with client.control.acquire(name="brk", read=None, write=[ch.key]) as ctrl:
            with pytest.raises(sy.ValidationError, match="name one channel"):
                ctrl.set({ch.key: 1.0, ch.name: 2.0})  # type: ignore[dict-item]

    def test_refuse_a_set_of_an_nfc_and_nfd_pair(self, client: sy.Synnax) -> None:
        """Should refuse a set that names one channel in NFC and in NFD."""
        name, nfd = nfd_name()
        ch = virtual(client, name=name)
        with client.control.acquire(name="brk", read=None, write=[ch.key]) as ctrl:
            with pytest.raises(sy.ValidationError, match="name one channel"):
                ctrl.set({name: 1.0, nfd: 2.0})

    def test_refuse_a_set_authority_of_one_channel_twice(
        self, client: sy.Synnax
    ) -> None:
        """Should refuse an authority record that names one channel twice."""
        ch = virtual(client)
        with client.control.acquire(name="brk", read=None, write=[ch.key]) as ctrl:
            record: dict[Any, sy.CrudeAuthority] = {ch.key: 200, ch.name: 100}
            with pytest.raises(sy.ValidationError, match="name one channel"):
                ctrl.set_authority(record)

    def test_refuse_an_acquire_of_one_channel_by_key_and_name(
        self, client: sy.Synnax
    ) -> None:
        """Should refuse an acquire that names one channel by its key and its name."""
        ch = virtual(client)
        with pytest.raises(sy.ValidationError, match="name one channel"):
            with client.control.acquire(name="brk", read=None, write=[ch.key, ch.name]):
                pass

    def test_refuse_a_read_acquire_of_a_missing_name(self, client: sy.Synnax) -> None:
        """Should refuse an acquire that reads a name no channel holds, naming it."""
        missing = channel_name()
        with pytest.raises(sy.NotFoundError, match=missing):
            with client.control.acquire(name="brk", read=[missing], write=None):
                pass


@contextlib.asynccontextmanager
async def processing(client: sy.Synnax):  # type: ignore[no-untyped-def]
    task = asyncio.create_task(client.signals.process())
    try:
        yield task
    finally:
        task.cancel()
        with contextlib.suppress(asyncio.CancelledError, Exception):
            await task


@pytest.mark.signals
class TestSignalsLookupBreaker:
    @pytest.mark.asyncio
    async def test_a_handler_reads_the_path_it_registered(
        self, fresh_client: sy.Synnax
    ) -> None:
        """Should let a handler registered by a path read the channel by that path."""
        client = fresh_client
        g = client.groups.create(channel_name(), scope="channel")
        ch = virtual(client, group=g.key)
        path = f"{g.name}.{ch.name}"
        values: list[SampleValue] = []
        errors: list[BaseException] = []

        @client.signals.on(path, filter_f=lambda _: True)
        def handler(state: Latest) -> None:
            try:
                values.append(state[path])
            except BaseException as e:
                errors.append(e)
                raise

        async with processing(client):
            async with asyncio.timeout(5):
                while not values and not errors:
                    with client.open_writer(sy.TimeStamp.now(), ch.key) as w:
                        w.write({ch.key: [3.0]})
                    await asyncio.sleep(0.05)
        assert errors == []
        assert values[-1] == 3.0


class TestStateLookupBreaker:
    def test_read_a_name_in_nfd(self) -> None:
        """Should read a channel by its name in NFD, as lookups compare in NFC."""
        name, nfd = nfd_name()
        ch = sy.channel.Payload(key=7, name=name, data_type=sy.DataType.FLOAT32)
        state = State([ch])
        state.update(
            sy.Frame([ch.key], [sy.Series([1.5], data_type=sy.DataType.FLOAT32)])
        )
        assert state[nfd].to_numpy().tolist() == [1.5]


class TestPayloadLookupBreaker:
    def test_one_key_selects_with_in(self) -> None:
        """Should select keys with in, even for one key, as the design states."""
        where = lookup_where([7])
        assert where.key is not None
        assert where.key.in_ == [7]
        assert where.key.eq is None
