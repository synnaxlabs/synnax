#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import pytest

from synnax.hardware import sequence


@pytest.mark.sequence
class TestSequence:
    def test_parse_sequence_config(self):
        sequence.Config.model_validate(
            {
                "rate": 10,
                "read": [],
                "write": [],
                "script": "",
                "globals": {},
            }
        )
