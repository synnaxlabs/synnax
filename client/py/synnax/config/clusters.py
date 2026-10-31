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

from synnax.config.file import File
from synnax.options import Options


class Cluster(BaseModel):
    options: Options


_PASSWORD_SERVICE = "synnax"
_API_KEY_SERVICE = "synnax.api_key"


# the entry holds the address, so a key never reaches another Core
def _api_key_entry(key: str, host: str, port: int) -> str:
    return f"{key}@{host}:{port}"


class Clusters:
    """Saves connection options per cluster. The API key lives in the keyring under its
    address and the rest in the config file. Reads a password that an older CLI saved,
    so that connecting can replace it with a key.
    """

    internal: File
    keyring: KeyringBackend

    def __init__(self, config: File, keyring: KeyringBackend):
        self.internal = config
        self.keyring = keyring

    def get(self, key: str = "default") -> Cluster | None:
        c = self.internal.get(f"clusters.{key}")
        if c is None:
            return None
        opts = c["options"]
        api_key = self.keyring.get_password(
            _API_KEY_SERVICE, _api_key_entry(key, opts["host"], opts["port"])
        )
        if api_key:
            # A saved key wins over a username left by an interrupted migration.
            return Cluster(
                options=Options(
                    host=opts["host"],
                    port=opts["port"],
                    secure=opts.get("secure", False),
                    ca_cert=opts.get("ca_cert"),
                    api_key=api_key,
                )
            )
        pwd = self.keyring.get_password(_PASSWORD_SERVICE, key) or ""
        return Cluster(options=Options(**opts, password=pwd))

    def set(self, c: Cluster, *, key: str = "default") -> None:
        """Saves the options and their API key, and deletes any saved password. A
        failed save leaves the saved options as they were.
        """
        prev = self.internal.get(f"clusters.{key}.options")
        p = c.model_dump(exclude={"options": {"username", "password"}})
        entry = _api_key_entry(key, c.options.host, c.options.port)
        prev_api_key = self.keyring.get_password(_API_KEY_SERVICE, entry)
        self.keyring.set_password(_API_KEY_SERVICE, entry, p["options"].pop("api_key"))
        try:
            self.internal.set(f"clusters.{key}", p)
        except OSError:
            if prev_api_key is None:
                self.keyring.delete_password(_API_KEY_SERVICE, entry)
            else:
                self.keyring.set_password(_API_KEY_SERVICE, entry, prev_api_key)
            raise
        stale = [(_PASSWORD_SERVICE, key)]
        if prev is not None:
            prev_entry = _api_key_entry(key, prev["host"], prev["port"])
            if prev_entry != entry:
                stale.append((_API_KEY_SERVICE, prev_entry))
        for service, username in stale:
            try:
                self.keyring.delete_password(service, username)
            except PasswordDeleteError:
                pass
