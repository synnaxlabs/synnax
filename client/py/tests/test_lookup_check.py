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
import unicodedata
from typing import Any, override

import numpy as np
import pandas as pd
import pytest

import synnax as sy
from alamos import NOOP
from synnax.channel.retrieve import Retriever
from synnax.channel.types_gen import RetrieveRequest
from synnax.framer.adapter import WriteFrameAdapter
from tests.channel import RetrieveCounter
from x.strings import random_name


def channel_name() -> str:
    return f"test_{random_name()}"


def in_group(client: sy.Synnax) -> tuple[sy.Channel, str]:
    """Creates a virtual channel in a new folder and returns it with its path."""
    g = client.groups.create(channel_name(), scope="channel")
    ch = client.channels.create(
        name=channel_name(), data_type=sy.DataType.FLOAT64, virtual=True, group=g.key
    )
    return ch, f"{g.name}.{ch.name}"


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


@pytest.mark.framer
class TestOpenLookupCheck:
    def test_open_sends_one_strict_request(
        self, client: sy.Synnax, requests: list[RetrieveRequest]
    ) -> None:
        """Should resolve the name and the path of an open in one strict request, so
        the Core refuses an ambiguous name.
        """
        by_name, _ = in_group(client)
        _, path = in_group(client)
        with client.open_writer(
            start=sy.TimeStamp.now(), channels=[by_name.name, path]
        ):
            pass
        assert [(r.strict, r.include_path) for r in requests] == [(True, True)]

    def test_retrieve_one_sends_a_text_strictly(
        self, client: sy.Synnax, requests: list[RetrieveRequest]
    ) -> None:
        """Should send one name or path strictly from retrieve_one, and a key not."""
        ch, path = in_group(client)
        retriever = client.channels._retriever
        retriever.retrieve_one(ch.name)
        retriever.retrieve_one(path)
        retriever.retrieve_one(ch.key)
        assert [r.strict for r in requests] == [True, True, False]

    def test_open_names_every_miss(self, client: sy.Synnax) -> None:
        """Should name every lookup that finds no channel."""
        a, b = channel_name(), f"{channel_name()}.{channel_name()}"
        with pytest.raises(
            sy.NotFoundError, match=f'channels not found matching \\["{a}", "{b}"\\]'
        ):
            client.open_writer(start=sy.TimeStamp.now(), channels=[a, b])


@pytest.mark.framer
class TestWriterLookupCheck:
    def test_opened_path_needs_no_retrieve(
        self, counted_client: tuple[sy.Synnax, RetrieveCounter]
    ) -> None:
        """Should answer a record keyed by the path that opened the writer with no
        retrieve.
        """
        client, counter = counted_client
        _, path = in_group(client)
        with client.open_writer(start=sy.TimeStamp.now(), channels=[path]) as w:
            counter.count = 0
            for i in range(3):
                w.write({path: np.array([float(i)])})
            assert counter.count == 0

    def test_unheld_paths_of_a_record_go_in_one_retrieve(
        self, counted_client: tuple[sy.Synnax, RetrieveCounter]
    ) -> None:
        """Should resolve the paths of one record that the writer did not open by in
        one retrieve.
        """
        client, counter = counted_client
        a, a_path = in_group(client)
        b, b_path = in_group(client)
        with client.open_writer(start=sy.TimeStamp.now(), channels=[a.key, b.key]) as w:
            counter.count = 0
            w.write({a_path: np.array([1.0]), b_path: np.array([1.0])})
            assert counter.count == 1

    def test_no_path_is_kept_across_records(self, client: sy.Synnax) -> None:
        """Should resolve a path that the writer did not open by again for each record,
        so a moved channel's old path finds nothing (D-314 Q2 corrected).
        """
        g = client.groups.create(channel_name(), scope="channel")
        ch = client.channels.create(
            name=channel_name(),
            data_type=sy.DataType.FLOAT64,
            virtual=True,
            group=g.key,
        )
        old = f"{g.name}.{ch.name}"
        with client.open_writer(start=sy.TimeStamp.now(), channels=[ch.key]) as w:
            w.write({old: np.array([1.0])})
            client.groups.rename(g.key, channel_name())
            with pytest.raises(sy.NotFoundError, match="channels not found matching"):
                w.write({old: np.array([2.0])})

    def test_extra_dataframe_column_needs_no_retrieve(
        self, counted_client: tuple[sy.Synnax, RetrieveCounter]
    ) -> None:
        """Should skip a DataFrame column that names no opened channel with no
        retrieve, as before 7i.
        """
        client, counter = counted_client
        ch, _ = in_group(client)
        with client.open_writer(
            start=sy.TimeStamp.now(), channels=[ch.key], err_on_extra_chans=False
        ) as w:
            counter.count = 0
            for i in range(3):
                w.write(pd.DataFrame({ch.name: [float(i)], "notes": [0.0]}))
            assert counter.count == 0


@pytest.mark.framer
class TestWriterHeldCheck:
    def test_refuse_a_name_that_two_held_channels_share(self) -> None:
        """Should refuse a record keyed by a name that two held channels share, as
        stored data from before unique names can hold.
        """
        held = [
            sy.channel.Payload(
                key=1, name="tank", data_type=sy.DataType.FLOAT64, path="a.tank"
            ),
            sy.channel.Payload(
                key=2, name="tank", data_type=sy.DataType.FLOAT64, path="b.tank"
            ),
        ]

        class Stub(Retriever):
            def __init__(self) -> None:
                self.instrumentation = NOOP

            @override
            def filter(self, req: RetrieveRequest) -> list[sy.channel.Payload]:
                return list(held)

        adapter = WriteFrameAdapter(retriever=Stub())
        adapter.update(["a.tank", "b.tank"])
        with pytest.raises(
            sy.ValidationError, match='The writer holds 2 channels named "tank"'
        ):
            adapter.adapt_dict_keys({"tank": 1.0})

    def test_skip_a_dataframe_column_of_an_unopened_channel(
        self, client: sy.Synnax
    ) -> None:
        """Should skip a DataFrame column that names a channel the writer did not
        open, or refuse it when err_on_extra_chans is set.
        """
        a, _ = in_group(client)
        b, _ = in_group(client)
        df = pd.DataFrame({a.name: [1.0], b.name: [2.0]})
        with client.open_streamer([a.key, b.key]) as s:
            with client.open_writer(
                start=sy.TimeStamp.now(), channels=[a.key], err_on_extra_chans=False
            ) as w:
                w.write(df)
            fr = s.read(timeout=5)
            assert fr is not None
            assert a.key in fr.channels
            assert b.key not in fr.channels
        with client.open_writer(start=sy.TimeStamp.now(), channels=[a.key]) as w:
            with pytest.raises(
                sy.ValidationError, match=f"Channel {b.name} was not provided"
            ):
                w.write(df)


@pytest.mark.control
class TestControllerLookupCheck:
    def test_access_by_path_sends_no_retrieves(
        self, counted_client: tuple[sy.Synnax, RetrieveCounter]
    ) -> None:
        """Should read, command, and set authority by an acquired path without
        retrieving, as by name.
        """
        client, counter = counted_client
        _, read_path = in_group(client)
        _, cmd_path = in_group(client)
        with client.control.acquire(
            name="resolution", read=[read_path], write=[cmd_path]
        ) as ctrl:
            counter.count = 0
            for i in range(3):
                ctrl.get(read_path)
                ctrl.set(cmd_path, i)
                ctrl.set({cmd_path: i})
                ctrl.set_authority(cmd_path, authority=200)
            assert counter.count == 0

    def test_access_by_nfd_name_sends_no_retrieves(
        self, counted_client: tuple[sy.Synnax, RetrieveCounter]
    ) -> None:
        """Should command an acquired channel by the NFD form of its name without
        retrieving.
        """
        client, counter = counted_client
        nfd = f"{channel_name()}_é"
        client.channels.create(name=nfd, data_type=sy.DataType.FLOAT64, virtual=True)
        nfc = unicodedata.normalize("NFC", nfd)
        with client.control.acquire(name="resolution", read=None, write=[nfc]) as ctrl:
            counter.count = 0
            ctrl.set(nfd, 1.0)
            assert counter.count == 0

    def test_read_acquire_sends_a_strict_request(
        self, client: sy.Synnax, requests: list[RetrieveRequest]
    ) -> None:
        """Should resolve the names of a read acquire in a strict request, as a write
        acquire does, so the Core refuses an ambiguous name.
        """
        ch, _ = in_group(client)
        with client.control.acquire(name="strict", read=[ch.name], write=None):
            pass
        assert requests[0].strict

    def test_acquire_refuses_a_key_and_a_name_of_one_channel(
        self, client: sy.Synnax
    ) -> None:
        """Should refuse a write acquire that names one channel by its key and its
        name.
        """
        ch, _ = in_group(client)
        with pytest.raises(
            sy.ValidationError, match=f'{ch.key} and "{ch.name}" name one channel'
        ):
            with client.control.acquire(
                name="shared", read=None, write=[ch.key, ch.name]
            ):
                pass

    def test_set_refuses_a_key_and_a_name_of_one_channel(
        self, client: sy.Synnax
    ) -> None:
        """Should refuse a set that names one channel by its key and its name, since
        one value would be lost.
        """
        ch, _ = in_group(client)
        values: Any = {ch.key: 1.0, ch.name: 2.0}
        authorities: Any = {ch.key: 100, ch.name: 200}
        with client.control.acquire(name="shared", read=None, write=[ch.key]) as ctrl:
            with pytest.raises(sy.ValidationError, match="name one channel"):
                ctrl.set(values)
            with pytest.raises(sy.ValidationError, match="name one channel"):
                ctrl.set_authority(authorities)

    def test_read_acquire_names_a_miss(self, client: sy.Synnax) -> None:
        """Should refuse a read acquire of a name that finds no channel, as a write
        acquire does.
        """
        name = channel_name()
        with pytest.raises(
            sy.NotFoundError, match=f'channels not found matching \\["{name}"\\]'
        ):
            with client.control.acquire(name="missing", read=[name], write=None):
                pass


@pytest.mark.signals
class TestSignalLookupCheck:
    @pytest.mark.asyncio
    async def test_handler_reads_by_its_path(
        self, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should give a handler registered by path its value by that path."""
        host, port, username, password = login_info
        client = sy.Synnax(
            host=host, port=port, username=username, password=password, secure=False
        )
        ch, path = in_group(client)
        values: list[Any] = []

        @client.signals.on(path, filter_f=lambda _: True)
        def handler(state: Any) -> None:
            values.append(state[path])

        task = asyncio.create_task(client.signals.process())
        try:
            async with asyncio.timeout(5):
                while not values and not task.done():
                    with client.open_writer(sy.TimeStamp.now(), ch.key) as w:
                        w.write({ch.key: [4.0]})
                    await asyncio.sleep(0.05)
            if task.done():
                task.result()
        finally:
            task.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await task
            client.close()
        assert values[-1] == 4.0
