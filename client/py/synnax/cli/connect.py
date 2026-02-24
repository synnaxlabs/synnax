#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter.exceptions import Unreachable

from synnax import Synnax
from synnax.cli.flow import Context
from synnax.config import load_options
from synnax.exceptions import AuthError
from synnax.options import Options


def prompt_client_options(ctx: Context) -> Options:
    """Prompts the user for the parameters to connect to a Synnax server.

    :param ctx: The context of the current flow.
    :return: The options to connect to a Synnax server.
    """
    ctx.console.info("Enter your Synnax connection parameters:")
    return Options(
        host=ctx.console.ask("Host", default="localhost"),
        port=ctx.console.ask("Port", default=9090),
        username=ctx.console.ask("Username", default="synnax"),
        password=ctx.console.ask("Password", password=True),
        secure=ctx.console.ask("Secure Connection?", default=False),
    )


def connect_client(ctx: Context) -> Synnax | None:
    """Connects to a Synnax server. Prompts the user for the connection parameters if
    no configuration file exists.

    :param ctx: The context of the current flow.
    :return: The connected Synnax py, or None if the connection failed.
    """
    opts = load_options()
    if opts is None:
        opts = prompt_client_options(ctx)
    else:
        ctx.console.info("Using saved credentials.")
    return connect_from_options(ctx, opts)


def connect_from_options(ctx: Context, opts: Options) -> Synnax | None:
    """Connects to a Synnax server. Prints user-friendly messages to the console if
    the connection fails.

    :param ctx: The context of the current flow.
    :param opts: The options to connect to a Synnax server.
    :return: The connected Synnax py, or None if the connection failed.
    """
    try:
        client = Synnax(**opts.model_dump())
    except Unreachable:
        ctx.console.error(f"Cannot reach Synnax server at {opts.host}:{opts.port}")
        return None
    except AuthError:
        ctx.console.error("Invalid credentials")
        return None
    except Exception as e:
        raise e
    ctx.console.success("Connection successful!")
    return client
