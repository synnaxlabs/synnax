#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from synnax.cli.connect import prompt_client_options
from synnax.cli.console.mock import MockConsole
from synnax.cli.flow import Context
from synnax.options import Options


@pytest.mark.cli
class TestPromptClientOptions:
    def test_responses(self) -> None:
        """Should build options from the prompt responses."""
        console = MockConsole(responses=["127.0.0.1", 9091, "admin", "secret", True])
        assert prompt_client_options(Context(console=console)) == Options(
            host="127.0.0.1",
            port=9091,
            username="admin",
            password="secret",
            secure=True,
        )
