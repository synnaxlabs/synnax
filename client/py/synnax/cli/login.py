#  Copyright 2026 Synnax Labs, Inc.
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

from synnax import config
from synnax.cli import console
from synnax.cli.connect import connect_from_options, prompt_client_options
from synnax.cli.flow import Context


@click.command()
def login() -> None:
    """Logs the user into a Synnax cluster and saves the parameters to the configuration
    file.
    """
    flow_ctx = Context(console=console.Rich())
    options = prompt_client_options(flow_ctx)
    synnax = connect_from_options(flow_ctx, options)
    if synnax is None:
        return
    cfg = config.Clusters(config.File(Path(os.path.expanduser("~/.synnax"))))
    cfg.set(config.Cluster(options=options))
    flow_ctx.console.info(SUCCESSFUL_LOGIN)


SUCCESSFUL_LOGIN = """Saved credentials. You can now use the Synnax Client
without having to log in. To connect the cluster in a Python shell, use the following:

from synnax import Synnax
client = Synnax()
"""
