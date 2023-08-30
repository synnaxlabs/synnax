#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
import numpy as np
import pandas as pd

import synnax as sy
from synnax.framer.frame import FramePayload


@pytest.mark.framer
@pytest.mark.telem
class TestFrame:
    def test_construction_from_cols_and_series(self):
        f = sy.Frame(["big", "red", "dog"], [np.array([1, 2, 3]), np.array([4, 5, 6])])
        assert f.col_type == "names"
        assert f["big"][0] == 1

    def test_construction_from_dict(self):
        f = sy.Frame({1: sy.Series([1, 2, 3, 4]), 3: sy.Series([4, 5, 6, 7])})
        assert f.col_type == "keys"
        assert f[1][0] == 1

    def test_construction_from_data_frame(self):
        f = sy.Frame(
            pd.DataFrame(
                {"big": sy.Series([1, 2, 3, 4]), "dog": sy.Series([4, 5, 6, 7])}
            )
        )
        assert f.col_type == "names"
        assert f["dog"][0] == 4

    def test_construction_from_frame(self):
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        f2 = sy.Frame(f)
        assert f is not f2
        assert f["big"] is f2["big"]
        assert f.col_type == "names"

    def test_construction_from_payload(self):
        pld = FramePayload([1, 2], [sy.Series([1, 2, 3]), sy.Series([4, 5, 6])])
        f = sy.Frame(pld)
        assert f.col_type == "keys"
        assert f[1][1] == 2
