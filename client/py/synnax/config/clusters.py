#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from keyring.backend import KeyringBackend
from keyring.errors import PasswordDeleteError
from pydantic import BaseModel

from synnax.config.file import ConfigFile
from synnax.options import Options


class ClusterConfig(BaseModel):
    options: Options


PASSWORD_SERVICE = "synnax"
API_KEY_SERVICE = "synnax.api_key"


class ClustersConfig:
    """Saves connection options per cluster. The API key lives in the keyring and the
    rest in the config file. Reads a password that an older CLI saved, so that
    connecting can replace it with a key.
    """

    internal: ConfigFile
    keyring: KeyringBackend

    def __init__(self, config: ConfigFile, keyring: KeyringBackend):
        self.internal = config
        self.keyring = keyring

    def get(self, key: str = "default") -> ClusterConfig | None:
        c = self.internal.get(f"clusters.{key}")
        if c is None:
            return None
        opts = c["options"]
        api_key = self.keyring.get_password(API_KEY_SERVICE, key)
        if api_key:
            # A saved key wins over a username left by an interrupted migration.
            return ClusterConfig(
                options=Options(
                    host=opts["host"],
                    port=opts["port"],
                    secure=opts.get("secure", False),
                    api_key=api_key,
                )
            )
        pwd = self.keyring.get_password(PASSWORD_SERVICE, key) or ""
        return ClusterConfig(options=Options(**opts, password=pwd))

    def set(self, c: ClusterConfig, key: str = "default") -> None:
        """Saves the options and their API key, and deletes any saved password. A
        failure at any step leaves options that still connect.
        """
        p = c.model_dump(exclude={"options": {"username", "password"}})
        self.keyring.set_password(API_KEY_SERVICE, key, p["options"].pop("api_key"))
        self.internal.set(f"clusters.{key}", p)
        try:
            self.keyring.delete_password(PASSWORD_SERVICE, key)
        except PasswordDeleteError:
            pass
