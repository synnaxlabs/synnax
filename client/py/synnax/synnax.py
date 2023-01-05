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

from freighter import URL
from pydantic import BaseModel

from .auth import AuthenticationClient

from .channel import ChannelClient
from .channel.create import ChannelCreator
from .channel.registry import ChannelRegistry
from .channel.retrieve import ChannelRetriever
from .framer import FramerClient

from .transport import Transport


class SynnaxOptions(BaseModel):
    """Options class for the Synnax client."""

    host: str
    port: int
    username: str = ""
    password: str = ""
    secure: bool = False


class Synnax:
    """Client to perform operations against a Synnax cluster. If no credentials are provided
    in the options, the client will attempt to load them from the configuration file (
    ~/.synnax/config.json) or from environment variables.

    :param host: Hostname of a Synnax server.
    :param port: Port of a Synnax server.
    :param username: Username to authenticate with. Not required if the Synnax cluster
    is running in insecure mode.
    :param password: Password to authenticate with. Not required if the Synnax cluster
    is running in insecure mode.
    """

    _transport: Transport
    channel: ChannelClient
    data: FramerClient

    def __init__(
        self,
        host: str = "",
        port: int = 0,
        username: str = "",
        password: str = "",
        secure: bool = False,
    ):
        self._transport = Transport(URL(host=host, port=port), secure)
        if username != "" or password != "":
            auth = AuthenticationClient(
                self._transport.http.post_client(), username, password
            )
            auth.authenticate()
            self._transport.use(*auth.middleware())
        ch_retriever = ChannelRetriever(self._transport.http)
        ch_creator = ChannelCreator(self._transport.http)
        ch_registry = ChannelRegistry(ch_retriever)
        self.data = FramerClient(self._transport, ch_registry)
        self.channel = ChannelClient(self.data, ch_retriever, ch_creator)
