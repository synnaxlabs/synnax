#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
from pathlib import Path

from synnax.config.clusters import ClusterConfig, ClustersConfig
from synnax.config.file import ConfigFile
from synnax.exceptions import ValidationError
from synnax.options import SynnaxOptions

CONFIG_FILE_PATH = Path(os.path.expanduser("~/.synnax"))


def load_options() -> SynnaxOptions | None:
    """Loads the connection parameters from a configuration file.
    :return: The options to connect to a Synnax server.
    """
    cluster = ClustersConfig(ConfigFile(CONFIG_FILE_PATH)).get()
    return None if cluster is None else cluster.options


NO_OPTIONS_MSG = """No options provided and no configuration file
                    was found. Please provide options or login via
                    the CLI  using the `synnax login` command."""


def try_load_options_if_none_provided(
    host: str = "",
    port: int = 0,
    username: str = "",
    password: str = "",
    secure: bool = False,
) -> SynnaxOptions:
    if len(host) == 0:
        opts = load_options()
        if opts is None:
            raise ValidationError(NO_OPTIONS_MSG)
        return opts
    return SynnaxOptions(
        host=host,
        port=port,
        username=username,
        password=password,
        secure=secure,
    )
