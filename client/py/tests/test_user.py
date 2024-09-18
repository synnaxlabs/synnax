#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
import synnax as sy
from uuid import UUID


@pytest.mark.user
class TestUserClient:

    def test_create(self, client: sy.Synnax):
        sy.NewUser(username="test", password="test")
        user = client.user.create(user=(username="test", password="test"))
        assert user.username == "test"
        assert user.key is not None
