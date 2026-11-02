#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

import pytest

from synnax.cli import connect, console
from synnax.cli.console.mock import Output
from synnax.cli.flow import Context
from synnax.options import Options


@pytest.mark.cli
class TestPromptClientOptions:
    def test_responses(self) -> None:
        """Should build options from the prompt responses."""
        c = console.Mock(responses=["127.0.0.1", 9091, "admin", "secret", True])
        assert connect.prompt_client_options(Context(console=c)) == Options(
            host="127.0.0.1",
            port=9091,
            username="admin",
            password="secret",
            secure=True,
        )

    def test_ca_cert(self) -> None:
        """Should store the absolute CA certificate path for a secure connection."""
        c = console.Mock(
            responses=["127.0.0.1", 9091, "admin", "secret", True, "certs/ca.crt"]
        )
        opts = connect.prompt_client_options(Context(console=c))
        assert opts.ca_cert == str(Path("certs/ca.crt").resolve())

    def test_insecure_skips_ca_cert(self) -> None:
        """Should not ask for a CA certificate for an insecure connection."""
        c = console.Mock(responses=["127.0.0.1", 9091, "admin", "secret", False])
        assert connect.prompt_client_options(Context(console=c)).ca_cert is None


@pytest.mark.cli
class TestFromContext:
    @pytest.mark.parametrize("contents", [None, "not a certificate"])
    def test_bad_ca_cert(self, tmp_path: Path, contents: str | None) -> None:
        """Should report a missing or invalid CA certificate instead of raising."""
        ca_cert = tmp_path / "ca.crt"
        if contents is not None:
            ca_cert.write_text(contents)
        output = Output()
        opts = Options(
            host="localhost",
            port=9090,
            username="synnax",
            password="seldon",
            secure=True,
            ca_cert=str(ca_cert),
        )
        assert (
            connect.from_context(
                Context(console=console.Mock(output=output)), opts=opts
            )
            is None
        )
        assert str(ca_cert) in (output.entries[-1].message or "")
