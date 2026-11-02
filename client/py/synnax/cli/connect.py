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

from keyring.errors import KeyringError

from freighter.exceptions import Unreachable
from synnax import Synnax
from synnax.auth import api_key_id
from synnax.cli.flow import Context
from synnax.config import Cluster, Clusters
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


def from_clusters(ctx: Context, clusters: Clusters) -> Synnax | None:
    """Connects to a Synnax server. Prompts the user for the connection parameters if
    none are saved. A saved password is replaced with an API key after the connection
    succeeds.

    :param ctx: The context of the current flow.
    :param clusters: The saved connection parameters.
    :returns: The connected client, or None if the connection failed.
    """
    cluster = clusters.get()
    if cluster is None:
        return from_context(ctx, opts=prompt_client_options(ctx))
    ctx.console.info("Using saved credentials.")
    client = from_context(ctx, opts=cluster.options)
    if client is not None and not cluster.options.api_key:
        try:
            save_api_key(client, opts=cluster.options, clusters=clusters)
        except (OSError, KeyringError, AuthError, Unreachable) as e:
            ctx.console.warn(f"Cannot replace the saved password with an API key: {e}")
    return client


def save_api_key(client: Synnax, *, opts: Options, clusters: Clusters) -> None:
    """Creates an API key for the logged-in user and saves it with opts in place of
    the username and password. Deletes the key when it is not saved, and, once the new
    one is saved, deletes the key it replaces when the same account on the same Core
    owns it.
    """
    previous = clusters.get()
    token, key = client.auth.create_key(name=f"Python CLI on {socket.gethostname()}")
    try:
        clusters.set(
            Cluster(
                options=Options(
                    host=opts.host,
                    port=opts.port,
                    secure=opts.secure,
                    ca_cert=opts.ca_cert,
                    api_key=token,
                )
            )
        )
    except (OSError, KeyringError):
        # a failure after the save, such as a stale password that cannot be deleted,
        # leaves the key in use
        saved = clusters.get()
        if saved is None or saved.options.api_key != token:
            client.auth.delete_keys(key.key)
        raise
    if (
        previous is None
        or not previous.options.api_key
        or (previous.options.host, previous.options.port) != (opts.host, opts.port)
    ):
        return
    previous_key = api_key_id(previous.options.api_key)
    # A key another account saved is not ours to delete.
    if any(k.key == previous_key for k in client.auth.retrieve_keys()):
        client.auth.delete_keys(previous_key)


def from_context(ctx: Context, *, opts: Options) -> Synnax | None:
    """Connects to a Synnax server. Prints user-friendly messages to the console if
    the connection fails.

    :param ctx: The context of the current flow.
    :param opts: The options to connect with.
    :returns: The connected client, or None if the connection failed.
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
