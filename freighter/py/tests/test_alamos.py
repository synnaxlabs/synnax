#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import Context, Role
from freighter import alamos


class TestAlamos:
    def test_middleware_propagation(self, instrumentation):
        """
        Should propagate the context to the server.
        """
        middleware = alamos.instrumentation_middleware(instrumentation)
        context = Context(protocol="http", target="test", role="client")
        res, exc = middleware(context, lambda c: (None, None))
        assert res is None
        assert exc is None
        assert "traceparent" in context.params
