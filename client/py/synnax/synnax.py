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

from .auth import AuthenticationClient
from .channel import ChannelClient
from .channel.create import ChannelCreator
from .channel.registry import ChannelRegistry
from .channel.retrieve import ChannelRetriever
from .config import load_options
from .framer import FramerClient
from .options import SynnaxOptions
from .transport import Transport


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
        opts = self.load_options(host, port, username, password, secure)
        self._transport = Transport(URL(host=opts.host, port=opts.port), opts.secure)
        if username != "" or password != "":
            auth = AuthenticationClient(
                self._transport.http.post_client(), opts.username, opts.password
            )
            auth.authenticate()
            self._transport.use(*auth.middleware())
        ch_retriever = ChannelRetriever(self._transport.http)
        ch_creator = ChannelCreator(self._transport.http)
        ch_registry = ChannelRegistry(ch_retriever)
        self.data = FramerClient(self._transport, ch_registry)
        self.channel = ChannelClient(self.data, ch_retriever, ch_creator)

    def load_options(
        self,
        host: str = "",
        port: int = 0,
        username: str = "",
        password: str = "",
        secure: bool = False,
    ) -> SynnaxOptions:
        if len(host) == 0:
            return load_options()
        return SynnaxOptions(
            host=host,
            port=port,
            username=username,
            password=password,
            secure=secure,
        )
