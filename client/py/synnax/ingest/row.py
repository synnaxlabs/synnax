import gc
from datetime import datetime

from pandas import DataFrame

from .. import Channel, Synnax
from ..telem import MEGABYTE
from ..io import RowReader
from ..segment import NumpyWriter


class RowIngestionEngine:
    """An ingestion engine that reads data from a row-based reader and writes it to a
    Synnax cluster.
    """
    client: Synnax
    writer: NumpyWriter
    reader: RowReader
    channels: list[Channel]
    idx_grouped: dict[Channel, list[Channel]]

    def __init__(
        self,
        client: Synnax,
        reader: RowReader,
        channels: list[Channel],
        soft_mem_limit: int = 500 * MEGABYTE,
    ):
        self.channels = channels
        self.idx_grouped = {ch: list() for ch in channels if ch.is_index}
        for ch in self.idx_grouped:
            self.idx_grouped[ch] = [_ch for _ch in channels if _ch.index == ch.key[-1]]

        self.mem_limit = soft_mem_limit
        self.reader = reader
        self.reader.set_chunk_size(self.get_chunk_size())
        self.client = client
        self.writer = self.client.data.new_writer([ch.key for ch in channels])

    def get_chunk_size(self):
        """Sum the density of all channels to determine the chunk size.
        """
        return self.mem_limit // sum(ch.density for ch in self.channels)

    def run(self):
        """Run the ingestion engine.
        """
        try:
            while True:
                try:
                    chunk = self.reader.read()
                    self._run_chunk(chunk)
                    gc.collect()
                except StopIteration:
                    break
            self.writer.commit()
        finally:
            self.writer.close()

    def _run_chunk(self, chunk: DataFrame):
        """Ingest a chunk of data.
        """
        for idx, channels in self.idx_grouped.items():
            idx_data = chunk[idx.name].to_numpy(dtype=idx.data_type)
            start = idx_data[0]
            print(self.writer.write(idx.key, start, idx_data))
            for ch in channels:
                t0 = datetime.now()
                ch_data = chunk[ch.name].to_numpy(dtype=ch.data_type)
                print("conversion time:", datetime.now() - t0)
                print(ch.key, len(ch_data))
                t0 = datetime.now()
                print(self.writer.write(ch.key, start, ch_data))
                print("write time:", datetime.now() - t0)
