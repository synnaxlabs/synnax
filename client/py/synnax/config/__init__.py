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

from ..options import SynnaxOptions
from .clusters import ClusterConfig, ClustersConfig
from .file import ConfigFile


def load_options() -> SynnaxOptions | None:
    """Loads the connection parameters from a configuration file.
    :return: The options to connect to a Synnax server.
    """
    cluster = ClustersConfig(ConfigFile(Path(os.path.expanduser("~/.synnax")))).get()
    return None if cluster is None else cluster.options
