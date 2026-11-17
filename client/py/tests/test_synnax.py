#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
import sys
import threading
from collections.abc import Iterator

import pytest

import synnax as sy


@pytest.fixture
def fresh_client(login_info: tuple[str, int, str, str]) -> Iterator[sy.Synnax]:
    host, port, username, password = login_info
    client = sy.Synnax(host=host, port=port, username=username, password=password)
    yield client
    client.close()


class TestSubClients:
    def test_same_instance(self, fresh_client: sy.Synnax) -> None:
        """Should return the same sub-client on each access."""
        assert fresh_client.tasks is fresh_client.tasks

    def test_concurrent_first_access(self, fresh_client: sy.Synnax) -> None:
        """Should give every thread the same sub-client when threads race to build
        it.
        """
        count = 16
        barrier = threading.Barrier(count)
        got: list[object] = [None] * count

        def access(i: int) -> None:
            barrier.wait()
            got[i] = fresh_client.ranges

        threads = [threading.Thread(target=access, args=(i,)) for i in range(count)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        assert all(c is got[0] for c in got)

    @pytest.mark.skipif(
        sys.version_info < (3, 15), reason="lazy imports need Python 3.15"
    )
    def test_unused_modules_stay_unloaded(
        self, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should not load the module of a sub-client that the script never uses."""
        host, port, username, password = login_info
        script = (
            "import sys, synnax as sy\n"
            f"c = sy.Synnax(host={host!r}, port={port}, username={username!r}, "
            f"password={password!r})\n"
            "loaded = [m for m in ('synnax.schematic', 'synnax.ni') "
            "if m in sys.modules]\n"
            "c.close()\n"
            "print(loaded)\n"
        )
        result = subprocess.run(
            [sys.executable, "-c", script],
            capture_output=True,
            text=True,
            timeout=60,
        )
        assert result.returncode == 0, result.stderr
        assert result.stdout.strip() == "[]"
