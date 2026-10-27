#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

import freighter
import synnax as sy
from synnax.exceptions import create_registry
from x.exceptions import ExceptionPayload


class TestCreateRegistry:
    def test_freighter(self) -> None:
        """Should decode the errors of Freighter."""
        pld = ExceptionPayload(type=freighter.EOF.TYPE, data="EOF")
        assert isinstance(create_registry().decode(pld), freighter.EOF)

    def test_path_error(self) -> None:
        """Should decode the error that a path error wraps."""
        data = json.dumps(
            {
                "path": ["channel", "name"],
                "error": {"type": sy.NotFoundError.TYPE, "data": "missing"},
            }
        )
        err = create_registry().decode(
            ExceptionPayload(type=sy.PathError.TYPE, data=data)
        )
        assert isinstance(err, sy.PathError)
        assert err.path == ["channel", "name"]
        assert isinstance(err.error, sy.NotFoundError)
