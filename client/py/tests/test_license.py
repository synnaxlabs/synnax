#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy


class TestLicenseClient:
    """Tests for the license client."""

    def test_retrieve(self, client: sy.Synnax):
        """Should report the Core's license state and host fingerprint."""
        info = client.license.retrieve()
        assert info.state in ("ok", "missing", "expired")
        assert isinstance(info.fingerprint, list)

    def test_activate_invalid_token(self, client: sy.Synnax):
        """Should refuse a token that cannot be verified."""
        with pytest.raises(sy.InvalidLicense):
            client.license.activate("not-a-token")
