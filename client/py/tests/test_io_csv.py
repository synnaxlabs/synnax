#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import math
import pathlib

import pandas as pd
import pytest

from synnax.io.csv import CSVWriter


@pytest.mark.io
class TestCSVWriter:
    def test_write_infinities_as_words(self, tmp_path: pathlib.Path) -> None:
        """Should write infinities as Infinity and -Infinity."""
        writer = CSVWriter(tmp_path / "out.csv")
        writer.write(pd.DataFrame({"a": [1.5, math.inf, -math.inf]}))
        writer.close()
        assert (tmp_path / "out.csv").read_text() == "a\n1.5\nInfinity\n-Infinity\n"
