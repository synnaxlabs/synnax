#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import UnaryClient
from synnax.opcua.server.client import Client as ServerClient


class Client:
    """Groups the clients of the OPC UA resources."""

    servers: ServerClient

    def __init__(self, *, client: UnaryClient) -> None:
        self.servers = ServerClient(client=client)
