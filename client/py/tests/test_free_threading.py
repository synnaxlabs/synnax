#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import importlib
import sys
import sysconfig

import pytest


@pytest.mark.skipif(
    not sysconfig.get_config_var("Py_GIL_DISABLED"),
    reason="needs a free-threaded build of Python",
)
def test_gil_stays_disabled_after_import() -> None:
    """Should keep the GIL off on a free-threaded build once synnax and its
    dependencies are imported. An extension module without free-threading support
    turns the GIL back on with only a warning.
    """
    importlib.import_module("synnax")
    assert not sys._is_gil_enabled()
