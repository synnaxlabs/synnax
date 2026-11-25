#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import contextlib
import copy
import json
import os
import pathlib
import shutil
import sys
import tempfile
import time
from collections.abc import Sequence
from typing import Any

from synnax.exceptions import ConfigurationError


def read_latest(paths: Sequence[pathlib.Path]) -> tuple[pathlib.Path, bytes] | None:
    """Reads the file that was modified last of the paths that can be read. On a tie,
    the earlier path wins.

    :returns: The path and its bytes, or None when no file exists.
    :raises ConfigurationError: If a file exists but none can be read. The message
        names each path that failed and why.
    """
    # Synnax 0.62 writes only the legacy file, so the newer copy is the current one.
    # Read only the first path when 0.62 support ends.
    read: list[tuple[int, pathlib.Path, bytes]] = []
    failed: list[tuple[pathlib.Path, OSError]] = []
    for path in paths:
        try:
            read.append((path.stat().st_mtime_ns, path, path.read_bytes()))
        except FileNotFoundError:
            continue
        except OSError as e:
            failed.append((path, e))
    if read:
        _, path, data = max(read, key=lambda r: r[0])
        return path, data
    if not failed:
        return None
    raise ConfigurationError(
        " ".join(f"Cannot read {p}: {e.strerror}." for p, e in failed)
    ) from failed[-1][1]


class File:
    """The global synnax py configuration file. Loads the newer readable file of
    config_dir and legacy_dir. Writes to both.
    """

    config_file: pathlib.Path
    legacy_file: pathlib.Path | None
    config: dict[str, Any]

    def __init__(
        self,
        *,
        config_dir: pathlib.Path,
        legacy_dir: pathlib.Path | None = None,
        file_name: str = "config.json",
    ):
        self.config_file = config_dir / file_name
        self.legacy_file = None if legacy_dir is None else legacy_dir / file_name
        self.config = {}
        self.load()

    @property
    def _paths(self) -> list[pathlib.Path]:
        return [p for p in (self.config_file, self.legacy_file) if p is not None]

    def load(self) -> None:
        """Loads the config file from disk. A missing file loads as an empty config.

        :raises ConfigurationError: If no file can be read, or if the file read is not
            valid JSON.
        """
        read = read_latest(self._paths)
        if read is None:
            self.config = {}
            return
        path, data = read
        try:
            self.config = json.loads(data.decode("utf-8"))
        except (json.JSONDecodeError, UnicodeDecodeError) as e:
            raise ConfigurationError(
                f"{path} is not valid JSON: {e}. Delete it, then log in again with "
                "`sy login`."
            ) from e

    def save(self) -> None:
        """Saves the config file to disk."""
        self._write(self.config)

    def _write(self, config: dict[str, Any]) -> None:
        # Clients of Synnax 0.62 read only the legacy file. Drop it when 0.62 support
        # ends. A failed write puts back each file that it replaced.
        temps: list[str] = []
        backups: list[pathlib.Path] = []
        replaced: list[tuple[pathlib.Path, pathlib.Path | None]] = []
        try:
            staged = [_stage(path, config, temps) for path in self._paths]
            for temp, path in staged:
                backup = _backup(path)
                if backup is not None:
                    backups.append(backup)
                _replace(temp, path)
                replaced.append((path, backup))
        except BaseException as e:
            for path, backup in replaced:
                try:
                    if backup is None:
                        path.unlink()
                    else:
                        _replace(str(backup), path)
                except OSError as restore_err:
                    e.add_note(f"Cannot restore {path}: {restore_err.strerror}.")
            raise
        finally:
            # Once every file is replaced, the write is done, so a backup that cannot be
            # removed stays behind, and the next write replaces it.
            for leftover in [*map(pathlib.Path, temps), *backups]:
                with contextlib.suppress(OSError):
                    leftover.unlink(missing_ok=True)

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


def _stage(
    file: pathlib.Path, config: dict[str, Any], temps: list[str]
) -> tuple[str, pathlib.Path]:
    """Writes config to a temporary file beside file, adds it to temps, and returns
    it with the path it replaces.
    """
    # Resolving first writes through a symlink instead of replacing it.
    path = file.resolve()
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        "w",
        encoding="utf-8",
        dir=path.parent,
        prefix=f".{path.name}.",
        delete=False,
    ) as f:
        temps.append(f.name)
        json.dump(config, f)
        f.flush()
        os.fsync(f.fileno())
    if path.exists():
        shutil.copymode(path, f.name)
    return f.name, path


def _backup(path: pathlib.Path) -> pathlib.Path | None:
    """Copies path to a backup beside itself and returns the backup, or None if path
    does not exist.
    """
    if not path.exists():
        return None
    backup = path.with_name(f".{path.name}.bak")
    shutil.copy2(path, backup)
    return backup


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
