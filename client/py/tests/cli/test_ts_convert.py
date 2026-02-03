#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy
from synnax.cli.console.mock import MockConsole
from synnax.cli.flow import Context
from synnax.cli.ts_convert import pure_tsconvert
from synnax.io import IO_FACTORY

from .data import DATA_DIR

IN_FILE = DATA_DIR / "tsconvert.csv"
OUT_FILE = DATA_DIR / "tsconvert_out.csv"


@pytest.fixture
def remove_testdata():
    yield
    if OUT_FILE.exists():
        OUT_FILE.unlink()


@pytest.mark.cli
@pytest.mark.tsconvert
class TestTSConvert:
    @pytest.mark.usefixtures("remove_testdata")
    def test_tsconvert(self):
        pure_tsconvert(
            ctx=Context(console=MockConsole(), prompt_enabled=False),
            input_path=DATA_DIR / "tsconvert.csv",
            output_path=DATA_DIR / "tsconvert_out.csv",
            input_channel="Time",
            output_channel="Time",
            input_precision="s",
            output_precision="ns",
        )
        f = IO_FACTORY.new_reader(DATA_DIR / "tsconvert_out.csv")
        f.set_chunk_size(1)
        f.seek_first()
        df = f.read()
        assert df["Time"].to_numpy()[0] == int(123e9)

    @pytest.mark.usefixtures("remove_testdata")
    def test_tsconvert_with_string_data_channel(self):
        pure_tsconvert(
            ctx=Context(console=MockConsole(), prompt_enabled=False),
            input_path=DATA_DIR / "tsconvert_string_first_col.csv",
            output_path=DATA_DIR / "tsconvert_out.csv",
            input_channel="Time",
            output_channel="Time",
            input_precision="s",
            output_precision="ns",
        )
        f = IO_FACTORY.new_reader(DATA_DIR / "tsconvert_out.csv")
        f.set_chunk_size(1)
        f.seek_first()
        df = f.read()
        assert df["Time"].to_numpy()[0] == int(1e9)

    @pytest.mark.usefixtures("remove_testdata")
    def test_tsconvert_iso(self):
        pure_tsconvert(
            ctx=Context(console=MockConsole(), prompt_enabled=False),
            input_path=DATA_DIR / "tsconvert_iso.csv",
            output_path=DATA_DIR / "tsconvert_out.csv",
            input_channel="Time",
            output_channel="Time",
            input_precision="iso",
            output_precision="ns",
        )
        f = IO_FACTORY.new_reader(DATA_DIR / "tsconvert_out.csv")
        f.set_chunk_size(1)
        f.seek_first()
        df = f.read()
        assert (
            sy.TimeStamp(df["Time"].to_numpy()[0]) - sy.TimeStamp(1483257600000000000)
            < sy.TimeSpan.DAY
        )
