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

from freighter.exceptions import Timeout, Unreachable
from synnax import Synnax
from synnax.auth import api_key_id
from synnax.cli.prompt import Prompt
from synnax.config import Cluster, Clusters
from synnax.exceptions import AuthError, ConfigurationError
from synnax.options import Options
from x.exceptions import Error


def _prompt_client_options(prompt: Prompt) -> Options:
    prompt.info("Enter your Synnax connection parameters:")
    host = prompt.ask("Host", default="localhost")
    port = prompt.ask_int("Port", default=9090)
    username = prompt.ask("Username", default="synnax")
    password = prompt.secret("Password")
    secure = prompt.confirm("Secure connection?")
    ca_cert = None
    if secure:
        path = prompt.ask("CA certificate path (empty for system roots)")
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


def _save_api_key(
    client: Synnax, *, opts: Options, clusters: Clusters, prompt: Prompt
) -> None:
    """Creates an API key for the logged-in user and saves it with opts in place of
    the username and password. Deletes the key when it is not saved, and, once the new
    one is saved, deletes the key it replaces when the same account on the same Core
    owns it. Warns when that last deletion fails, because the login has succeeded.
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
    except (OSError, KeyringError, ConfigurationError) as e:
        saved = clusters.get()
        if saved is not None and saved.options.api_key == token:
            # The save worked, and only the cleanup of the old password failed.
            prompt.error(f"Saved the API key, but cannot delete the old password: {e}")
            return
        try:
            client.auth.delete_keys(key.key)
        except Error as cleanup:
            e.add_note(f"The new API key {key.key} is still on the Core: {cleanup}")
        raise
    if (
        previous is None
        or not previous.options.api_key
        or (previous.options.host, previous.options.port) != (opts.host, opts.port)
    ):
        return
    previous_key = api_key_id(previous.options.api_key)
    try:
        # A key another account saved is not ours to delete.
        if any(k.key == previous_key for k in client.auth.retrieve_keys()):
            client.auth.delete_keys(previous_key)
    except Error as e:
        prompt.error(f"Saved the new API key, but cannot delete the old one: {e}")


def _connect(prompt: Prompt, *, opts: Options) -> Synnax | None:
    try:
        client = Synnax(**opts.model_dump())
    except Unreachable:
        prompt.error(f"Cannot reach Synnax server at {opts.host}:{opts.port}")
        return None
    except Timeout:
        prompt.error(f"No response from Synnax server at {opts.host}:{opts.port}")
        return None
    except AuthError:
        prompt.error("Invalid credentials")
        return None
    except (FileNotFoundError, ssl.SSLError) as e:
        if opts.ca_cert is None:
            raise
        prompt.error(f"Cannot load the CA certificate at {opts.ca_cert}: {e}")
        return None
    prompt.info("Connection successful!")
    return client


def login(prompt: Prompt, clusters: Clusters) -> bool:
    """Prompts for connection parameters, connects, and saves an API key for the
    logged-in user to clusters. Prints why a login fails.

    :returns: Whether the login succeeded.
    """
    options = _prompt_client_options(prompt)
    synnax = _connect(prompt, opts=options)
    if synnax is None:
        return False
    try:
        _save_api_key(synnax, opts=options, clusters=clusters, prompt=prompt)
    except ConfigurationError as e:
        prompt.error(" ".join([str(e), *getattr(e, "__notes__", [])]))
        return False
    except (OSError, KeyringError, Error) as e:
        notes = getattr(e, "__notes__", [])
        prompt.error(" ".join([f"Cannot save the API key: {e}", *notes]))
        return False
    prompt.info(_SUCCESSFUL_LOGIN)
    return True


_SUCCESSFUL_LOGIN = """Saved an API key. You can now use the Synnax Client
without having to log in. To connect the cluster in a Python shell, use the following:

from synnax import Synnax
client = Synnax()
"""
