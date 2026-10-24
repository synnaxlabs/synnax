#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from examples.mqtt_sim import sparkplug_b_pb2 as pb
from examples.mqtt_sim.sparkplug import Value, metric_value, new_metric


@pytest.mark.mqtt
class TestSparkplugMetric:
    """Tests for the Sparkplug B metrics of the mock MQTT plant."""

    @pytest.mark.parametrize(
        ("data_type", "value"),
        [
            (pb.Int8, -128),
            (pb.Int16, -2),
            (pb.Int32, -2_000_000_000),
            (pb.Int64, -(2**63)),
            (pb.UInt32, 2**32 - 1),
            (pb.UInt64, 2**64 - 1),
            (pb.Float, 0.25),
            (pb.Double, -1.5),
            (pb.Boolean, True),
            (pb.String, "open"),
        ],
    )
    def test_value_round_trip(self, data_type: int, value: Value) -> None:
        """Should read back the value of a metric after a wire round trip."""
        payload = pb.Payload(metrics=[new_metric(data_type, value, name="tag")])
        decoded = pb.Payload.FromString(payload.SerializeToString()).metrics[0]
        assert decoded.name == "tag"
        assert metric_value(decoded) == value

    def test_signed_integer_two_complement(self) -> None:
        """Should put a negative 32-bit integer in int_value as two's complement."""
        assert new_metric(pb.Int32, -1).int_value == 0xFFFF_FFFF

    def test_omit_unset_fields(self) -> None:
        """Should leave out the name, alias, and timestamp when they are not given."""
        m = new_metric(pb.Double, 1.0)
        assert not m.HasField("name")
        assert not m.HasField("alias")
        assert not m.HasField("timestamp")

    def test_reject_unsupported_data_type(self) -> None:
        """Should reject a data type that is not a scalar."""
        with pytest.raises(ValueError, match="unsupported data type 16"):
            new_metric(pb.DataSet, 1)

    def test_no_value(self) -> None:
        """Should return None for a metric that holds no value."""
        assert metric_value(pb.Payload.Metric(name="tag")) is None
