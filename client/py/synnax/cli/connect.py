import os
from pathlib import Path

from freighter.exceptions import Unreachable

from .. import Synnax, AuthError, SynnaxOptions
from .flow import Context
from ..config import ConfigFile, ClustersConfig


def load_config_options(ctx: Context) -> SynnaxOptions:
    """Loads the connection parameters from a configuration file.

    :param ctx: The context of the current flow.
    :return: The options to connect to a Synnax server.
    """
    cfg = ClustersConfig(ConfigFile(Path(os.path.expanduser("~/.synnax"))))
    return cfg.get().options


def prompt_client_options(ctx: Context) -> SynnaxOptions:
    """Prompts the user for the parameters to connect to a Synnax server.

    :param ctx: The context of the current flow.
    :return: The options to connect to a Synnax server.
    """
    ctx.console.info("Enter your Synnax connection parameters:")
    params = dict()
    params['host'] = ctx.console.ask("Host", default="localhost")
    params['port'] = ctx.console.ask_int("Port", default=9090)
    params['username'] = ctx.console.ask("Username", default="synnax")
    params['password'] = ctx.console.ask_password("Password")
    return SynnaxOptions(**params)


def connect_client(ctx: Context, opts: SynnaxOptions) -> Synnax | None:
    """Connects to a Synnax server. Prints user-friendly messages to the console if
    the connection fails.

    :param ctx: The context of the current flow.
    :param opts: The options to connect to a Synnax server.
    :return: The connected Synnax client, or None if the connection failed.
    """
    try:
        client = Synnax(**opts.dict())
    except Unreachable:
        return ctx.console.error(
            f"Cannot reach Synnax server at {opts.host}:{opts.port}"
        )
    except AuthError:
        return ctx.console.error("Invalid credentials")
    except Exception as e:
        raise e
        # return ctx.console.error(f"An error occurred: {e}")
    ctx.console.success("Connection successful!")
    return client
