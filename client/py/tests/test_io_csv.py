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

import numpy as np
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

    @pytest.mark.parametrize(
        ("value", "text"),
        [
            (1e-7, "1e-7"),
            (1.5e-7, "1.5e-7"),
            (1e-6, "0.000001"),
            (1e20, "100000000000000000000"),
            (1e21, "1e+21"),
            (1e10, "10000000000"),
            (1.0, "1"),
            (0.1, "0.1"),
            (-0.0, "-0"),
        ],
    )
    def test_write_float64_text(
        self, tmp_path: pathlib.Path, value: float, text: str
    ) -> None:
        """Should write the shortest float64 text, in exponent form past the cutoffs."""
        writer = CSVWriter(tmp_path / "out.csv")
        writer.write(pd.DataFrame({"a": np.array([value], dtype=np.float64)}))
        assert (tmp_path / "out.csv").read_text() == f"a\n{text}\n"

    def test_write_float32_text(self, tmp_path: pathlib.Path) -> None:
        """Should write the shortest text at float32 width."""
        writer = CSVWriter(tmp_path / "out.csv")
        writer.write(pd.DataFrame({"a": np.array([0.1, 1e-7], dtype=np.float32)}))
        assert (tmp_path / "out.csv").read_text() == "a\n0.1\n1e-7\n"

    def test_write_nan(self, tmp_path: pathlib.Path) -> None:
        """Should write NaN as NaN and keep other columns as they are."""
        writer = CSVWriter(tmp_path / "out.csv")
        writer.write(pd.DataFrame({"a": [math.nan, 2.5], "b": [1, 2]}))
        assert (tmp_path / "out.csv").read_text() == "a,b\nNaN,1\n2.5,2\n"

    def test_write_floats_in_object_columns(self, tmp_path: pathlib.Path) -> None:
        """Should format the floats of a mixed object column."""
        writer = CSVWriter(tmp_path / "out.csv")
        col = pd.Series([math.inf, 1e-7, "offline"], dtype=object)
        writer.write(pd.DataFrame({"a": col}))
        assert (tmp_path / "out.csv").read_text() == "a\nInfinity\n1e-7\noffline\n"
