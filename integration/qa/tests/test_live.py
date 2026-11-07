#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np

from qa.live import frame


class TestFrame:
    """Tests the shapes, timestamps, and value ranges of generated signals."""

    def test_should_hold_one_sample_per_stamp_on_each_channel(self) -> None:
        stamps = np.arange(5, dtype=np.int64)
        f = frame(stamps, np.arange(5) / 50)
        assert list(f) == ["qa_time", "qa_sine", "qa_noise", "qa_square"]
        assert all(series.size == 5 for series in f.values())
        assert f["qa_time"] is stamps

    def test_should_keep_each_signal_in_its_range(self) -> None:
        f = frame(np.zeros(1000, dtype=np.int64), np.arange(1000) / 50)
        assert f["qa_sine"].min() >= 0
        assert f["qa_sine"].max() <= 100
        assert f["qa_square"].dtype == np.uint8
        assert set(np.unique(f["qa_square"])) <= {0, 1}
