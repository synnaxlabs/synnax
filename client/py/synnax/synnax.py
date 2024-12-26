#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import NOOP, Instrumentation
from freighter import URL
from synnax import PolicyClient

from synnax.auth import AuthenticationClient
from synnax.channel import ChannelClient
from synnax.channel.retrieve import CacheChannelRetriever, ClusterChannelRetriever
from synnax.channel.writer import ChannelWriter
from synnax.config import try_load_options_if_none_provided
from synnax.control import Client as ControlClient
from synnax.framer import Client
from synnax.framer.deleter import Deleter
from synnax.hardware.client import Client as HardwareClient
from synnax.hardware.task import Client as TaskClient
from synnax.hardware.device import Client as DeviceClient
from synnax.hardware.rack import Client as RackClient
from synnax.options import SynnaxOptions
from synnax.ranger import RangeRetriever, RangeWriter
from synnax.ranger.client import RangeClient
from synnax.signals.signals import Registry
from synnax.telem import TimeSpan
from synnax.transport import Transport
from synnax.user.client import Client as UserClient
from synnax.ontology import Client as OntologyClient
from synnax.ontology.group import Client as GroupClient


class Synnax(Client):
    """Client to perform operations against a Synnax cluster.

    If using the python client for data analysis/personal use, the easiest way to
    connect is to use the `synnax login` command, which will prompt and securely
    store your credentials. The client can then be initialized without parameters. When
    using the client in a production environment, it's best to provide the connection
    parameter as arguments loaded from a configuration or environment variable.

    After running the synnax login command::
        client = Synnax()

    Without running the synnax login command::
        client = Synnax(
            host="synnax.example.com",
            port=9090,
            username="synnax",
            password="seldon",
            secure=True
        )
    """

    channels: ChannelClient
    access: PolicyClient
    user: UserClient
    ranges: RangeClient
    control: ControlClient
    signals: Registry
    hardware: HardwareClient
    ontology: OntologyClient

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
        instrumentation: Instrumentation = NOOP,
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
        :param secure: Whether to use TLS when connecting to the cluster.
        """
        opts = try_load_options_if_none_provided(host, port, username, password, secure)
        self._transport = _configure_transport(
            opts=opts,
            open_timeout=open_timeout,
            read_timeout=read_timeout,
            keep_alive=keep_alive,
            max_retries=max_retries,
            instrumentation=instrumentation,
        )
        if opts.username != "" or opts.password != "":
            self.auth = AuthenticationClient(
                transport=self._transport.unary,
                username=opts.username,
                password=opts.password,
            )
            self.auth.authenticate()
            self._transport.use(*self.auth.middleware())
            self._transport.use_async(*self.auth.async_middleware())

        ch_retriever = CacheChannelRetriever(
            ClusterChannelRetriever(self._transport.unary, instrumentation),
            instrumentation,
        )
        deleter = Deleter(self._transport.unary, instrumentation)
        ch_creator = ChannelWriter(
            self._transport.unary,
            instrumentation,
            ch_retriever,
        )
        super().__init__(
            stream_client=self._transport.stream,
            async_client=self._transport.stream_async,
            unary_client=self._transport.unary,
            retriever=ch_retriever,
            deleter=deleter,
            instrumentation=instrumentation,
        )
        groups = GroupClient(self._transport.unary)
        self.ontology = OntologyClient(client=self._transport.unary, groups=groups)
        self.channels = ChannelClient(self, ch_retriever, ch_creator)
        range_retriever = RangeRetriever(self._transport.unary, instrumentation)
        range_creator = RangeWriter(self._transport.unary, instrumentation)
        self.signals = Registry(frame_client=self, channels=ch_retriever)
        racks = RackClient(client=self._transport.unary)
        tasks = TaskClient(
            client=self._transport.unary, frame_client=self, rack_client=racks
        )
        devices = DeviceClient(client=self._transport.unary)
        self.ranges = RangeClient(
            unary_client=self._transport.unary,
            frame_client=self,
            channel_retriever=ch_retriever,
            writer=range_creator,
            retriever=range_retriever,
            signals=self.signals,
            ontology=self.ontology,
            tasks=tasks,
        )
        self.control = ControlClient(self, ch_retriever)

        self.hardware = HardwareClient(tasks=tasks, devices=devices, racks=racks)
        self.access = PolicyClient(self._transport.unary, instrumentation)
        self.user = UserClient(self._transport.unary)

    def close(self):
        """Shuts down the client and closes all connections. All open iterators or
        writers must be closed before calling this method.
        """
        # No-op for now, we'll definitely add cleanup logic in the future, so it's
        # good to have this API defined.
        ...


def _configure_transport(
    opts: SynnaxOptions,
    open_timeout: TimeSpan,
    read_timeout: TimeSpan,
    keep_alive: TimeSpan,
    max_retries: int,
    instrumentation: Instrumentation = NOOP,
) -> Transport:
    t = Transport(
        instrumentation=instrumentation,
        url=URL(host=opts.host, port=opts.port),
        secure=opts.secure,
        open_timeout=open_timeout,
        read_timeout=read_timeout,
        keep_alive=keep_alive,
        max_retries=max_retries,
    )
    return t
