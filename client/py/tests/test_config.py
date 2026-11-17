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
import socket
import subprocess
import sys
from collections.abc import Callable
from pathlib import Path

import pytest

from synnax.config import (
    DESKTOP_KEY_ACCOUNT,
    DESKTOP_KEY_SERVICE,
    DESKTOP_NOT_RUNNING_MSG,
    KEYCHAIN_UNAVAILABLE_MSG,
    NO_OPTIONS_MSG,
    OLD_DESKTOP_MSG,
    Cluster,
    Clusters,
    File,
    Keyring,
    load_desktop_options,
    load_env_options,
    try_load_options_if_none_provided,
)
from synnax.exceptions import ConfigurationError, ValidationError
from synnax.options import Options
from tests.memory_keyring import MemoryKeyring, NoKeychain


def write_desktop_file(
    path: Path, keychain: Keyring, *, port: int, api_key: str = "syk_k_s"
) -> None:
    """Saves the address and the API key the way Synnax Desktop does."""
    keychain.set_password(DESKTOP_KEY_SERVICE, DESKTOP_KEY_ACCOUNT, api_key)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps({"host": "127.0.0.1", "port": port, "secure": False}),
        encoding="utf-8",
    )


@pytest.fixture
def clusters(tmp_path: Path) -> Clusters:
    return Clusters(File(config_dir=tmp_path), MemoryKeyring())


@pytest.fixture
def desktop_file(tmp_path: Path) -> Path:
    return tmp_path / "desktop.json"


class TestLoadDesktopOptions:
    def test_reads_the_file_and_the_keychain(self, desktop_file: Path) -> None:
        """Should read the address from the file and the API key from the keychain."""
        keychain = MemoryKeyring()
        write_desktop_file(desktop_file, keychain, port=4321)
        assert load_desktop_options(keychain, desktop_file) == Options(
            host="127.0.0.1", port=4321, secure=False, api_key="syk_k_s"
        )

    def test_reads_utf8_text(self, desktop_file: Path) -> None:
        """Should decode the file as UTF-8 whatever the locale code page is."""
        cert = "C:\\Users\\Zoë\\ca.pem"
        keychain = MemoryKeyring()
        keychain.set_password(DESKTOP_KEY_SERVICE, DESKTOP_KEY_ACCOUNT, "syk_k_s")
        desktop_file.parent.mkdir(parents=True, exist_ok=True)
        desktop_file.write_bytes(
            json.dumps(
                {"host": "127.0.0.1", "port": 4321, "secure": True, "ca_cert": cert},
                ensure_ascii=False,
            ).encode("utf-8")
        )
        opts = load_desktop_options(keychain, desktop_file)
        assert opts is not None
        assert opts.ca_cert == cert

    def test_none_without_a_file(self, desktop_file: Path) -> None:
        """Should return None when Synnax Desktop saved nothing."""
        assert load_desktop_options(MemoryKeyring(), desktop_file) is None

    def test_none_without_a_key(self, desktop_file: Path) -> None:
        """Should return None when the keychain holds no key for the address."""
        write_desktop_file(desktop_file, MemoryKeyring(), port=4321)
        assert load_desktop_options(MemoryKeyring(), desktop_file) is None

    def test_older_file_with_a_key(self, desktop_file: Path) -> None:
        """Should ask for a newer Synnax Desktop, and not read a key from the file."""
        desktop_file.write_text(
            json.dumps(
                {"host": "127.0.0.1", "port": 1, "secure": False, "api_key": "syk_k"}
            ),
            encoding="utf-8",
        )
        with pytest.raises(ConfigurationError, match=OLD_DESKTOP_MSG):
            load_desktop_options(MemoryKeyring(), desktop_file)

    def test_no_keychain(self, desktop_file: Path) -> None:
        """Should fail with advice when no OS keychain is available."""
        write_desktop_file(desktop_file, MemoryKeyring(), port=4321)
        with pytest.raises(ConfigurationError, match=KEYCHAIN_UNAVAILABLE_MSG):
            load_desktop_options(NoKeychain(), desktop_file)


ENV = {"SYNNAX_HOST": "core.lab", "SYNNAX_API_KEY": "syk_env"}


class TestLoadEnvOptions:
    def test_none_without_variables(self) -> None:
        """Should return None when neither SYNNAX_HOST nor SYNNAX_API_KEY is set."""
        assert load_env_options({"SYNNAX_PORT": "1"}) is None

    def test_defaults(self) -> None:
        """Should use port 9090 and no TLS when only the host and key are set."""
        assert load_env_options(ENV) == Options(
            host="core.lab", port=9090, api_key="syk_env"
        )

    def test_all_variables(self) -> None:
        """Should read the port, TLS, and CA certificate."""
        opts = load_env_options(
            {
                **ENV,
                "SYNNAX_PORT": "9091",
                "SYNNAX_SECURE": "TRUE",
                "SYNNAX_CA_CERT": "~/ca.crt",
            }
        )
        assert opts is not None
        assert (opts.port, opts.secure) == (9091, True)
        assert opts.ca_cert == str(Path("~/ca.crt").expanduser())

    def test_strips_values(self) -> None:
        """Should strip whitespace, such as the newline that ends a secret file."""
        opts = load_env_options(
            {
                "SYNNAX_HOST": " core.lab\n",
                "SYNNAX_API_KEY": "syk_env\n",
                "SYNNAX_SECURE": "true ",
            }
        )
        assert opts == Options(
            host="core.lab", port=9090, api_key="syk_env", secure=True
        )

    def test_empty_port_is_default(self) -> None:
        """Should use port 9090 when SYNNAX_PORT is set but empty."""
        opts = load_env_options({**ENV, "SYNNAX_PORT": ""})
        assert opts is not None
        assert opts.port == 9090

    def test_blank_host_is_unset(self) -> None:
        """Should treat a host of only whitespace as unset."""
        assert load_env_options({"SYNNAX_HOST": "  "}) is None

    @pytest.mark.parametrize(
        ("value", "secure"),
        [("1", True), ("yes", True), ("0", False), ("false", False), ("", False)],
    )
    def test_secure_values(self, value: str, secure: bool) -> None:
        """Should read the true and false spellings of SYNNAX_SECURE."""
        opts = load_env_options({**ENV, "SYNNAX_SECURE": value})
        assert opts is not None
        assert opts.secure is secure

    @pytest.mark.parametrize(
        ("env", "message"),
        [
            ({"SYNNAX_HOST": "h"}, "set SYNNAX_API_KEY with SYNNAX_HOST"),
            ({"SYNNAX_API_KEY": "k"}, "set SYNNAX_HOST with SYNNAX_API_KEY"),
            ({**ENV, "SYNNAX_PORT": "abc"}, "SYNNAX_PORT must be a port"),
            ({**ENV, "SYNNAX_PORT": "0"}, "SYNNAX_PORT must be a port"),
            ({**ENV, "SYNNAX_PORT": "65536"}, "SYNNAX_PORT must be a port"),
            ({**ENV, "SYNNAX_SECURE": "maybe"}, "SYNNAX_SECURE must be true or false"),
        ],
        ids=["host-only", "key-only", "port-text", "port-zero", "port-high", "secure"],
    )
    def test_invalid(self, env: dict[str, str], message: str) -> None:
        """Should name the variable that is wrong."""
        with pytest.raises(ValidationError, match=message):
            load_env_options(env)


class TestTryLoadOptions:
    def test_given_options(self, clusters: Clusters, desktop_file: Path) -> None:
        """Should use the given options, which Synnax Desktop did not save."""
        write_desktop_file(desktop_file, clusters.keyring, port=4321)
        opts, embedded = try_load_options_if_none_provided(
            host="h", port=1, clusters=clusters, desktop_file=desktop_file
        )
        assert (opts.host, opts.port, embedded) == ("h", 1, False)

    def test_given_desktop_options(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should know the options that Synnax Desktop saved when a caller gives them,
        as the CLI does.
        """
        write_desktop_file(desktop_file, clusters.keyring, port=4321)
        _, embedded = try_load_options_if_none_provided(
            host="127.0.0.1",
            port=4321,
            api_key="syk_k_s",
            clusters=clusters,
            desktop_file=desktop_file,
        )
        assert embedded

    def test_given_desktop_address_without_a_keychain(
        self, clusters: Clusters, desktop_file: Path, tmp_path: Path
    ) -> None:
        """Should know the Synnax Desktop address from the file alone, with no keychain
        and a key that differs from the saved one.
        """
        write_desktop_file(desktop_file, clusters.keyring, port=4321)
        _, embedded = try_load_options_if_none_provided(
            host="127.0.0.1",
            port=4321,
            api_key="syk_other_s",
            clusters=Clusters(File(config_dir=tmp_path / "empty"), NoKeychain()),
            desktop_file=desktop_file,
        )
        assert embedded

    @pytest.mark.parametrize(
        "break_file",
        # A directory at the path cannot be read as a file, on any platform or user.
        [lambda path: path.write_text("not json"), Path.mkdir],
        ids=["malformed", "unreadable"],
    )
    def test_given_options_with_a_broken_desktop_file(
        self,
        clusters: Clusters,
        desktop_file: Path,
        break_file: Callable[[Path], object],
    ) -> None:
        """Should connect with the given options when the Desktop file is malformed or
        cannot be read.
        """
        break_file(desktop_file)
        opts, embedded = try_load_options_if_none_provided(
            host="h",
            port=1,
            api_key="syk_k_s",
            clusters=clusters,
            desktop_file=desktop_file,
        )
        assert (opts.host, embedded) == ("h", False)

    def test_saved_login_before_desktop(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should use the login that `sy login` saved before the Desktop file."""
        clusters.set(Cluster(options=Options(host="h", port=1, api_key="syk_saved")))
        write_desktop_file(desktop_file, clusters.keyring, port=4321)
        opts, embedded = try_load_options_if_none_provided(
            clusters=clusters, desktop_file=desktop_file
        )
        assert (opts.api_key, embedded) == ("syk_saved", False)

    def test_desktop_without_a_saved_login(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should use the Desktop file when `sy login` saved nothing."""
        write_desktop_file(desktop_file, clusters.keyring, port=4321)
        opts, embedded = try_load_options_if_none_provided(
            clusters=clusters, desktop_file=desktop_file
        )
        assert (opts.port, opts.api_key, embedded) == (4321, "syk_k_s", True)

    def test_given_options_before_env(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should use the given options and ignore the environment variables."""
        opts, _ = try_load_options_if_none_provided(
            host="h", port=1, env=ENV, clusters=clusters, desktop_file=desktop_file
        )
        assert (opts.host, opts.api_key) == ("h", "")

    def test_env_before_saved_login(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should use the environment variables before a login and Synnax Desktop."""
        clusters.set(Cluster(options=Options(host="h", port=1, api_key="syk_saved")))
        write_desktop_file(desktop_file, clusters.keyring, port=4321)
        opts, embedded = try_load_options_if_none_provided(
            env=ENV, clusters=clusters, desktop_file=desktop_file
        )
        assert (opts.host, opts.api_key, embedded) == ("core.lab", "syk_env", False)

    def test_saved_login_without_a_keychain(
        self, tmp_path: Path, desktop_file: Path
    ) -> None:
        """Should fail, and not connect to Synnax Desktop, when the keychain that holds
        a saved login is not available.
        """
        config = File(config_dir=tmp_path)
        Clusters(config, MemoryKeyring()).set(
            Cluster(options=Options(host="h", port=1, api_key="syk_saved"))
        )
        write_desktop_file(desktop_file, MemoryKeyring(), port=4321)
        with pytest.raises(ConfigurationError, match=KEYCHAIN_UNAVAILABLE_MSG):
            try_load_options_if_none_provided(
                clusters=Clusters(config, NoKeychain()), desktop_file=desktop_file
            )

    def test_desktop_without_a_keychain(
        self, tmp_path: Path, desktop_file: Path
    ) -> None:
        """Should fail with advice when Synnax Desktop saved a key and no OS keychain
        is available to read it.
        """
        write_desktop_file(desktop_file, MemoryKeyring(), port=4321)
        with pytest.raises(ConfigurationError, match=KEYCHAIN_UNAVAILABLE_MSG):
            try_load_options_if_none_provided(
                clusters=Clusters(File(config_dir=tmp_path), NoKeychain()),
                desktop_file=desktop_file,
            )

    def test_nothing_saved(self, clusters: Clusters, desktop_file: Path) -> None:
        """Should explain how to connect when nothing is saved."""
        with pytest.raises(ValidationError, match=NO_OPTIONS_MSG):
            try_load_options_if_none_provided(
                clusters=clusters, desktop_file=desktop_file
            )


class TestDesktopConnection:
    def test_not_running(self, tmp_path: Path) -> None:
        """Should say that Synnax Desktop is not running when its Core is gone."""
        with socket.socket() as s:
            s.bind(("127.0.0.1", 0))
            port = int(s.getsockname()[1])
        write_desktop_file(
            tmp_path / ".synnax" / "desktop.json", MemoryKeyring(), port=port
        )
        # A fresh interpreter reads the home directory of the test, with a key in a
        # keychain that is not the one of the operating system.
        env = {
            **{k: v for k, v in os.environ.items() if not k.startswith("SYNNAX_")},
            "HOME": str(tmp_path),
            "USERPROFILE": str(tmp_path),
        }
        # keyring.set_keyring accepts only a KeyringBackend, which this script defines.
        script = (
            "import keyring\n"
            "from keyring.backend import KeyringBackend\n"
            "class Keychain(KeyringBackend):\n"
            "    priority = 1\n"
            "    def get_password(self, service, username):\n"
            f"        if (service, username) == ({DESKTOP_KEY_SERVICE!r}, "
            f"{DESKTOP_KEY_ACCOUNT!r}):\n"
            "            return 'syk_k_s'\n"
            "    def set_password(self, service, username, password): pass\n"
            "    def delete_password(self, service, username): pass\n"
            "keyring.set_keyring(Keychain())\n"
            "import synnax as sy\n"
            "sy.Synnax(max_retries=0)\n"
        )
        result = subprocess.run(
            [sys.executable, "-c", script],
            env=env,
            capture_output=True,
            text=True,
            timeout=30,
        )
        assert result.returncode != 0
        assert DESKTOP_NOT_RUNNING_MSG in result.stderr


class TestEnvConnection:
    def test_synnax_reads_the_environment(self, tmp_path: Path) -> None:
        """Should connect to the Core that the SYNNAX_* variables name."""
        with socket.socket() as s:
            s.bind(("127.0.0.1", 0))
            port = int(s.getsockname()[1])
        env = {
            **{k: v for k, v in os.environ.items() if not k.startswith("SYNNAX_")},
            "HOME": str(tmp_path),
            "USERPROFILE": str(tmp_path),
            "SYNNAX_HOST": "127.0.0.1",
            "SYNNAX_PORT": str(port),
            "SYNNAX_API_KEY": "syk_env",
        }
        result = subprocess.run(
            [sys.executable, "-c", "import synnax as sy; sy.Synnax(max_retries=0)"],
            env=env,
            capture_output=True,
            text=True,
            timeout=30,
        )
        assert result.returncode != 0
        assert f"127.0.0.1:{port}" in result.stderr
