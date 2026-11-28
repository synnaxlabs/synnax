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
        """Should trust the system roots and present no certificate."""
        t = sy.tls.TLS.model_validate({})
        assert isinstance(t.server_trust, sy.tls.SystemRootsServerTrust)
        assert t.client_certificate is None

    def test_parses_a_certificate_authority(self) -> None:
        """Should keep a certificate authority and a client certificate."""
        ca, client = uuid4(), uuid4()
        t = sy.tls.TLS.model_validate(
            {
                "server_trust": {"type": "certificate_authority", "certificate": ca},
                "client_certificate": client,
            }
        )
        assert t.server_trust == sy.tls.CertificateAuthorityServerTrust(certificate=ca)
        assert t.client_certificate == client

    def test_parses_disabled_validation(self) -> None:
        """Should accept a server trust that disables validation."""
        t = sy.tls.TLS.model_validate({"server_trust": {"type": "validation_disabled"}})
        assert isinstance(t.server_trust, sy.tls.ValidationDisabledServerTrust)

    def test_rejects_a_certificate_that_is_not_a_key(self) -> None:
        """Should reject a certificate authority that is not a UUID."""
        with pytest.raises(ValidationError):
            sy.tls.TLS.model_validate(
                {
                    "server_trust": {
                        "type": "certificate_authority",
                        "certificate": "ca.pem",
                    }
                }
            )
