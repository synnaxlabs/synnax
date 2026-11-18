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
import subprocess
from collections.abc import Callable, Mapping
from pathlib import Path
from typing import Any, TypedDict

import keyring

from freighter.exceptions import Unreachable
from synnax.config.clusters import (
    KEYCHAIN_UNAVAILABLE_MSG,
    Cluster,
    Clusters,
    Keyring,
)
from synnax.config.file import File
from synnax.exceptions import ConfigurationError, ValidationError
from synnax.options import Options

__all__ = [
    "KEYCHAIN_UNAVAILABLE_MSG",
    "OLD_DESKTOP_MSG",
    "Cluster",
    "Clusters",
    "File",
    "Keyring",
    "PartialOptions",
    "RunHelper",
    "default_clusters",
    "load_desktop_options",
    "load_env_options",
    "load_options",
]

CONFIG_FILE_PATH = Path(os.path.expanduser("~/.synnax"))
DESKTOP_FILE_PATH = CONFIG_FILE_PATH / "desktop.json"


def default_clusters() -> Clusters:
    """Returns the clusters saved in the user's config file and OS keyring."""
    return Clusters(File(config_dir=CONFIG_FILE_PATH), keyring.get_keyring())


OLD_DESKTOP_MSG = (
    "Synnax Desktop saved its API key in an older format. Update Synnax Desktop and "
    "open it."
)

DESKTOP_NOT_RUNNING_MSG = (
    "Synnax Desktop is not running. Open it and try again. To connect to another "
    "Core, give a host and a port."
)

NO_KEY_HELPER_MSG = (
    "Synnax Desktop saved no key helper. Update Synnax Desktop and open it."
)

# A locked keychain can make the key helper wait for a password.
KEY_HELPER_TIMEOUT = 60

RunHelper = Callable[[Path], subprocess.CompletedProcess[str]]
"""Runs the key helper of Synnax Desktop. Raises OSError if the helper cannot start,
and subprocess.TimeoutExpired if it does not exit in time.
"""


def _run_helper(path: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [str(path)],
        stdin=subprocess.DEVNULL,
        capture_output=True,
        encoding="utf-8",
        timeout=KEY_HELPER_TIMEOUT,
        check=False,
    )


def _read_desktop_file(path: Path) -> dict[str, Any] | None:
    try:
        text = path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return None
    except OSError as e:
        raise ConfigurationError(f"Cannot read {path}: {e.strerror}.") from e
    raw: dict[str, Any] = json.loads(text)
    return raw


def _read_key(helper: Path, run: RunHelper, target: str) -> str:
    try:
        result = run(helper)
    except FileNotFoundError as e:
        # The helper of an AppImage goes away with its mount when Synnax Desktop closes.
        raise Unreachable(target=target, message=DESKTOP_NOT_RUNNING_MSG) from e
    except OSError as e:
        raise ConfigurationError(
            f"Cannot run the key helper of Synnax Desktop at {helper}: {e.strerror}. "
            "Update Synnax Desktop and open it."
        ) from e
    except subprocess.TimeoutExpired as e:
        raise ConfigurationError(
            f"The key helper of Synnax Desktop did not exit in {KEY_HELPER_TIMEOUT} "
            "seconds. Unlock the keychain of your operating system and try again."
        ) from e
    if result.returncode != 0:
        line = result.stderr.strip() or f"exit code {result.returncode}"
        raise ConfigurationError(f"The key helper of Synnax Desktop failed: {line}")
    if not (api_key := result.stdout.strip()):
        raise ConfigurationError("The key helper of Synnax Desktop printed no API key.")
    return api_key


def load_desktop_options(
    path: Path = DESKTOP_FILE_PATH, run: RunHelper = _run_helper
) -> Options | None:
    """Loads the options that Synnax Desktop saved for scripts on this computer: the
    address in a file, and the API key that the key helper named in the file prints.

    :param path: The file Synnax Desktop writes.
    :param run: Runs the key helper.
    :returns: The options, or None if Synnax Desktop has saved none.
    :raises ConfigurationError: If the file exists but cannot be read, if it names no
        key helper, if the key helper fails, or if an older Synnax Desktop saved the
        key in the file.
    :raises Unreachable: If the key helper is gone, as an AppImage helper is after
        Synnax Desktop closes.
    """
    raw = _read_desktop_file(path)
    if raw is None:
        return None
    if "api_key" in raw:
        raise ConfigurationError(OLD_DESKTOP_MSG)
    if not (helper := raw.get("api_key_helper")):
        raise ConfigurationError(NO_KEY_HELPER_MSG)
    target = f"{raw.get('host')}:{raw.get('port')}"
    api_key = _read_key(Path(helper), run, target)
    return Options.model_validate({**raw, "api_key": api_key})


_TRUE = ("1", "true", "yes")
_FALSE = ("0", "false", "no")


class PartialOptions(TypedDict, total=False):
    """The connection options that one source sets. A source leaves out the options that
    it does not set.
    """

    host: str
    port: int
    username: str
    password: str
    api_key: str
    secure: bool
    ca_cert: str


# Values from env files and secrets often carry a trailing newline, and an empty value
# usually means an unset variable in Docker Compose or CI.
def _env(env: Mapping[str, str], name: str) -> str:
    return env.get(name, "").strip()


def load_env_options(env: Mapping[str, str]) -> PartialOptions:
    """Loads the options in the SYNNAX_HOST, SYNNAX_PORT, SYNNAX_API_KEY, SYNNAX_SECURE,
    and SYNNAX_CA_CERT environment variables. Values are stripped, and an empty value
    counts as unset.

    :param env: The environment variables, such as os.environ.
    :returns: The options that the variables set.
    :raises ValidationError: If SYNNAX_PORT or SYNNAX_SECURE is not valid.
    """
    opts = PartialOptions()
    if host := _env(env, "SYNNAX_HOST"):
        opts["host"] = host
    if api_key := _env(env, "SYNNAX_API_KEY"):
        opts["api_key"] = api_key
    if raw_port := _env(env, "SYNNAX_PORT"):
        try:
            port = int(raw_port)
        except ValueError:
            port = 0
        if not 0 < port < 65536:
            raise ValidationError(
                f"SYNNAX_PORT must be a port from 1 to 65535, not {raw_port!r}"
            )
        opts["port"] = port
    if raw_secure := _env(env, "SYNNAX_SECURE"):
        if raw_secure.lower() not in _TRUE + _FALSE:
            raise ValidationError(
                f"SYNNAX_SECURE must be true or false, not {raw_secure!r}"
            )
        opts["secure"] = raw_secure.lower() in _TRUE
    if ca_cert := _env(env, "SYNNAX_CA_CERT"):
        opts["ca_cert"] = str(Path(ca_cert).expanduser())
    return opts


def _saved_by_desktop(opts: Options, path: Path) -> bool:
    if not opts.api_key:
        return False
    try:
        raw = _read_desktop_file(path)
    except (ValueError, ConfigurationError):
        # The file changes only the advice of errors, so a broken one must not break
        # a connection that names its Core.
        return False
    # The address alone tells the Core, so the check runs no key helper.
    return raw is not None and (raw.get("host"), raw.get("port")) == (
        opts.host,
        opts.port,
    )


NO_OPTIONS_MSG = (
    "No host was given and no Core is saved. Give a host, set SYNNAX_HOST, or log "
    "in with `sy login`."
)

NO_CREDENTIALS_MSG = (
    "No credentials were given. Set SYNNAX_API_KEY, or give an API key or a username "
    "and password."
)

DEFAULT_PORT = 9090


def load_options(
    *,
    host: str = "",
    port: int = 0,
    username: str = "",
    password: str = "",
    secure: bool | None = None,
    ca_cert: str | None = None,
    api_key: str = "",
    env: Mapping[str, str] | None = None,
    clusters: Clusters | None = None,
    desktop_file: Path = DESKTOP_FILE_PATH,
    run_desktop_helper: RunHelper = _run_helper,
) -> tuple[Options, bool]:
    """Chooses each option from the first source that sets it: the argument, then its
    SYNNAX_* environment variable, then the default. When no argument or variable gives
    a host, the Core that `sy login` saved, or else the Core that Synnax Desktop saved,
    comes before the default. The credentials (api_key, username, password) and the TLS
    options (secure, ca_cert) each come whole from the first source that sets one of
    them. A ca_cert with no secure turns TLS on.

    :param env: The environment variables, such as os.environ. Defaults to none.
    :param clusters: The saved logins. Defaults to those of the current user.
    :param desktop_file: The file where Synnax Desktop saves its options.
    :param run_desktop_helper: Runs the key helper of Synnax Desktop.
    :returns: The options, and whether they connect to the Core of Synnax Desktop.
    :raises ValidationError: If the environment variables are not valid, or if no source
        gives a host or credentials.
    :raises ConfigurationError: If no OS keychain is available to read the key of the
        saved login, if the file of Synnax Desktop exists but cannot be read, if its key
        helper fails, or if an older Synnax Desktop left its key in the file.
    :raises Unreachable: If the key helper of Synnax Desktop is gone.
    """
    given = PartialOptions()
    if host:
        given["host"] = host
    if port:
        given["port"] = port
    if username:
        given["username"] = username
    if password:
        given["password"] = password
    if api_key:
        given["api_key"] = api_key
    if secure is not None:
        given["secure"] = secure
    if ca_cert is not None:
        given["ca_cert"] = ca_cert
    from_env = load_env_options(env or {})
    sources: list[Mapping[str, Any]] = [given, from_env]
    if "host" not in given and "host" not in from_env:
        if clusters is None:
            clusters = default_clusters()
        cluster = clusters.get()
        saved = (
            cluster.options
            if cluster is not None
            else load_desktop_options(desktop_file, run_desktop_helper)
        )
        if saved is None:
            raise ValidationError(NO_OPTIONS_MSG)
        sources.append(saved.model_dump())
    opts = Options.model_validate({"port": DEFAULT_PORT, **_merge(sources)})
    if not opts.api_key and not opts.username:
        raise ValidationError(NO_CREDENTIALS_MSG)
    return opts, _saved_by_desktop(opts, desktop_file)


# Each group comes whole from one source, so a password never pairs with another
# source's key, and a CA certificate never outlives the TLS choice of its source.
_GROUPS = (("api_key", "username", "password"), ("secure", "ca_cert"))


def _merge(sources: list[Mapping[str, Any]]) -> dict[str, Any]:
    merged: dict[str, Any] = {}
    for source in reversed(sources):
        merged.update(source)
    for group in _GROUPS:
        for k in group:
            merged.pop(k, None)
        source = next((s for s in sources if any(k in s for k in group)), {})
        merged.update({k: source[k] for k in group if k in source})
    # A CA certificate has a use only with TLS.
    if merged.get("ca_cert") is not None:
        merged.setdefault("secure", True)
    return merged
