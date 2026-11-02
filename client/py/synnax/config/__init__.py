#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
from pathlib import Path

import keyring

from synnax.config.clusters import Cluster, Clusters
from synnax.config.file import File
from synnax.exceptions import ValidationError
from synnax.options import Options

__all__ = [
    "Cluster",
    "Clusters",
    "File",
    "default_clusters",
    "load_desktop_options",
]

CONFIG_FILE_PATH = Path(os.path.expanduser("~/.synnax"))
DESKTOP_FILE_PATH = CONFIG_FILE_PATH / "desktop.json"


def default_clusters() -> Clusters:
    """Returns the clusters saved in the user's config file and OS keyring."""
    return Clusters(File(config_dir=CONFIG_FILE_PATH), keyring.get_keyring())


def load_desktop_options(path: Path = DESKTOP_FILE_PATH) -> Options | None:
    """Loads the options that Synnax Desktop saved for scripts on this computer.

    :param path: The file Synnax Desktop writes.
    :return: The options, or None if Synnax Desktop has saved none.
    """
    try:
        raw = json.loads(path.read_text())
    except FileNotFoundError:
        return None
    return Options.model_validate(raw)


NO_OPTIONS_MSG = (
    "No connection parameters were given and none are saved. Open Synnax Desktop, "
    "log in with `sy login`, or give a host and a port."
)

DESKTOP_NOT_RUNNING_MSG = (
    "Synnax Desktop is not running. Open it and try again. To connect to another "
    "Core, give a host and a port."
)


def try_load_options_if_none_provided(
    *,
    host: str = "",
    port: int = 0,
    username: str = "",
    password: str = "",
    secure: bool = False,
    ca_cert: str | None = None,
    api_key: str = "",
) -> tuple[Options, bool]:
    """Returns the given options, or, without a host, the options that `sy login`
    saved, then those that Synnax Desktop saved.

    :return: The options, and whether Synnax Desktop saved them.
    :raises ValidationError: If an API key has no host, or nothing is saved.
    """
    if len(host) == 0:
        if api_key:
            raise ValidationError("give a host with an API key")
        cluster = default_clusters().get()
        if cluster is not None:
            return cluster.options, False
        opts = load_desktop_options(DESKTOP_FILE_PATH)
        if opts is None:
            raise ValidationError(NO_OPTIONS_MSG)
        return opts, True
    return (
        Options(
            host=host,
            port=port,
            username=username,
            password=password,
            api_key=api_key,
            secure=secure,
            ca_cert=ca_cert,
        ),
        False,
    )
