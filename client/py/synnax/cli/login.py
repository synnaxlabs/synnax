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

import click

from synnax.cli.connect import connect_from_options, prompt_client_options
from synnax.cli.console.rich import RichConsole
from synnax.cli.flow import Context
from synnax.config import ClusterConfig, ClustersConfig, ConfigFile


@click.command()
def login():
    """Logs the user into a Synnax cluster and saves the parameters to the configuration
    file.
    """
    ctx = Context(console=RichConsole())
    options = prompt_client_options(ctx)
    synnax = connect_from_options(ctx, options)
    if synnax is None:
        return
    cfg = ClustersConfig(ConfigFile(Path(os.path.expanduser("~/.synnax"))))
    cfg.set(ClusterConfig(options=options))
    ctx.console.info(SUCCESSFUL_LOGIN)


SUCCESSFUL_LOGIN = """Saved credentials. You can now use the Synnax Client
without having to log in. To connect the cluster in a Python shell, use the following:

from synnax import Synnax
client = Synnax()
"""
