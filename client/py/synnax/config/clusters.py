#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import keyring
from pydantic import BaseModel

from synnax.config.file import File
from synnax.options import Options


class Cluster(BaseModel):
    options: Options


class Clusters:
    internal: File

    def __init__(self, config: File):
        self.internal = config

    def get(self, key: str = "default") -> Cluster | None:
        c = self.internal.get(f"clusters.{key}")
        if c is None:
            return None
        opts = c["options"]
        pwd = keyring.get_password("synnax", key)
        pwd = pwd or ""
        return Cluster(options=Options(**opts, password=pwd))

    def set(self, c: Cluster, key: str = "default") -> None:
        p = c.model_dump()
        keyring.set_password("synnax", key, p["options"].pop("password"))
        self.internal.set(f"clusters.{key}", p)
