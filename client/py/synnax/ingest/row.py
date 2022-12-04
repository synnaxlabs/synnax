import gc
from datetime import datetime

from pandas import DataFrame
from rich.progress import Progress, TextColumn, BarColumn, TaskProgressColumn, \
    TimeElapsedColumn

from .. import Channel, Synnax
from ..telem import MEGABYTE, TimeStamp
from ..io import RowReader
from ..framer import DataFrameWriter


class RowIngestionEngine:
    """An ingestion engine that reads data from a row-based reader and writes it to a
    Synnax cluster.
    """
    client: Synnax
    writer: DataFrameWriter
    reader: RowReader
    channels: list[Channel]
    idx_grouped: dict[Channel, list[Channel]]

    def __init__(
        self,
        client: Synnax,
        reader: RowReader,
        channels: list[Channel],
        soft_mem_limit: int = 10 * MEGABYTE,
    ):
        self.channels = channels
        self.idx_grouped = {ch: list() for ch in channels if ch.is_index}
        for ch in self.idx_grouped:
            self.idx_grouped[ch] = [_ch for _ch in channels if _ch.index == ch.key[-1]]

        self.mem_limit = soft_mem_limit
        self.reader = reader
        self.client = client
        self.reader.set_chunk_size(1)
        df = self.reader.read()
        self.writer = self.client.data.new_writer(
            start=TimeStamp(df[list(self.idx_grouped.keys())[0].name][0]),
            keys=[ch.key for ch in channels],
        )
        self.reader.set_chunk_size(self.get_chunk_size())

    def get_chunk_size(self):
        """Sum the density of all channels to determine the chunk size.
        """
        return self.mem_limit // sum(ch.density for ch in self.channels)

    def run(self):
        """Run the ingestion engine.
        """
        try:
            with Progress(
                BarColumn(),
                TaskProgressColumn(),
                TextColumn("{task.completed} out of {task.total} samples"),
                TimeElapsedColumn(),
                TextColumn("{task.fields[tp]} samples/s"),
            ) as progress:
                task = progress.add_task("ingest", total=self.reader.nsamples, tp=0)
                while True:
                    try:
                        t0 = datetime.now()
                        chunk = self.reader.read()
                        self._write(chunk)
                        gc.collect()
                        progress.update(task, advance=chunk.size,
                                        tp=chunk.size / (
                                            datetime.now() - t0).total_seconds())
                    except StopIteration:
                        break
            self.writer.commit()
        finally:
            self.writer.close()

    def _write(self, df: DataFrame):
        for channel in self.channels:
            if channel.name in df.columns:
                df.rename(columns={channel.name: channel.key}, inplace=True)
        self.writer.write(df)

    def _get_channel(self, name: str):
        """Get the channel object from the list of channels"""
        for ch in self.channels:
            print(name, ch.name)
            if ch.name == name:
                return ch
        return None
