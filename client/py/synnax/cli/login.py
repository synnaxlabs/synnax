#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax import config
from synnax.cli import connect
from synnax.cli.prompt import Prompt


def login() -> bool:
    """Logs the user into a Core, creates an API key, and saves the key and the
    connection parameters. Returns whether the login succeeded.
    """
    return connect.login(Prompt(), config.default_clusters())
