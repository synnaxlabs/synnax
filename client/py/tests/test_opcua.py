#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from synnax.hardware.opcua import ReadTaskConfig


@pytest.mark.opcua
class TestOPCUATask:
    def test_parse_opcua_read_task(self):
        data = {
            "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
            "sample_rate": 10,
            "stream_rate": 5,
            "array_mode": False,
            "array_size": 1,
            "data_saving": False,
            "channels": [
                {
                    "name": "",
                    "key": "k09AWoiyLxN",
                    "node_id": "NS=2;I=8",
                    "channel": 1234,
                    "enabled": True,
                    "use_as_index": False,
                }
            ],
        }
        ReadTaskConfig.parse_obj(data)
