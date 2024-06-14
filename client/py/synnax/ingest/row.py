#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from datetime import datetime

from pandas import DataFrame
from rich.progress import (
    BarColumn,
    Progress,
    TaskProgressColumn,
    TextColumn,
    TimeElapsedColumn,
)

from synnax import Synnax
from synnax.channel import Channel
from synnax.framer import Writer
from synnax.io import RowFileReader
from synnax.telem import Size, TimeStamp


class RowIngestionEngine:
    """An ingestion engine that reads data from a row-based reader and writes it to a
    Synnax cluster.
    """

    client: Synnax
    writer: Writer
    reader: RowFileReader
    channels: list[Channel]
    idx_grouped: dict[Channel, list[Channel]]
    end: TimeStamp

    def __init__(
        self,
        client: Synnax,
        reader: RowFileReader,
        channels: list[Channel],
        start: TimeStamp,
        soft_mem_limit: int = 10 * Size.MB,
    ):
        self.channels = channels
        self.idx_grouped = {ch: list() for ch in channels if ch.is_index}
        for ch in self.idx_grouped:
            self.idx_grouped[ch] = [_ch for _ch in channels if _ch.index == ch.key]
        self.mem_limit = soft_mem_limit
        self.reader = reader
        self.client = client
        self.reader.set_chunk_size(self.get_chunk_size())
        self.writer = self.client.open_writer(
            start, [ch.key for ch in channels], err_on_extra_chans=False
        )
        self.end = start

    def get_chunk_size(self):
        """Sum the density of all channels to determine the chunk size."""
        return self.mem_limit // sum(ch.data_type.density for ch in self.channels)

    def run(self):
        """Run the ingestion engine."""
        self.reader.seek_first()
        try:
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
                        t0 = datetime.now()
                        chunk = self.reader.read()
                        self._write(chunk)
                        tp = chunk.size / (datetime.now() - t0).total_seconds()
                        progress.update(task, advance=chunk.size, tp=int(tp))
                    except StopIteration:
                        break
            self.end, _ = self.writer.commit()
        finally:
            self.reader.close()
            self.writer.close()

    def _write(self, df: DataFrame):
        for channel in self.channels:
            if channel.name in df.columns:
                df.rename(columns={channel.name: channel.key}, inplace=True)
                df[channel.key] = df[channel.key].astype(channel.data_type.np)
        self.writer.write(df)
