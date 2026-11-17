#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
from collections.abc import Mapping
from pathlib import Path
from typing import Any

import keyring

from synnax.config.clusters import (
    KEYCHAIN_UNAVAILABLE_MSG,
    Cluster,
    Clusters,
    Keyring,
    _unavailable,
)
from synnax.config.file import File
from synnax.exceptions import ConfigurationError, ValidationError
from synnax.options import Options

__all__ = [
    "DESKTOP_KEY_ACCOUNT",
    "DESKTOP_KEY_SERVICE",
    "KEYCHAIN_UNAVAILABLE_MSG",
    "OLD_DESKTOP_MSG",
    "Cluster",
    "Clusters",
    "File",
    "Keyring",
    "default_clusters",
    "load_desktop_options",
    "load_env_options",
]

CONFIG_FILE_PATH = Path(os.path.expanduser("~/.synnax"))
DESKTOP_FILE_PATH = CONFIG_FILE_PATH / "desktop.json"


def default_clusters() -> Clusters:
    """Returns the clusters saved in the user's config file and OS keyring."""
    return Clusters(File(config_dir=CONFIG_FILE_PATH), keyring.get_keyring())


DESKTOP_KEY_SERVICE = "com.synnaxlabs.desktop"
DESKTOP_KEY_ACCOUNT = "api_key"

OLD_DESKTOP_MSG = (
    "Synnax Desktop saved its API key in an older format. Update Synnax Desktop and "
    "open it."
)


def _read_desktop_file(path: Path) -> dict[str, Any] | None:
    try:
        raw: dict[str, Any] = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        return None
    return raw


def load_desktop_options(
    keychain: Keyring, path: Path = DESKTOP_FILE_PATH
) -> Options | None:
    """Loads the options that Synnax Desktop saved for scripts on this computer: the
    address in a file, and the API key in the OS keychain.

    :param keychain: The keychain that holds the API key.
    :param path: The file Synnax Desktop writes.
    :returns: The options, or None if Synnax Desktop has saved none.
    :raises ConfigurationError: If no OS keychain is available, or if an older Synnax
        Desktop saved the key in the file.
    """
    raw = _read_desktop_file(path)
    if raw is None:
        return None
    if "api_key" in raw:
        raise ConfigurationError(OLD_DESKTOP_MSG)
    try:
        api_key = keychain.get_password(DESKTOP_KEY_SERVICE, DESKTOP_KEY_ACCOUNT)
    except Exception as e:
        if not _unavailable(e):
            raise
        raise ConfigurationError(
            "Cannot read the API key of Synnax Desktop. " + KEYCHAIN_UNAVAILABLE_MSG
        ) from e
    if api_key is None:
        return None
    return Options.model_validate({**raw, "api_key": api_key})


_TRUE = ("1", "true", "yes")
_FALSE = ("", "0", "false", "no")


# Values from env files and secrets often carry a trailing newline, and an empty value
# usually means an unset variable in Docker Compose or CI.
def _env(env: Mapping[str, str], name: str) -> str:
    return env.get(name, "").strip()


def load_env_options(env: Mapping[str, str]) -> Options | None:
    """Loads the options in the SYNNAX_HOST, SYNNAX_PORT, SYNNAX_API_KEY,
    SYNNAX_SECURE, and SYNNAX_CA_CERT environment variables.

    :param env: The environment variables, such as os.environ.
    :returns: The options, or None if SYNNAX_HOST and SYNNAX_API_KEY are not set.
    :raises ValidationError: If SYNNAX_HOST or SYNNAX_API_KEY is set without the
        other, or if SYNNAX_PORT or SYNNAX_SECURE is not valid. Values are stripped,
        and an empty value counts as unset.
    """
    host = _env(env, "SYNNAX_HOST")
    api_key = _env(env, "SYNNAX_API_KEY")
    if not host and not api_key:
        return None
    if not host:
        raise ValidationError("set SYNNAX_HOST with SYNNAX_API_KEY")
    if not api_key:
        raise ValidationError("set SYNNAX_API_KEY with SYNNAX_HOST")
    raw_port = _env(env, "SYNNAX_PORT") or "9090"
    try:
        port = int(raw_port)
    except ValueError:
        port = 0
    if not 0 < port < 65536:
        raise ValidationError(
            f"SYNNAX_PORT must be a port from 1 to 65535, not {raw_port!r}"
        )
    raw_secure = _env(env, "SYNNAX_SECURE")
    if raw_secure.lower() not in _TRUE + _FALSE:
        raise ValidationError(
            f"SYNNAX_SECURE must be true or false, not {raw_secure!r}"
        )
    ca_cert = _env(env, "SYNNAX_CA_CERT") or None
    return Options(
        host=host,
        port=port,
        api_key=api_key,
        secure=raw_secure.lower() in _TRUE,
        ca_cert=None if ca_cert is None else str(Path(ca_cert).expanduser()),
    )


def _saved_by_desktop(opts: Options, path: Path) -> bool:
    if not opts.api_key:
        return False
    try:
        raw = _read_desktop_file(path)
    except (ValueError, OSError):
        # The file changes only the advice of errors, so a broken one must not break
        # a connection that names its Core.
        return False
    # The address alone tells the Core, so the check needs no keychain.
    return raw is not None and (raw.get("host"), raw.get("port")) == (
        opts.host,
        opts.port,
    )


NO_OPTIONS_MSG = (
    "No connection parameters were given and none are saved. Open Synnax Desktop, "
    "log in with `sy login`, set SYNNAX_HOST and SYNNAX_API_KEY, or give a host and "
    "a port."
)

DESKTOP_NOT_RUNNING_MSG = (
    "Synnax Desktop is not running. Open it and try again. To connect to another "
    "Core, give a host and a port."
)


def try_load_options_if_none_provided(
    *,
    host: str = "",
    port: int = 0,
    username: str = "",
    password: str = "",
    secure: bool = False,
    ca_cert: str | None = None,
    api_key: str = "",
    env: Mapping[str, str] | None = None,
    clusters: Clusters | None = None,
    desktop_file: Path = DESKTOP_FILE_PATH,
) -> tuple[Options, bool]:
    """Returns the given options. Without a host, returns the options in the
    environment variables, then the options that `sy login` saved, then those that
    Synnax Desktop saved.

    :param env: The environment variables, such as os.environ. Defaults to none.
    :param clusters: The saved logins. Defaults to those of the current user.
    :param desktop_file: The file where Synnax Desktop saves its options.
    :returns: The options, and whether they are the options Synnax Desktop saved.
    :raises ValidationError: If an API key has no host, the environment variables are
        not valid, or nothing is saved.
    :raises ConfigurationError: If no OS keychain is available to read the key of the
        saved login or of Synnax Desktop, or if an older Synnax Desktop left its key in
        the discovery file.
    """
    if len(host) == 0:
        if api_key:
            raise ValidationError("give a host with an API key")
        from_env = load_env_options(env or {})
        if from_env is not None:
            return from_env, _saved_by_desktop(from_env, desktop_file)
        if clusters is None:
            clusters = default_clusters()
        cluster = clusters.get()
        if cluster is not None:
            return cluster.options, False
        opts = load_desktop_options(clusters.keyring, desktop_file)
        if opts is None:
            raise ValidationError(NO_OPTIONS_MSG)
        return opts, True
    opts = Options(
        host=host,
        port=port,
        username=username,
        password=password,
        api_key=api_key,
        secure=secure,
        ca_cert=ca_cert,
    )
    return opts, _saved_by_desktop(opts, desktop_file)
