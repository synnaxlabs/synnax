#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import keyring
from pydantic import BaseModel

from synnax.config.file import ConfigFile
from synnax.options import SynnaxOptions


class ClusterConfig(BaseModel):
    options: SynnaxOptions


class ClustersConfig:
    internal: ConfigFile

    def __init__(self, config: ConfigFile):
        self.internal = config

    def get(self, key: str = "default") -> ClusterConfig | None:
        c = self.internal.get(f"clusters.{key}")
        if c is None:
            return None
        opts = c["options"]
        pwd = keyring.get_password("synnax", key)
        pwd = pwd or ""
        return ClusterConfig(options=SynnaxOptions(**opts, password=pwd))

    def set(self, c: ClusterConfig, key: str = "default"):
        p = c.dict()
        keyring.set_password("synnax", key, p["options"].pop("password"))
        self.internal.set(f"clusters.{key}", p)
