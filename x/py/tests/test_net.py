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

from x.net import Address


class TestAddress:
    """Tests for the network address type."""

    def test_parses_an_address_with_no_port(self) -> None:
        """Should leave an absent port as None."""
        address = Address.model_validate({"host": "kafka.example.com"})
        assert address.host == "kafka.example.com"
        assert address.port is None

    def test_parses_an_address_with_a_port(self) -> None:
        """Should keep the port of an address."""
        assert Address.model_validate({"host": "10.0.0.2", "port": 502}).port == 502

    @pytest.mark.parametrize(
        "address",
        [{}, {"host": ""}, {"host": "plc", "port": 0}, {"host": "plc", "port": 65536}],
    )
    def test_rejects_an_invalid_address(self, address: dict[str, object]) -> None:
        """Should reject no host, an empty host, port 0, and a port over 65535."""
        with pytest.raises(ValidationError):
            Address.model_validate(address)
