#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from synnax.cli.console import MockConsole
from synnax.cli.flow import Context
from synnax.cli.ingest import pure_ingest
from synnax.exceptions import QueryError
from synnax.synnax import Synnax

from .data import DATA_DIR

CHANNELS = ["ingest-valid-idx", "ingest-valid-1", "ingest-valid-2"]

@pytest.mark.cli
class TestIngest:
    def test_valid_ingest(self, client: Synnax):
        try:
            c = MockConsole(
                responses=[
                    True,  # Ingest all channels?
                    True,  # Channels not found, create them?
                    True,  # Are any channels indexed?
                    "ingest-valid-idx",  # Index channel
                    True,  # Do all non-indexed channels have the same data rate?
                    "ingest-valid-idx",  # Enter the name of the data rate or index?,
                    0,  # Guess data types from file.
                    True,  # Is the starting timestamp correct?
                    "Random Range",
                ]
            )
            pure_ingest(
                path_=DATA_DIR / "ingest_valid_1.csv",
                client=client,
                ctx=Context(console=c),
            )
        finally:
            client.channels.delete(CHANNELS)

