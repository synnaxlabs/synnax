#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import socket
import ssl
from pathlib import Path

from freighter.exceptions import Unreachable
from synnax import Synnax
from synnax.cli.flow import Context
from synnax.config import ClusterConfig, ClustersConfig
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
    password = ctx.console.ask("Password", str, password=True)
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


def connect_client(ctx: Context, clusters: ClustersConfig) -> Synnax | None:
    """Connects to a Synnax server. Prompts the user for the connection parameters if
    none are saved. A saved password is replaced with an API key after the connection
    succeeds.

    :param ctx: The context of the current flow.
    :param clusters: The saved connection parameters.
    :return: The connected Synnax py, or None if the connection failed.
    """
    cluster = clusters.get()
    if cluster is None:
        return connect_from_options(ctx, prompt_client_options(ctx))
    ctx.console.info("Using saved credentials.")
    client = connect_from_options(ctx, cluster.options)
    if client is not None and not cluster.options.api_key:
        save_api_key(client, cluster.options, clusters)
    return client


def save_api_key(client: Synnax, opts: Options, clusters: ClustersConfig) -> None:
    """Creates an API key for the logged-in user and saves it with opts in place of
    the username and password.
    """
    token, _ = client.auth.create_key(name=f"Python CLI on {socket.gethostname()}")
    clusters.set(
        ClusterConfig(
            options=Options(
                host=opts.host,
                port=opts.port,
                secure=opts.secure,
                ca_cert=opts.ca_cert,
                api_key=token,
            )
        )
    )


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
    except (FileNotFoundError, ssl.SSLError) as e:
        if opts.ca_cert is None:
            raise
        ctx.console.error(f"Cannot load the CA certificate at {opts.ca_cert}: {e}")
        return None
    except Exception as e:
        raise e
    ctx.console.success("Connection successful!")
    return client
