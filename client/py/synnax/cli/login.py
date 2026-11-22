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
    # The standard getpass reads the terminal, and on Windows the console, even when
    # stdin is a pipe. A frontend such as Jupyter replaces it with a masked prompt.
    masked = sys.stdin.isatty() or getpass.getpass.__module__ != "getpass"
    read_secret = getpass.getpass if masked else input
    return connect.login(Prompt(read_secret=read_secret), config.default_clusters())
