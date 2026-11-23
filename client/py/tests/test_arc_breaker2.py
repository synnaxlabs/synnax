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
from x.testutil import assert_eventually


@pytest.mark.arc
class TestArcBreaker2:
    def test_create_from_a_retrieved_record(self, client: sy.Synnax) -> None:
        """Should create an Arc again from its retrieved record after a delete."""
        suffix = uuid4().hex[:8]
        src = client.channels.create(
            name=f"pb2_in_{suffix}", data_type=sy.DataType.FLOAT64, virtual=True
        )
        dst = client.channels.create(
            name=f"pb2_out_{suffix}", data_type=sy.DataType.FLOAT64, virtual=True
        )
        created = client.arcs.create(
            name=f"pb2-{suffix}",
            mode="text",
            text=sy.Text(raw=f"{src.name} * 2.0 -> {dst.name}"),
        )

        def compiled() -> sy.arc.Arc:
            retrieved = client.arcs.retrieve(key=created.key)
            assert retrieved.reads == [src.key]
            assert retrieved.writes == [dst.key]
            return retrieved

        assert_eventually(compiled)
        retrieved = compiled()
        client.arcs.delete(created.key)
        restored = client.arcs.create(retrieved)
        assert restored.key == created.key
