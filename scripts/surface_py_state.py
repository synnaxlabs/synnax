#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints the state the Python client saves on a computer, one sorted line each.

Compare the output of two revisions to see if a pull request changes what a client finds
on disk after an upgrade. The script saves a cluster with the real client code, then
prints each key of the config file with the type of its value, and each keyring entry.

    uv run --project scripts scripts/surface_py_state.py
"""

import json
import sys
from dataclasses import dataclass, field
from pathlib import Path
from tempfile import TemporaryDirectory
from typing import Any

from keyring.backend import KeyringBackend
from keyring.errors import PasswordDeleteError
from synnax.config import Cluster, Clusters, File
from synnax.options import Options


@dataclass
class MemoryKeyring(KeyringBackend):
    priority = 1
    entries: dict[tuple[str, str], str] = field(default_factory=dict)

    def get_password(self, service: str, username: str) -> str | None:
        return self.entries.get((service, username))

    def set_password(self, service: str, username: str, password: str) -> None:
        self.entries[(service, username)] = password

    def delete_password(self, service: str, username: str) -> None:
        if self.entries.pop((service, username), None) is None:
            raise PasswordDeleteError(username)


def flatten(value: Any, *, path: str) -> list[str]:
    if isinstance(value, dict):
        return [
            line for k, v in value.items() for line in flatten(v, path=f"{path}.{k}")
        ]
    return [f"{path} {type(value).__name__}"]


def report() -> str:
    options = Options(
        host="host",
        port=9090,
        username="username",
        password="password",
        api_key="api_key",
        secure=True,
        ca_cert="ca_cert",
    )
    keyring = MemoryKeyring()
    with TemporaryDirectory() as directory:
        file = File(config_dir=Path(directory))
        Clusters(file, keyring).set(Cluster(options=options))
        lines = flatten(
            json.loads(file.config_file.read_text()), path=file.config_file.name
        )
    lines += [f"keyring {service} {entry}" for service, entry in keyring.entries]
    return "".join(f"{line}\n" for line in sorted(lines))


def main() -> None:
    sys.stdout.write(report())


if __name__ == "__main__":
    main()
