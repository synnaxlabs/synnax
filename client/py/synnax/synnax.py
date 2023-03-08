#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import URL

from synnax.auth import AuthenticationClient
from synnax.channel import ChannelClient
from synnax.channel.create import ChannelCreator
from synnax.channel.retrieve import ClusterChannelRetriever, CacheChannelRetriever
from synnax.config import try_load_options_if_none_provided
from synnax.framer import FrameClient
from synnax.options import SynnaxOptions
from synnax.transport import Transport
from synnax.telem import TimeSpan


class Synnax(FrameClient):
    """Client to perform operations against a Synnax cluster.

    If using the client for data analysis/personal use, the easiest way to connect
    is to use the ``synnax login`` command, which will prompt and securely store your
    credentials. The client can then be initialized without parameters.

    After running the synnax login command::
        client = Synnax()

    Without running the synnax login command::
        client = Synnax(
            host="synnax.example.com",
            port=9090,
            username="synnax",
            password="seldon",
            secure=True,
        )
    """

    channels: ChannelClient

    _transport: Transport

    def __init__(
        self,
        host: str = "",
        port: int = 0,
        username: str = "",
        password: str = "",
        secure: bool = False,
        open_timeout: TimeSpan = TimeSpan.SECOND * 5,
        read_timeout: TimeSpan = TimeSpan.SECOND * 5,
        keep_alive: TimeSpan = TimeSpan.SECOND * 30,
        max_retries: int = 3,
    ):
        """Creates a new client. Connection parameters can be provided as arguments, or,
        if none are provided, the client will attempt to load them from the Synnax
        configuration file (~/.synnax/config.json) as well as credentials stored in the
        operating system's keychain.

        If using the client for data analysis/personal use, the easiest way to connect
        is to use the `synnax login` command, which will prompt and securely store your
        credentials. The client can then be initialized without parameters.

        :param host: Hostname of a node in the Synnax cluster.
        :param port: Port of the node.
        :param username: Username to authenticate with.
        :param password: Password to authenticate with.
        :param secure: Whether to use TLS when connnecting to the cluster.
        """
        opts = try_load_options_if_none_provided(host, port, username, password, secure)
        self._transport = self._configure_transport(
            opts=opts,
            open_timeout=open_timeout,
            read_timeout=read_timeout,
            keep_alive=keep_alive,
            max_retries=max_retries,
        )
        ch_retriever = CacheChannelRetriever(
            ClusterChannelRetriever(self._transport.http)
        )
        ch_creator = ChannelCreator(self._transport.http)
        super().__init__(self._transport, ch_retriever)
        self.channels = ChannelClient(self, ch_retriever, ch_creator)

    def close(self):
        """Shuts down the client and closes all connections. All open iterators or
        writers must be closed before calling this method.
        """
        # No-op for now, we'll definitely add cleanup logic in the future, so it's
        # good to have this API defined.
        ...

    def _configure_transport(
        self,
        opts: SynnaxOptions,
        open_timeout: TimeSpan,
        read_timeout: TimeSpan,
        keep_alive: TimeSpan,
        max_retries: int,
    ) -> Transport:
        t = Transport(
            url=URL(host=opts.host, port=opts.port),
            secure=opts.secure,
            open_timeout=open_timeout,
            read_timeout=read_timeout,
            keep_alive=keep_alive,
            max_retries=max_retries,
        )
        if opts.username != "" or opts.password != "":
            auth = AuthenticationClient(
                transport=t.http.post_client(),
                username=opts.username,
                password=opts.password,
            )
            auth.authenticate()
            t.use(*auth.middleware())
        return t
