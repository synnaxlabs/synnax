#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from alamos import NOOP
from freighter import URL
from synnax.exceptions import create_registry
from synnax.transport import Transport


class TestTransport:
    def test_ca_cert_insecure(self) -> None:
        """Should reject a CA certificate for an insecure connection."""
        with pytest.raises(ValueError, match="secure"):
            Transport(
                URL("localhost", 9090),
                NOOP,
                create_registry(),
                ca_cert="/certs/ca.crt",
            )
