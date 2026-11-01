#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

from pandas import DataFrame
from rich.progress import (
    BarColumn,
    Progress,
    TaskProgressColumn,
    TextColumn,
    TimeElapsedColumn,
)

from synnax import Synnax
from synnax.channel import Channel, Key
from synnax.framer import Writer, WriterMode
from synnax.io import RowFileReader
from synnax.telem import Size, TimeStamp


class Engine:
    """An ingestion engine that reads data from a row-based reader and writes it to a
    Synnax cluster.
    """

    client: Synnax
    writers: list[tuple[Writer, list[Key]]]
    reader: RowFileReader
    channels: list[Channel]
    idx_grouped: dict[Channel, list[Channel]]
    start: TimeStamp
    end: TimeStamp

    def __init__(
        self,
        *,
        client: Synnax,
        reader: RowFileReader,
        channels: list[Channel],
        start: TimeStamp,
        soft_mem_limit: int = 10 * Size.MB,
    ):
        self.channels = channels
        self.idx_grouped = {ch: [] for ch in channels if ch.is_index}
        for ch in self.idx_grouped:
            self.idx_grouped[ch] = [
                _ch for _ch in channels if ch.key in (_ch.key, _ch.index)
            ]
        self.mem_limit = soft_mem_limit
        self.reader = reader
        self.client = client
        self.reader.set_chunk_size(self.get_chunk_size())
        self.writers = []
        self.start = start
        self.end = start

    def get_chunk_size(self) -> int:
        """Sum the density of all channels to determine the chunk size."""
        return self.mem_limit // sum(ch.data_type.density for ch in self.channels)

    def _open_writer(self, start: TimeStamp, channels: list[Channel]) -> None:
        keys = [ch.key for ch in channels]
        writer = self.client.open_writer(start, keys, mode=WriterMode.PERSIST)
        self.writers.append((writer, keys))

    def _open_writers(self) -> None:
        """Open a writer for each index at its own first timestamp, since a start from
        another index can land inside this index's stored data. Channels with no index
        in the file share a writer at the start given to the engine.
        """
        self.reader.seek_first()
        first = self.reader.read()
        for idx, channels in self.idx_grouped.items():
            self._open_writer(TimeStamp(first[idx.name].to_numpy()[0]), channels)
        indexes = {idx.key for idx in self.idx_grouped}
        data_only = [
            ch for ch in self.channels if not ch.is_index and ch.index not in indexes
        ]
        if data_only:
            self._open_writer(self.start, data_only)

    def run(self) -> None:
        """Run the ingestion engine."""
        try:
            self._open_writers()
            self.reader.seek_first()
            with Progress(
                BarColumn(),
                TaskProgressColumn(),
                TextColumn("{task.completed} out of {task.total} samples"),
                TimeElapsedColumn(),
                TextColumn("{task.fields[tp]} samples/s"),
            ) as progress:
                task = progress.add_task("ingest", total=self.reader.nsamples(), tp=0)
                while True:
                    try:
                        t0 = time.perf_counter()
                        chunk = self.reader.read()
                        self._write(chunk)
                        tp = chunk.size / (time.perf_counter() - t0)
                        progress.update(task, advance=chunk.size, tp=int(tp))
                    except StopIteration:
                        break
            self.end = max(writer.commit() for writer, _ in self.writers)
        finally:
            self.reader.close()
            for writer, _ in self.writers:
                writer.close()

    def _write(self, df: DataFrame) -> None:
        for channel in self.channels:
            if channel.name in df.columns:
                df.rename(columns={channel.name: channel.key}, inplace=True)
                df[channel.key] = df[channel.key].astype(channel.data_type.np)
        for writer, keys in self.writers:
            writer.write(df[[key for key in keys if key in df.columns]])
