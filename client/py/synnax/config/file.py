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
import os
import pathlib
import shutil
import sys
import tempfile
import time
from typing import Any

from synnax.exceptions import ConfigurationError


class File:
    """The global synnax py configuration file. Writes go to config_dir. When
    config_dir holds no file, the config loads from fallback_dir, so the first write
    copies it.
    """

    config_file: pathlib.Path
    fallback_file: pathlib.Path | None
    config: dict[str, Any]

    def __init__(
        self,
        *,
        config_dir: pathlib.Path,
        fallback_dir: pathlib.Path | None = None,
        file_name: str = "config.json",
    ):
        self.config_file = config_dir / file_name
        self.fallback_file = None if fallback_dir is None else fallback_dir / file_name
        self.config = {}
        self.load()

    def load(self) -> None:
        """Loads the config file from disk. A missing file loads as an empty config.

        :raises ConfigurationError: If the file cannot be read or is not valid JSON.
        """
        self.config = {}
        for path in (self.config_file, self.fallback_file):
            if path is None:
                continue
            try:
                data = path.read_bytes()
            except FileNotFoundError:
                continue
            except OSError as e:
                raise ConfigurationError(f"Cannot read {path}: {e.strerror}.") from e
            try:
                self.config = json.loads(data.decode("utf-8"))
            except (json.JSONDecodeError, UnicodeDecodeError) as e:
                raise ConfigurationError(
                    f"{path} is not valid JSON: {e}. Delete it, then log in again "
                    "with `sy login`."
                ) from e
            return

    def save(self) -> None:
        """Saves the config file to disk."""
        self._write(self.config)

    def _write(self, config: dict[str, Any]) -> None:
        # Resolving first writes through a symlink instead of replacing it.
        path = self.config_file.resolve()
        path.parent.mkdir(parents=True, exist_ok=True)
        # Replacing the file whole keeps a failed write from leaving a partial file.
        with tempfile.NamedTemporaryFile(
            "w",
            encoding="utf-8",
            dir=path.parent,
            prefix=f".{path.name}.",
            delete_on_close=False,
        ) as f:
            json.dump(config, f)
            f.flush()
            os.fsync(f.fileno())
            f.close()
            if path.exists():
                shutil.copymode(path, f.name)
            _replace(f.name, path)

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


# Windows refuses to replace a file that another process, such as a virus scanner,
# holds open for a moment. On other platforms, a PermissionError is final.
_REPLACE_RETRY_DELAYS: tuple[float, ...] = (
    (0.01, 0.05, 0.1, 0.5) if sys.platform == "win32" else ()
)


def _replace(src: str, dst: pathlib.Path) -> None:
    for delay in _REPLACE_RETRY_DELAYS:
        try:
            os.replace(src, dst)
            return
        except PermissionError:
            time.sleep(delay)
    os.replace(src, dst)


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
