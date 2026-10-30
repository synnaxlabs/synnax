#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import click

from synnax import config
from synnax.cli import connect, console
from synnax.cli.flow import Context


@click.command()
def login() -> None:
    """Logs the user into a Synnax cluster, creates an API key, and saves the key and
    the connection parameters.
    """
    run(Context(console=console.Rich()), config.default_clusters())


def run(ctx: Context, clusters: config.Clusters) -> None:
    """Prompts for connection parameters, connects, and saves an API key for the
    logged-in user to clusters.
    """
    options = connect.prompt_client_options(ctx)
    synnax = connect.from_context(ctx, opts=options)
    if synnax is None:
        return
    connect.save_api_key(synnax, opts=options, clusters=clusters)
    ctx.console.info(_SUCCESSFUL_LOGIN)


_SUCCESSFUL_LOGIN = """Saved an API key. You can now use the Synnax Client
without having to log in. To connect the cluster in a Python shell, use the following:

from synnax import Synnax
client = Synnax()
"""
