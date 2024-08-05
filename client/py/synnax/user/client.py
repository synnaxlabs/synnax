#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import UnaryClient
from synnax.auth import InsecureCredentials, TokenResponse
from synnax.user.payload import UserPayload


class UserClient:
    _REGISTER_ENDPOINT = "/user/register"

    client: UnaryClient

    def __init__(
        self,
        transport: UnaryClient,
    ) -> None:
        self.client = transport

    def register(self, username: str, password: str) -> UserPayload:
        res, exc = self.client.send(
            self._REGISTER_ENDPOINT,
            InsecureCredentials(username=username, password=password),
            TokenResponse,
        )
        if exc is not None:
            raise exc
        assert res is not None
        return res.user
