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
from pydantic import ValidationError

from synnax import log


class TestLog:
    def test_round_trip(self) -> None:
        """Should parse channel entries and dump them back unchanged."""
        lg = log.Log.model_validate(
            {"name": "Events", "channels": [{"channel": 1, "color": "#00ff00"}]}
        )
        entry = lg.channels[0]
        assert entry.precision is None
        assert entry.notation == "standard"
        assert entry.timestamp == log.TimestampConfig(format="preciseDate", tz="local")
        assert log.Log.model_validate_json(lg.model_dump_json()) == lg

    def test_precision_bounds(self) -> None:
        """Should reject a precision above the float64 limit."""
        with pytest.raises(ValidationError):
            log.ChannelEntry.model_validate(
                {"channel": 1, "color": "#00ff00", "precision": 18}
            )

    def test_ontology_id(self) -> None:
        """Should build the ontology ID of a log."""
        key = uuid4()
        assert log.ontology_id(key).type == log.ONTOLOGY_TYPE.type
        assert log.ontology_id(key).key == str(key)
        lg = log.Log(name="Events")
        assert hash(lg) == hash(lg.key)
