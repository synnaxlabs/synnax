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

import synnax as sy


class TestTLS:
    """Tests for the shared TLS configuration type."""

    def test_applies_the_defaults(self) -> None:
        """Should leave both certificates absent and keep validation on."""
        t = sy.tls.TLS.model_validate({})
        assert t.ca_certificate is None
        assert t.client_certificate is None
        assert t.certificate_validation_disabled is False

    def test_parses_both_certificates(self) -> None:
        """Should keep a CA, a client certificate, and the validation flag."""
        ca, client = uuid4(), uuid4()
        t = sy.tls.TLS.model_validate(
            {
                "ca_certificate": ca,
                "client_certificate": client,
                "certificate_validation_disabled": True,
            }
        )
        assert t.ca_certificate == ca
        assert t.client_certificate == client
        assert t.certificate_validation_disabled is True

    def test_rejects_a_certificate_that_is_not_a_key(self) -> None:
        """Should reject a CA certificate that is not a UUID."""
        with pytest.raises(ValidationError):
            sy.tls.TLS.model_validate({"ca_certificate": "ca.pem"})
