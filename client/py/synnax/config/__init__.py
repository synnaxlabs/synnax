#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
from pathlib import Path

import keyring

from synnax.config.clusters import Cluster, Clusters
from synnax.config.file import File
from synnax.exceptions import ValidationError
from synnax.options import Options

__all__ = ["Cluster", "Clusters", "File", "default_clusters"]

CONFIG_FILE_PATH = Path(os.path.expanduser("~/.synnax"))


def default_clusters() -> Clusters:
    """Returns the clusters saved in the user's config file and OS keyring."""
    return Clusters(File(config_dir=CONFIG_FILE_PATH), keyring.get_keyring())


def load_options() -> Options | None:
    """Loads the connection parameters from a configuration file.
    :return: The options to connect to a Synnax server.
    """
    cluster = default_clusters().get()
    return None if cluster is None else cluster.options


NO_OPTIONS_MSG = """No options provided and no configuration file was found. Please
                    provide options or login via the CLI  using the `sy login`
                    command."""


def try_load_options_if_none_provided(
    *,
    host: str = "",
    port: int = 0,
    username: str = "",
    password: str = "",
    secure: bool = False,
    ca_cert: str | None = None,
    api_key: str = "",
) -> Options:
    if len(host) == 0:
        if api_key:
            raise ValidationError("give a host with an API key")
        opts = load_options()
        if opts is None:
            raise ValidationError(NO_OPTIONS_MSG)
        return opts
    return Options(
        host=host,
        port=port,
        username=username,
        password=password,
        api_key=api_key,
        secure=secure,
        ca_cert=ca_cert,
    )
