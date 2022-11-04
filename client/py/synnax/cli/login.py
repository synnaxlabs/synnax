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

