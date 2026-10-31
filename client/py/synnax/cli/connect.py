#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ssl
from pathlib import Path

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
    host = ctx.console.ask("Host", default="localhost")
    port = ctx.console.ask("Port", type_=int, default=9090)
    username = ctx.console.ask("Username", default="synnax")
    password = ctx.console.ask("Password", type_=str, password=True)
    secure = ctx.console.ask("Secure Connection?", type_=bool, default=False)
    ca_cert = None
    if secure:
        path = ctx.console.ask(
            "CA certificate path (empty for system roots)", default=""
        )
        if path:
            ca_cert = str(Path(path).expanduser().resolve())
    return Options(
        host=host,
        port=port,
        username=username,
        password=password,
        secure=secure,
        ca_cert=ca_cert,
    )


def from_context(ctx: Context, *, opts: Options | None = None) -> Synnax | None:
    """Connects to a Synnax server. Prints user-friendly messages to the console if
    the connection fails.

    :param ctx: The context of the current flow.
    :param opts: The options to connect with. If None, loads the saved options, or
        prompts the user for them if no configuration file exists.
    :returns: The connected client, or None if the connection failed.
    """
    if opts is None:
        opts = load_options()
        if opts is None:
            opts = prompt_client_options(ctx)
        else:
            ctx.console.info("Using saved credentials.")
    try:
        client = Synnax(**opts.model_dump())
    except Unreachable:
        ctx.console.error(f"Cannot reach Synnax server at {opts.host}:{opts.port}")
        return None
    except AuthError:
        ctx.console.error("Invalid credentials")
        return None
    except (FileNotFoundError, ssl.SSLError) as e:
        if opts.ca_cert is None:
            raise
        ctx.console.error(f"Cannot load the CA certificate at {opts.ca_cert}: {e}")
        return None
    except Exception as e:
        raise e
    ctx.console.success("Connection successful!")
    return client
