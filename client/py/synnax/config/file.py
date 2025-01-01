#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import pathlib

CONFIG_DIR_NAME = "./synnax"


class ConfigFile:
    """The global synnax py configuration file."""

    file: pathlib.Path
    config: dict

    def __init__(
        self,
        config_dir: pathlib.Path,
        file_name: str = "config.json",
    ):
        self.config_file = config_dir / file_name
        self.config = {}
        self.load()

    def load(self):
        """Loads the config file from disk. If the file does not exist, it will
        be created.
        """
        if not self.config_file.exists():
            self.save()
        with open(self.config_file, "r") as f:
            self.config = json.load(f)

    def save(self):
        """Saves the config file to disk."""
        self.config_file.parent.mkdir(
            parents=True,
            exist_ok=True,
        )
        with open(self.config_file, "w") as f:
            json.dump(self.config, f)

    def get(self, key):
        """Gets a value from the config file."""
        return get_nested(self.config, key)

    def set(self, key, value):
        """Sets a value in the config file."""
        set_nested(self.config, key, value)
        self.save()

    def delete(self, key):
        """Deletes a value from the config file."""
        del self.config[key]
        self.save()


def set_nested(d, key, value):
    keys = key.split(".")
    for key in keys[:-1]:
        d = d.setdefault(key, {})
    d[keys[-1]] = value


def get_nested(d, key):
    keys = key.split(".")
    for key in keys[:-1]:
        d = d.get(key, {})
    return d.get(keys[-1])
