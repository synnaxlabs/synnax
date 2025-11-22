#  Copyright 2025 Synnax Labs, Inc.
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
from synnax.synnax import Synnax

from .data import DATA_DIR
from synnax.util.rand import rand_name

CHANNELS = ["ingest_valid_idx", "ingest_valid_1", "ingest_valid_2"]


@pytest.mark.cli
class TestIngest:
    def test_valid_ingest(self, client: Synnax):
        try:
            c = MockConsole(
                responses=[
                    True,  # Ingest all channels?
                    True,  # Channels not found, create them?
                    True,  # Are any channels indexed?
                    "ingest_valid_idx",  # Index channel
                    True,  # Do all non-indexed channels have the same data rate?
                    "ingest_valid_idx",  # Enter the name of the data rate or index?,
                    0,  # Guess data types from file.
                    True,  # Is the starting timestamp correct?
                    rand_name(),
                ]
            )
            pure_ingest(
                path_=DATA_DIR / "ingest_valid_1.csv",
                client=client,
                ctx=Context(console=c),
            )
        finally:
            client.channels.delete(CHANNELS)
