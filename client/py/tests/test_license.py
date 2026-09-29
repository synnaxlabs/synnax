#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re

import pytest

import freighter
import synnax as sy


@pytest.mark.license
class TestLicenseClient:
    """Tests for the license client."""

    def test_retrieve(self, client: sy.Synnax):
        """Should report the Core's license state and host fingerprint."""
        info = client.license.retrieve()
        assert all(re.fullmatch(r"[0-9a-f]{64}", h) for h in info.fingerprint)
        assert (info.license is None) == (info.state == "missing")

    def test_activate_invalid_token(self, client: sy.Synnax):
        """Should refuse a token that cannot be verified."""
        with pytest.raises(sy.InvalidLicense):
            client.license.activate("not-a-token")

    @pytest.mark.parametrize(
        "type_, exc",
        [
            ("sy.license", sy.LicenseError),
            ("sy.license.missing", sy.MissingLicense),
            ("sy.license.expired", sy.ExpiredLicense),
            ("sy.license.invalid", sy.InvalidLicense),
            ("sy.license.fingerprint", sy.LicenseFingerprintMismatch),
            ("sy.license.too_many", sy.LicenseLimitExceeded),
            ("sy.license.unknown", sy.LicenseError),
        ],
    )
    def test_decode(self, type_: str, exc: type[sy.LicenseError]):
        """Should decode each license error type the Core sends."""
        err = freighter.decode_exception(
            freighter.ExceptionPayload(type=type_, data="refused")
        )
        assert type(err) is exc
        assert str(err) == "refused"
