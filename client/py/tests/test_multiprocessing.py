#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import multiprocessing
import pickle

import pytest

import synnax as sy

_START_METHODS = [
    m for m in ("spawn", "forkserver") if m in multiprocessing.get_all_start_methods()
]


def _check(login_info: tuple[str, int, str, str]) -> str:
    host, port, username, password = login_info
    client = sy.Synnax(host=host, port=port, username=username, password=password)
    try:
        return client.connectivity.check().status
    finally:
        client.close()


class TestMultiprocessing:
    """The client in worker processes."""

    @pytest.mark.parametrize("method", _START_METHODS)
    def test_worker_creates_client(
        self, method: str, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should connect from a client that each worker creates."""
        ctx = multiprocessing.get_context(method)
        with ctx.Pool(2) as pool:
            statuses = pool.map(_check, [login_info, login_info], chunksize=1)
        assert statuses == ["connected", "connected"]

    def test_client_not_picklable(self, client: sy.Synnax) -> None:
        """Should refuse to pickle, so a client cannot go to a worker."""
        with pytest.raises((TypeError, AttributeError, pickle.PicklingError)):
            pickle.dumps(client)
