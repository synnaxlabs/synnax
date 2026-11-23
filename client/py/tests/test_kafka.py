#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest

import synnax as sy


@pytest.mark.kafka
class TestKafkaReadRecord:
    """Tests for Kafka read record configuration."""

    def test_take_icd_read_messages(self) -> None:
        """Should take the ICD read messages that an MQTT entry takes."""
        message = sy.icd.ReadMessage(
            message=uuid4(),
            bindings=[sy.icd.ReadBinding(field=uuid4(), channel=12)],
        )
        record = sy.kafka.ReadRecord(icd=uuid4(), messages=[message])
        assert record.messages == [message]
