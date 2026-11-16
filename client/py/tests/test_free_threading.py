#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
import sysconfig

import pytest

import synnax  # noqa: F401


@pytest.mark.skipif(
    not sysconfig.get_config_var("Py_GIL_DISABLED"),
    reason="needs a free-threaded build of Python",
)
def test_gil_stays_disabled_after_import() -> None:
    """Should keep the GIL off on a free-threaded build once synnax and its
    dependencies are imported. An extension module without free-threading support
    turns the GIL back on with only a warning.
    """
    # mypy targets 3.12 until the floor rises, and its stubs add this only for 3.13.
    is_gil_enabled = vars(sys)["_is_gil_enabled"]
    assert not is_gil_enabled()
