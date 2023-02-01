#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

import pytest

from synnax.cli.ts_convert import pure_tsconvert
from synnax.io import IO_FACTORY

DATA_DIR = Path(__file__).parent / "testdata"

@pytest.fixture
def remove_testdata():
    yield
    (DATA_DIR / "tsconvert_out.csv").unlink()

class TestTSConvert:
    @pytest.mark.focus
    def test_tsconvert(self, remove_testdata):
        pure_tsconvert(
            path=DATA_DIR / "tsconvert.csv",
            out=DATA_DIR / "tsconvert_out.csv",
            channel="Time",
            input="s",
            output="ns",
        )
        f = IO_FACTORY.new_reader(DATA_DIR / "tsconvert_out.csv")
        f.set_chunk_size(1)
        f.seek_first()
        df = f.read()
        assert df["Time"][0] == int(123e9)
