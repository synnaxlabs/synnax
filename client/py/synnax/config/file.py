#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import copy
import json
import pathlib
from typing import Any

_CONFIG_DIR_NAME = "./synnax"


class File:
    """The global synnax py configuration file."""

    file: pathlib.Path
    config: dict[str, Any]

    def __init__(
        self,
        *,
        config_dir: pathlib.Path,
        file_name: str = "config.json",
    ):
        self.config_file = config_dir / file_name
        self.config = {}
        self.load()

    def load(self) -> None:
        """Loads the config file from disk. If the file does not exist, it will
        be created.
        """
        if not self.config_file.exists():
            self.save()
        with open(self.config_file) as f:
            self.config = json.load(f)

    def save(self) -> None:
        """Saves the config file to disk."""
        self._write(self.config)

    def _write(self, config: dict[str, Any]) -> None:
        self.config_file.parent.mkdir(
            parents=True,
            exist_ok=True,
        )
        with open(self.config_file, "w") as f:
            json.dump(config, f)

    def get(self, key: str) -> Any:
        """Gets a value from the config file."""
        return _get_nested(self.config, key)

    def set(self, key: str, value: Any) -> None:
        """Sets a value in the config file. A failed write leaves the config as it
        was.
        """
        config = copy.deepcopy(self.config)
        _set_nested(config, key, value)
        self._write(config)
        self.config = config

    def delete(self, key: str) -> None:
        """Deletes a value from the config file."""
        del self.config[key]
        self.save()


def _set_nested(d: dict[str, Any], key: str, value: Any) -> None:
    keys = key.split(".")
    for key in keys[:-1]:
        d = d.setdefault(key, {})
    d[keys[-1]] = value


def _get_nested(d: dict[str, Any], key: str) -> Any:
    keys = key.split(".")
    for key in keys[:-1]:
        d = d.get(key, {})
    return d.get(keys[-1])
