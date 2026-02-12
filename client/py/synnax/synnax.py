#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import warnings

from alamos import NOOP, Instrumentation
from freighter import URL

from synnax import (
    access,
    arc,
    auth,
    channel,
    control,
    device,
    framer,
    group,
    ontology,
    rack,
    ranger,
    signals,
    status,
    task,
    user,
)
from synnax.config import try_load_options_if_none_provided
from synnax.options import Options
from synnax.telem import TimeSpan
from synnax.transport import Transport


class Synnax(framer.Client):
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

    channels: channel.Client
    access: access.Client
    users: user.Client
    ranges: ranger.Client
    control: control.Client
    signals: signals.Registry
    racks: rack.Client
    devices: device.Client
    tasks: task.Client
    ontology: ontology.Client
    statuses: status.Client
    arcs: arc.Client
    groups: group.Client

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
        cache_channels: bool = True,
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
        self.auth = auth.Client(
            transport=self._transport.unary,
            username=opts.username,
            password=opts.password,
        )
        self.auth.authenticate()
        self._transport.use(self.auth.middleware())
        self._transport.use_async(self.auth.async_middleware())

        ch_retriever = channel.ClusterRetriever(self._transport.unary, instrumentation)
        if cache_channels:
            ch_retriever = channel.CacheRetriever(ch_retriever, instrumentation)
        deleter = framer.Deleter(self._transport.unary, instrumentation)
        ch_creator = channel.Writer(
            self._transport.unary,
            instrumentation,
            ch_retriever if cache_channels else None,
        )
        super().__init__(
            stream_client=self._transport.stream,
            async_client=self._transport.stream_async,
            unary_client=self._transport.unary,
            retriever=ch_retriever,
            deleter=deleter,
            instrumentation=instrumentation,
        )
        self.groups = group.Client(self._transport.unary)
        self.ontology = ontology.Client(client=self._transport.unary)
        self.channels = channel.Client(self, ch_retriever, ch_creator)
        range_retriever = ranger.Retriever(self._transport.unary, instrumentation)
        range_creator = ranger.Writer(self._transport.unary, instrumentation)
        self.signals = signals.Registry(frame_client=self, channels=ch_retriever)
        self.racks = rack.Client(client=self._transport.unary)
        self.devices = device.Client(client=self._transport.unary)
        self.tasks = task.Client(
            client=self._transport.unary,
            frame_client=self,
            rack_client=self.racks,
            device_client=self.devices,
        )
        self.ranges = ranger.Client(
            unary_client=self._transport.unary,
            frame_client=self,
            channel_retriever=ch_retriever,
            writer=range_creator,
            retriever=range_retriever,
            signals=self.signals,
            ontology=self.ontology,
            tasks=self.tasks,
        )
        self.control = control.Client(self, ch_retriever)
        self.users = user.Client(self._transport.unary)
        self.statuses = status.Client(self._transport.unary)
        self.arcs = arc.Client(self._transport.unary)
        self.access = access.Client(self._transport.unary)

    @property
    def hardware(self) -> "Synnax":
        """Deprecated: Use client.devices, client.tasks, client.racks directly."""
        warnings.warn(
            "client.hardware is deprecated and will be removed in a future version. "
            "Use client.devices, client.tasks, client.racks directly instead.",
            FutureWarning,
            stacklevel=2,
        )
        return self

    def close(self):
        """Shuts down the client and closes all connections. All open iterators or
        writers must be closed before calling this method.
        """
        # No-op for now, we'll definitely add cleanup logic in the future, so it's
        # good to have this API defined.
        ...


def _configure_transport(
    opts: Options,
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
