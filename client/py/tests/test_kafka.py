#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from pydantic import ValidationError

import synnax as sy


@pytest.mark.kafka
class TestKafkaReadTask:
    """Tests for Kafka read task configuration."""

    def test_parse_field_kinds(self) -> None:
        """Should select each read field variant from its type."""
        cfg = sy.kafka.ReadConfig.model_validate(
            {
                "device": "cluster-1",
                "topic": "telemetry",
                "records": [
                    {
                        "fields": [
                            {
                                "type": "integer",
                                "channel": 3,
                                "pointer": "/state",
                                "enum_values": [{"label": "ON", "value": "1"}],
                                "fill": "-1",
                            },
                            {
                                "type": "timestamp",
                                "pointer": "/ts",
                                "time_format": "unix_ms",
                            },
                            {"type": "json", "pointer": "/meta"},
                        ]
                    }
                ],
            }
        )
        integer, timestamp, json = cfg.records[0].fields
        assert isinstance(integer, sy.kafka.IntegerReadField)
        assert integer.enum_values[0].value == 1
        assert integer.fill == -1
        assert isinstance(timestamp, sy.kafka.TimestampReadField)
        assert timestamp.time_format == "unix_ms"
        assert isinstance(json, sy.kafka.JsonReadField)
        assert json.pointer == "/meta"

    def test_reject_unknown_field_type(self) -> None:
        """Should reject a field whose type is not a kind."""
        with pytest.raises(ValidationError, match="does not match any of the"):
            sy.kafka.ReadConfig.model_validate(
                {"records": [{"fields": [{"type": "number"}]}]}
            )


@pytest.mark.kafka
class TestKafkaProperties:
    """Tests for Kafka device properties."""

    def test_secure(self) -> None:
        """Should read and write TLS as secure."""
        props = sy.kafka.Properties.model_validate(
            {"brokers": ["localhost:9092"], "secure": True}
        )
        assert props.secure is True
        assert props.model_dump()["secure"] is True
        assert "tls" not in props.model_dump()
