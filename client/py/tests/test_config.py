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
import struct
import subprocess
import sys
import threading
from collections.abc import Callable
from pathlib import Path
from typing import Any

import pytest

from freighter.exceptions import Unreachable
from synnax.config import (
    DESKTOP_NOT_RUNNING_MSG,
    KEYCHAIN_UNAVAILABLE_MSG,
    NO_CREDENTIALS_MSG,
    NO_KEY_HELPER_MSG,
    OLD_DESKTOP_MSG,
    Cluster,
    Clusters,
    File,
    load_desktop_options,
    load_env_options,
    load_options,
)
from synnax.exceptions import ConfigurationError, ValidationError
from synnax.options import Options
from tests.memory_keyring import MemoryKeyring, NoKeychain

HELPER = Path("/opt/synnax/synnax-desktop-key")


def write_desktop_file(path: Path, *, port: int, helper: Path = HELPER) -> None:
    """Saves the address and the key helper the way Synnax Desktop does."""
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps(
            {
                "host": "127.0.0.1",
                "port": port,
                "secure": False,
                "api_key_helper": str(helper),
            }
        ),
        encoding="utf-8",
    )


def write_helper(directory: Path) -> Path:
    """Writes a key helper that prints syk_k_s, and returns its path."""
    if sys.platform == "win32":
        helper = directory / "synnax-desktop-key.bat"
        helper.write_text("@echo syk_k_s\r\n", encoding="utf-8")
    else:
        helper = directory / "synnax-desktop-key"
        helper.write_text("#!/bin/sh\necho syk_k_s\n", encoding="utf-8")
        helper.chmod(0o755)
    return helper


class FakeHelper:
    """Stands in for the key helper of Synnax Desktop and records where it ran."""

    def __init__(
        self, returncode: int = 0, stdout: str = "syk_k_s\n", stderr: str = ""
    ) -> None:
        self.result = (returncode, stdout, stderr)
        self.paths: list[Path] = []

    def __call__(self, path: Path) -> subprocess.CompletedProcess[str]:
        self.paths.append(path)
        return subprocess.CompletedProcess([str(path)], *self.result)


def time_out(path: Path) -> subprocess.CompletedProcess[str]:
    raise subprocess.TimeoutExpired(str(path), 60)


@pytest.fixture
def clusters(tmp_path: Path) -> Clusters:
    return Clusters(File(config_dir=tmp_path), MemoryKeyring())


@pytest.fixture
def desktop_file(tmp_path: Path) -> Path:
    return tmp_path / "desktop.json"


class TestLoadDesktopOptions:
    def test_reads_the_file_and_the_helper(self, desktop_file: Path) -> None:
        """Should read the address from the file and the API key from the helper that
        the file names.
        """
        write_desktop_file(desktop_file, port=4321)
        helper = FakeHelper()
        assert load_desktop_options(desktop_file, helper) == Options(
            host="127.0.0.1", port=4321, secure=False, api_key="syk_k_s"
        )
        assert helper.paths == [HELPER]

    def test_runs_a_real_helper(self, desktop_file: Path, tmp_path: Path) -> None:
        """Should run the helper program and read the key that it prints."""
        write_desktop_file(desktop_file, port=4321, helper=write_helper(tmp_path))
        opts = load_desktop_options(desktop_file)
        assert opts is not None
        assert opts.api_key == "syk_k_s"

    def test_reads_utf8_text(self, desktop_file: Path) -> None:
        """Should decode the file as UTF-8 whatever the locale code page is."""
        cert = "C:\\Users\\Zoë\\ca.pem"
        desktop_file.parent.mkdir(parents=True, exist_ok=True)
        desktop_file.write_bytes(
            json.dumps(
                {
                    "host": "127.0.0.1",
                    "port": 4321,
                    "secure": True,
                    "ca_cert": cert,
                    "api_key_helper": str(HELPER),
                },
                ensure_ascii=False,
            ).encode("utf-8")
        )
        opts = load_desktop_options(desktop_file, FakeHelper())
        assert opts is not None
        assert opts.ca_cert == cert

    def test_none_without_a_file(self, desktop_file: Path) -> None:
        """Should return None when Synnax Desktop saved nothing."""
        assert load_desktop_options(desktop_file, FakeHelper()) is None

    def test_unreadable_file(self, desktop_file: Path) -> None:
        """Should name the file and the reason when the file exists but cannot be
        read.
        """
        # A directory at the path cannot be read as a file, on any platform or user.
        desktop_file.mkdir()
        with pytest.raises(ConfigurationError, match="Cannot read") as e:
            load_desktop_options(desktop_file, FakeHelper())
        assert str(desktop_file) in str(e.value)
        assert isinstance(e.value.__cause__, OSError)

    def test_older_file_with_a_key(self, desktop_file: Path) -> None:
        """Should ask for a newer Synnax Desktop, and not read a key from the file."""
        desktop_file.write_text(
            json.dumps(
                {"host": "127.0.0.1", "port": 1, "secure": False, "api_key": "syk_k"}
            ),
            encoding="utf-8",
        )
        with pytest.raises(ConfigurationError, match=OLD_DESKTOP_MSG):
            load_desktop_options(desktop_file, FakeHelper())

    def test_no_helper(self, desktop_file: Path) -> None:
        """Should ask for a newer Synnax Desktop when the file names no helper."""
        desktop_file.write_text(
            json.dumps({"host": "127.0.0.1", "port": 1, "secure": False}),
            encoding="utf-8",
        )
        with pytest.raises(ConfigurationError, match=NO_KEY_HELPER_MSG):
            load_desktop_options(desktop_file, FakeHelper())

    def test_missing_helper(self, desktop_file: Path, tmp_path: Path) -> None:
        """Should say that Synnax Desktop is not running when its helper is gone."""
        write_desktop_file(desktop_file, port=4321, helper=tmp_path / "missing")
        with pytest.raises(Unreachable, match=DESKTOP_NOT_RUNNING_MSG) as e:
            load_desktop_options(desktop_file)
        assert e.value.target == "127.0.0.1:4321"

    def test_helper_cannot_start(self, desktop_file: Path) -> None:
        """Should name the helper when it exists but cannot start."""
        write_desktop_file(desktop_file, port=4321)

        def deny(path: Path) -> subprocess.CompletedProcess[str]:
            raise PermissionError(13, "Permission denied")

        with pytest.raises(ConfigurationError, match="Cannot run the key helper") as e:
            load_desktop_options(desktop_file, deny)
        assert str(HELPER) in str(e.value)

    def test_failed_helper(self, desktop_file: Path) -> None:
        """Should give the error line of a helper that fails."""
        write_desktop_file(desktop_file, port=4321)
        helper = FakeHelper(1, "", "Synnax Desktop has saved no API key.\n")
        with pytest.raises(
            ConfigurationError,
            match="key helper of Synnax Desktop failed: Synnax Desktop has saved no",
        ):
            load_desktop_options(desktop_file, helper)

    def test_helper_with_no_key(self, desktop_file: Path) -> None:
        """Should fail when the helper exits with success and prints no key."""
        write_desktop_file(desktop_file, port=4321)
        with pytest.raises(ConfigurationError, match="printed no API key"):
            load_desktop_options(desktop_file, FakeHelper(stdout="\n"))

    def test_helper_times_out(self, desktop_file: Path) -> None:
        """Should fail with advice when the helper does not exit."""
        write_desktop_file(desktop_file, port=4321)
        with pytest.raises(ConfigurationError, match="did not exit"):
            load_desktop_options(desktop_file, time_out)


ENV = {"SYNNAX_HOST": "core.lab", "SYNNAX_API_KEY": "syk_env"}


class TestLoadEnvOptions:
    def test_empty_without_variables(self) -> None:
        """Should set nothing when no variable is set."""
        assert load_env_options({}) == {}

    def test_all_variables(self) -> None:
        """Should read every variable."""
        assert load_env_options(
            {
                **ENV,
                "SYNNAX_PORT": "9091",
                "SYNNAX_SECURE": "TRUE",
                "SYNNAX_CA_CERT": "~/ca.crt",
            }
        ) == {
            "host": "core.lab",
            "api_key": "syk_env",
            "port": 9091,
            "secure": True,
            "ca_cert": os.path.expanduser("~/ca.crt"),
        }

    def test_one_variable(self) -> None:
        """Should read one variable without the others."""
        assert load_env_options({"SYNNAX_API_KEY": "syk_env"}) == {"api_key": "syk_env"}

    def test_strips_values(self) -> None:
        """Should strip whitespace, such as the newline that ends a secret file."""
        assert load_env_options(
            {"SYNNAX_HOST": " core.lab\n", "SYNNAX_SECURE": "true "}
        ) == {"host": "core.lab", "secure": True}

    @pytest.mark.parametrize(
        "name",
        ["SYNNAX_HOST", "SYNNAX_PORT", "SYNNAX_API_KEY", "SYNNAX_SECURE"],
    )
    def test_blank_is_unset(self, name: str) -> None:
        """Should treat a value of only whitespace as unset."""
        assert load_env_options({name: "  "}) == {}

    @pytest.mark.parametrize(
        ("value", "secure"),
        [("1", True), ("yes", True), ("0", False), ("false", False), ("no", False)],
    )
    def test_secure_values(self, value: str, secure: bool) -> None:
        """Should read the true and false spellings of SYNNAX_SECURE."""
        assert load_env_options({"SYNNAX_SECURE": value}) == {"secure": secure}

    @pytest.mark.parametrize(
        ("env", "message"),
        [
            ({"SYNNAX_PORT": "abc"}, "SYNNAX_PORT must be a port"),
            ({"SYNNAX_PORT": "0"}, "SYNNAX_PORT must be a port"),
            ({"SYNNAX_PORT": "65536"}, "SYNNAX_PORT must be a port"),
            ({"SYNNAX_SECURE": "maybe"}, "SYNNAX_SECURE must be true or false"),
        ],
        ids=["port-text", "port-zero", "port-high", "secure"],
    )
    def test_invalid(self, env: dict[str, str], message: str) -> None:
        """Should name the variable that is wrong."""
        with pytest.raises(ValidationError, match=message):
            load_env_options(env)


SAVED = Options(
    host="saved.lab",
    port=1,
    api_key="syk_saved",
    secure=True,
    ca_cert="/saved/ca.crt",
)


class TestLoadOptions:
    @pytest.mark.parametrize(
        ("field", "arg", "env", "expected"),
        [
            ("host", "arg.lab", {"SYNNAX_HOST": "env.lab"}, "arg.lab"),
            ("host", None, {"SYNNAX_HOST": "env.lab"}, "env.lab"),
            ("port", 1, {"SYNNAX_PORT": "2"}, 1),
            ("port", None, {"SYNNAX_PORT": "2"}, 2),
            ("port", None, {}, 9090),
            ("api_key", "syk_arg", {"SYNNAX_API_KEY": "syk_env"}, "syk_arg"),
            ("api_key", None, {"SYNNAX_API_KEY": "syk_env"}, "syk_env"),
            ("secure", False, {"SYNNAX_SECURE": "true"}, False),
            ("secure", None, {"SYNNAX_SECURE": "true"}, True),
            ("secure", None, {}, False),
            ("ca_cert", "/arg.crt", {"SYNNAX_CA_CERT": "/env.crt"}, "/arg.crt"),
            ("ca_cert", None, {"SYNNAX_CA_CERT": "/env.crt"}, "/env.crt"),
            ("ca_cert", None, {}, None),
        ],
    )
    def test_field_order(
        self,
        clusters: Clusters,
        desktop_file: Path,
        field: str,
        arg: object,
        env: dict[str, str],
        expected: object,
    ) -> None:
        """Should take each field from the argument, then the variable, then the
        default.
        """
        args: dict[str, Any] = {"host": "h", "api_key": "syk_k"}
        args.pop(field, None)
        if arg is not None:
            args[field] = arg
        opts, _ = load_options(
            **args, env=env, clusters=clusters, desktop_file=desktop_file
        )
        assert getattr(opts, field) == expected

    def test_host_in_code_with_env_key(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should connect to the host in code with the key in SYNNAX_API_KEY."""
        opts, embedded = load_options(
            host="rig",
            port=9090,
            env={"SYNNAX_API_KEY": "syk_env"},
            clusters=clusters,
            desktop_file=desktop_file,
        )
        assert opts == Options(host="rig", port=9090, api_key="syk_env")
        assert not embedded

    def test_password_replaces_env_key(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should log in with the given username and password, not SYNNAX_API_KEY."""
        opts, _ = load_options(
            username="u",
            password="p",
            env=ENV,
            clusters=clusters,
            desktop_file=desktop_file,
        )
        assert (opts.username, opts.password, opts.api_key) == ("u", "p", "")

    @pytest.mark.parametrize(
        ("args", "env", "expected"),
        [
            ({}, {}, SAVED),
            ({"port": 2}, {}, SAVED.model_copy(update={"port": 2})),
            ({}, {"SYNNAX_PORT": "3"}, SAVED.model_copy(update={"port": 3})),
            (
                {"secure": False},
                {},
                SAVED.model_copy(update={"secure": False, "ca_cert": None}),
            ),
            (
                {},
                {"SYNNAX_SECURE": "false"},
                SAVED.model_copy(update={"secure": False, "ca_cert": None}),
            ),
            (
                {},
                {"SYNNAX_SECURE": "true", "SYNNAX_CA_CERT": "/env.crt"},
                SAVED.model_copy(update={"ca_cert": "/env.crt"}),
            ),
            (
                {},
                {"SYNNAX_CA_CERT": "/env.crt"},
                SAVED.model_copy(update={"ca_cert": "/env.crt"}),
            ),
            (
                {"ca_cert": "/arg.crt"},
                {},
                SAVED.model_copy(update={"ca_cert": "/arg.crt"}),
            ),
            (
                {"api_key": "syk_arg"},
                {},
                SAVED.model_copy(update={"api_key": "syk_arg"}),
            ),
            (
                {},
                {"SYNNAX_API_KEY": "syk_env"},
                SAVED.model_copy(update={"api_key": "syk_env"}),
            ),
            (
                {"username": "u", "password": "p"},
                {"SYNNAX_API_KEY": "syk_env"},
                SAVED.model_copy(
                    update={"api_key": "", "username": "u", "password": "p"}
                ),
            ),
        ],
        ids=[
            "saved",
            "port-arg",
            "port-env",
            "secure-arg",
            "secure-env",
            "tls-env",
            "ca-cert-env",
            "ca-cert-arg",
            "key-arg",
            "key-env",
            "password-arg",
        ],
    )
    def test_overrides_saved_core(
        self,
        clusters: Clusters,
        desktop_file: Path,
        args: dict[str, Any],
        env: dict[str, str],
        expected: Options,
    ) -> None:
        """Should let arguments and variables override the fields of the saved Core."""
        clusters.set(Cluster(options=SAVED))
        opts, _ = load_options(
            **args, env=env, clusters=clusters, desktop_file=desktop_file
        )
        assert opts == expected

    def test_password_login_replaced_by_env_key(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should drop the saved username and password for SYNNAX_API_KEY."""
        clusters.set(
            Cluster(options=Options(host="h", port=1, username="u", password="p"))
        )
        opts, _ = load_options(
            env={"SYNNAX_API_KEY": "syk_env"},
            clusters=clusters,
            desktop_file=desktop_file,
        )
        assert (opts.username, opts.password, opts.api_key) == ("", "", "syk_env")

    @pytest.mark.parametrize(
        ("args", "expected"),
        [
            ({"secure": True}, (True, None)),
            ({"secure": False}, (False, None)),
            ({"secure": True, "ca_cert": "/arg.crt"}, (True, "/arg.crt")),
            ({"ca_cert": "/arg.crt"}, (True, "/arg.crt")),
            ({}, (True, "/env.crt")),
        ],
        ids=["system-roots", "no-tls", "arg-cert", "arg-cert-alone", "env"],
    )
    def test_tls_from_one_source(
        self,
        clusters: Clusters,
        desktop_file: Path,
        args: dict[str, Any],
        expected: tuple[bool, str | None],
    ) -> None:
        """Should take secure and ca_cert together from the first source that sets
        one of them.
        """
        opts, _ = load_options(
            host="h",
            api_key="syk_k",
            **args,
            env={"SYNNAX_SECURE": "true", "SYNNAX_CA_CERT": "/env.crt"},
            clusters=clusters,
            desktop_file=desktop_file,
        )
        assert (opts.secure, opts.ca_cert) == expected

    @pytest.mark.parametrize(
        ("args", "env"),
        [
            ({"host": "h"}, {}),
            ({}, {"SYNNAX_HOST": "h"}),
            ({"host": "h", "password": "p"}, {}),
        ],
        ids=["host-arg", "host-env", "password-only"],
    )
    def test_no_credentials(
        self,
        clusters: Clusters,
        desktop_file: Path,
        args: dict[str, Any],
        env: dict[str, str],
    ) -> None:
        """Should explain how to log in when no source gives credentials."""
        with pytest.raises(ValidationError, match=NO_CREDENTIALS_MSG):
            load_options(**args, env=env, clusters=clusters, desktop_file=desktop_file)

    def test_given_options(self, clusters: Clusters, desktop_file: Path) -> None:
        """Should use the given options, which Synnax Desktop did not save."""
        write_desktop_file(desktop_file, port=4321)
        opts, embedded = load_options(
            host="h",
            port=1,
            api_key="syk_k",
            clusters=clusters,
            desktop_file=desktop_file,
        )
        assert (opts.host, opts.port, embedded) == ("h", 1, False)

    def test_given_desktop_options(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should know the options that Synnax Desktop saved when a caller gives them,
        as the CLI does.
        """
        write_desktop_file(desktop_file, port=4321)
        _, embedded = load_options(
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
        write_desktop_file(desktop_file, port=4321)
        _, embedded = load_options(
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
        opts, embedded = load_options(
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
        write_desktop_file(desktop_file, port=4321)
        opts, embedded = load_options(clusters=clusters, desktop_file=desktop_file)
        assert (opts.api_key, embedded) == ("syk_saved", False)

    def test_desktop_without_a_saved_login(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should use the Desktop file when `sy login` saved nothing."""
        write_desktop_file(desktop_file, port=4321)
        opts, embedded = load_options(
            clusters=clusters,
            desktop_file=desktop_file,
            run_desktop_helper=FakeHelper(),
        )
        assert (opts.port, opts.api_key, embedded) == (4321, "syk_k_s", True)

    def test_env_host_before_saved_login(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should not use a login or Synnax Desktop when SYNNAX_HOST is set."""
        clusters.set(Cluster(options=SAVED))
        write_desktop_file(desktop_file, port=4321)
        opts, embedded = load_options(
            env=ENV,
            clusters=clusters,
            desktop_file=desktop_file,
        )
        assert opts == Options(host="core.lab", port=9090, api_key="syk_env")
        assert not embedded

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
        write_desktop_file(desktop_file, port=4321)
        with pytest.raises(ConfigurationError, match=KEYCHAIN_UNAVAILABLE_MSG):
            load_options(
                clusters=Clusters(config, NoKeychain()), desktop_file=desktop_file
            )

    def test_desktop_without_a_keychain(
        self, tmp_path: Path, desktop_file: Path
    ) -> None:
        """Should get the key of Synnax Desktop from its helper when Python has no OS
        keychain.
        """
        write_desktop_file(desktop_file, port=4321)
        opts, _ = load_options(
            clusters=Clusters(File(config_dir=tmp_path), NoKeychain()),
            desktop_file=desktop_file,
            run_desktop_helper=FakeHelper(),
        )
        assert opts.api_key == "syk_k_s"

    @pytest.mark.parametrize(
        ("args", "env"),
        [
            ({}, {}),
            ({"port": 1, "api_key": "syk_k", "secure": True}, {}),
            ({}, {"SYNNAX_PORT": "1", "SYNNAX_API_KEY": "syk_k"}),
        ],
        ids=["nothing", "args-without-host", "env-without-host"],
    )
    def test_no_host(
        self,
        clusters: Clusters,
        desktop_file: Path,
        args: dict[str, Any],
        env: dict[str, str],
    ) -> None:
        """Should explain how to connect, without naming Synnax Desktop, when no source
        gives a host and Synnax Desktop saved no file.
        """
        with pytest.raises(ValidationError) as e:
            load_options(**args, env=env, clusters=clusters, desktop_file=desktop_file)
        assert str(e.value) == (
            "No host was given and no Core is saved. Give a host, set SYNNAX_HOST, or "
            "log in with `sy login`."
        )


class TestDesktopConnection:
    def test_not_running(self, tmp_path: Path) -> None:
        """Should say that Synnax Desktop is not running when its Core is gone."""
        with socket.socket() as s:
            s.bind(("127.0.0.1", 0))
            port = int(s.getsockname()[1])
        write_desktop_file(
            tmp_path / ".synnax" / "desktop.json",
            port=port,
            helper=write_helper(tmp_path),
        )
        # A fresh interpreter reads the home directory of the test.
        env = {
            **{k: v for k, v in os.environ.items() if not k.startswith("SYNNAX_")},
            "HOME": str(tmp_path),
            "USERPROFILE": str(tmp_path),
        }
        script = "import synnax as sy\nsy.Synnax(max_retries=0)\n"
        result = subprocess.run(
            [sys.executable, "-c", script],
            env=env,
            capture_output=True,
            text=True,
            timeout=30,
        )
        assert result.returncode != 0
        assert DESKTOP_NOT_RUNNING_MSG in result.stderr

    def test_reset(self, tmp_path: Path) -> None:
        """Should not say that Synnax Desktop is not running when its Core accepts
        the connection and then resets it.
        """
        listener = socket.create_server(("127.0.0.1", 0))

        def reset() -> None:
            conn, _ = listener.accept()
            with conn:
                conn.recv(1 << 16)
                conn.setsockopt(
                    socket.SOL_SOCKET, socket.SO_LINGER, struct.pack("ii", 1, 0)
                )

        thread = threading.Thread(target=reset, daemon=True)
        thread.start()
        write_desktop_file(
            tmp_path / ".synnax" / "desktop.json",
            port=int(listener.getsockname()[1]),
            helper=write_helper(tmp_path),
        )
        env = {
            **{k: v for k, v in os.environ.items() if not k.startswith("SYNNAX_")},
            "HOME": str(tmp_path),
            "USERPROFILE": str(tmp_path),
        }
        with listener:
            result = subprocess.run(
                [sys.executable, "-c", "import synnax as sy; sy.Synnax(max_retries=0)"],
                env=env,
                capture_output=True,
                text=True,
                timeout=30,
            )
            thread.join(5)
        assert result.returncode != 0
        assert "Unreachable" in result.stderr
        assert DESKTOP_NOT_RUNNING_MSG not in result.stderr


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
