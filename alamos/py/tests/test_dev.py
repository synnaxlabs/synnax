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

import pytest


class TestDev:
    def test_missing_uptrace(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Should name the extra to install when uptrace is missing."""
        monkeypatch.setitem(sys.modules, "uptrace", None)
        monkeypatch.delitem(sys.modules, "alamos.dev", raising=False)
        with pytest.raises(ImportError, match=r"alamos\[dev\]"):
            importlib.import_module("alamos.dev")
