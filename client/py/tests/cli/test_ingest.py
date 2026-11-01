#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

import pytest

import synnax as sy
from synnax.cli import console
from synnax.cli.flow import Context
from synnax.cli.ingest import _pure_ingest
from synnax.ingest import row
from synnax.io import FACTORY
from synnax.io.csv import CSVReader
from synnax.synnax import Synnax
from x.strings import random_name
from x.telem import seconds_linspace

from .data import DATA_DIR


@pytest.mark.cli
class TestIngest:
    def test_valid_ingest(self, client: Synnax) -> None:
        try:
            c = console.Mock(
                responses=[
                    True,  # Ingest all channels?
                    True,  # Channels not found, create them?
                    True,  # Are any channels indexed?
                    "ingest_valid_idx",  # Index channel
                    True,  # Do all non-indexed channels have the same data rate?
                    "ingest_valid_idx",  # Enter the name of the data rate or index?,
                    0,  # Guess data types from file.
                    True,  # Is the starting timestamp correct?
                    random_name(),
                ]
            )
            _pure_ingest(
                path_=DATA_DIR / "ingest_valid_1.csv",
                client=client,
                ctx=Context(console=c),
            )
        finally:
            client.channels.delete(
                ["ingest_valid_idx", "ingest_valid_1", "ingest_valid_2"]
            )


@pytest.mark.cli
class TestRowEngine:
    def test_ingest_with_index(
        self,
        client: Synnax,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        tmp_path: Path,
    ) -> None:
        """Should write a file that holds its index without the start it was given"""
        idx, data = indexed_pair
        client.write(1 * sy.TimeSpan.SECOND, {idx.key: seconds_linspace(1, count=1)})
        path = tmp_path / "with_index.csv"
        path.write_text(f"{idx.name},{data.name}\n10000000000,1.5\n11000000000,2.5\n")
        row.Engine(
            client=client,
            reader=FACTORY.new_reader(path),
            channels=[idx, data],
            start=sy.TimeStamp(1 * sy.TimeSpan.SECOND),
        ).run()
        frame = client.read(sy.TimeRange.MAX, [idx.key, data.key])
        assert list(frame[idx.key]) == [
            1 * sy.TimeSpan.SECOND,
            10 * sy.TimeSpan.SECOND,
            11 * sy.TimeSpan.SECOND,
        ]
        assert list(frame[data.key]) == [1.5, 2.5]

    def test_ingest_without_index(
        self,
        client: Synnax,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        tmp_path: Path,
    ) -> None:
        """Should align a file without its index to an index written before it"""
        idx, data = indexed_pair
        client.write(5 * sy.TimeSpan.SECOND, {idx.key: seconds_linspace(5, count=2)})
        path = tmp_path / "without_index.csv"
        path.write_text(f"{idx.name},{data.name}\n5000000000,1.5\n6000000000,2.5\n")
        row.Engine(
            client=client,
            reader=CSVReader(path, keys=[data.name]),
            channels=[data],
            start=sy.TimeStamp(5 * sy.TimeSpan.SECOND),
        ).run()
        frame = client.read(
            sy.TimeRange(5 * sy.TimeSpan.SECOND, 7 * sy.TimeSpan.SECOND), data.key
        )
        assert list(frame) == [1.5, 2.5]
