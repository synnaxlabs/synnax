#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import getpass
import sys

from synnax import config
from synnax.cli import connect
from synnax.cli.prompt import Prompt


def login() -> bool:
    """Logs the user into a Core, creates an API key, and saves the key and the
    connection parameters. Returns whether the login succeeded.
    """
    # getpass reads the terminal, and on Windows the console, even when stdin is a
    # pipe. Piped input would then wait for keys that never come.
    read_secret = getpass.getpass if sys.stdin.isatty() else input
    return connect.login(Prompt(read_secret=read_secret), config.default_clusters())
