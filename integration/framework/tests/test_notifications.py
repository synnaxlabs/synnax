#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from console.notifications import NotificationsClient


class FakeToastsClient(NotificationsClient):
    def __init__(self) -> None:
        """Holds no page: the fake read never reaches one."""

    def _read(self) -> object:
        return [{"variant": "error", "message": "Failed", "description": "No Core"}]


class TestCheck:
    def test_should_return_each_notification_as_a_status(self) -> None:
        status = FakeToastsClient().check()[0]
        assert isinstance(status, sy.Status)
        assert status.variant == "error"
        assert status.message == "Failed"
        assert status.description == "No Core"
