#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from framework.config_client import ConfigClient
from framework.target_filter import parse_target


class TestLoad:
    """Tests for loading the tests a filter selects."""

    def test_should_name_every_sequence_filter_when_nothing_matches(self) -> None:
        target = parse_target("console/missing_a,missing_b/...")
        with pytest.raises(
            ValueError,
            match=r"^No tests found matching filters: sequence='missing_a,missing_b'$",
        ):
            ConfigClient().load(target)

    def test_should_record_the_test_file_of_each_test(self) -> None:
        _, definitions = ConfigClient().load(parse_target("driver/pagerduty"))
        assert [(d.case, d.file) for d in definitions] == [
            ("client/pagerduty_alert", "driver")
        ]
