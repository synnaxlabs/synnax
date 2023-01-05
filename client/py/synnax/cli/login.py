#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
from pathlib import Path

from synnax import Synnax
from synnax.cli.connect import prompt_client_options, connect_client
from synnax.cli.console.rich import RichConsole
from synnax.cli.flow import Context
from synnax.config import ConfigFile, ClustersConfig, ClusterConfig


def login():
    """Logs the user into a Synnax cluster and saves the parameters to the configuration
    file.
    """
    ctx = Context(console=RichConsole())
    options = prompt_client_options(ctx)
    connect_client(ctx, options)
    cfg = ClustersConfig(ConfigFile(Path(os.path.expanduser("~/.synnax"))))
    cfg.set(ClusterConfig(options=options))

    cfg.get()


def credentials():
    """Prints the credentials of the currently logged in cluster."""
