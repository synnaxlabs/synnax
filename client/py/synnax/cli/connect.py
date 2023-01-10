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

from freighter.exceptions import Unreachable

from .. import AuthError, Synnax, SynnaxOptions
from ..config import ClustersConfig, ConfigFile, load_options
from .flow import Context


def prompt_client_options(ctx: Context) -> SynnaxOptions:
    """Prompts the user for the parameters to connect to a Synnax server.

    :param ctx: The context of the current flow.
    :return: The options to connect to a Synnax server.
    """
    ctx.console.info("Enter your Synnax connection parameters:")
    params = dict()
    params["host"] = ctx.console.ask("Host", default="localhost")
    params["port"] = ctx.console.ask_int("Port", default=9090)
    params["username"] = ctx.console.ask("Username", default="synnax")
    params["password"] = ctx.console.ask_password("Password")
    params["secure"] = ctx.console.confirm("Secure connection?", default=False)
    return SynnaxOptions(**params)


def connect_client(ctx: Context) -> Synnax | None:
    """Connects to a Synnax server. Prompts the user for the connection parameters if
    no configuration file exists.

    :param ctx: The context of the current flow.
    :return: The connected Synnax client, or None if the connection failed.
    """
    opts = load_options()
    if opts is None:
        opts = prompt_client_options(ctx)
    else:
        ctx.console.info("Using saved credentials.")
    return connect_from_options(ctx, opts)


def connect_from_options(ctx: Context, opts: SynnaxOptions) -> Synnax | None:
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
