#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import weakref
from importlib.metadata import version as _version

from alamos import NOOP, Instrumentation
from freighter import URL
from freighter.exceptions import Unreachable
from synnax import (
    arc,
    auth,
    channel,
    connection,
    device,
    exceptions,
    framer,
    group,
    project,
    rack,
    ranger,
    status,
    task,
    user,
    view,
)
from synnax.access import Client as AccessClient
from synnax.config import try_load_options_if_none_provided
from synnax.control import Client as ControlClient
from synnax.exceptions import ValidationError
from synnax.imex import Client as ImexClient
from synnax.license import Client as LicenseClient
from synnax.ontology import Client as OntologyClient
from synnax.options import Options
from synnax.ranger.retrieve import Retriever as RangeRetriever
from synnax.signals import Registry as SignalsRegistry
from synnax.telem import TimeSpan
from synnax.transport import Transport
from x.exceptions import Registry


class Synnax(framer.Client):
    """Client to perform operations against a Synnax Core.

    If using the Python client for data analysis/personal use, the easiest way to
    connect is to use the `sy login` command, which will prompt and securely store your
    credentials. The client can then be initialized without parameters. When using the
    client in a production environment, it's best to provide the connection parameter as
    arguments loaded from a configuration or environment variable.

    After running the sy login command::
        client = sy.Synnax()

    Without running the sy login command::
        client = sy.Synnax(
            host="synnax.example.com",
            port=9090,
            username="synnax",
            password="seldon",
            secure=True,
            ca_cert="/path/to/ca.crt",
        )

    With an API key::
        client = sy.Synnax(host="synnax.example.com", port=9090, api_key="syk_...")
    """

    channels: channel.Client
    connectivity: connection.Checker
    access: AccessClient
    users: user.Client
    ranges: ranger.Client
    control: ControlClient
    signals: SignalsRegistry
    racks: rack.Client
    devices: device.Client
    tasks: task.Client
    ontology: OntologyClient
    license: LicenseClient
    statuses: status.Client
    arcs: arc.Client
    groups: group.Client
    views: view.Client
    imex: ImexClient
    projects: project.Client

    _transport: Transport

    def __init__(
        self,
        *,
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
        clock_skew_threshold: TimeSpan = TimeSpan.SECOND,
        ca_cert: str | None = None,
        api_key: str = "",
    ):
        """Creates a new client. Connection parameters can be provided as arguments, or,
        if none are provided, the client will attempt to load them from the Synnax
        configuration file (~/.synnax/config.json) as well as credentials stored in the
        operating system's keychain.

        If using the client for data analysis/personal use, the easiest way to connect
        is to use the `sy login` command, which will prompt and securely store your
        credentials. The client can then be initialized without parameters.

        :param host: Hostname of a Core in the Synnax cluster.
        :param port: Port of the Core.
        :param username: Username to authenticate with.
        :param password: Password to authenticate with.
        :param secure: Whether to use TLS when connecting to the Core. The client
            verifies the Core's certificate against ca_cert, or the system trust roots
            when ca_cert is None.
        :param clock_skew_threshold: The maximum allowable clock skew between this
            host and Synnax Core before a warning is issued. Defaults to 1 second.
        :param api_key: API key to authenticate with, in place of a username and
            password. Requires a host.
        :param ca_cert: The path to a PEM file of CA certificates that sign the Core's
            certificate, such as the ca.crt that the Core generates.
        """
        if api_key and (username or password):
            raise ValidationError(
                "give either an API key or a username and password, not both"
            )
        opts = try_load_options_if_none_provided(
            host=host,
            port=port,
            username=username,
            password=password,
            secure=secure,
            ca_cert=ca_cert,
            api_key=api_key,
        )
        registry = exceptions.create_registry()
        self._transport = _configure_transport(
            opts=opts,
            open_timeout=open_timeout,
            read_timeout=read_timeout,
            keep_alive=keep_alive,
            max_retries=max_retries,
            instrumentation=instrumentation,
            registry=registry,
        )
        self.auth = auth.Client(
            transport=self._transport.unary,
            proof=(
                auth.APIKeyProof(key=opts.api_key)
                if opts.api_key
                else auth.PasswordProof(username=opts.username, password=opts.password)
            ),
        )
        self.auth.authenticate()
        self._transport.use(self.auth.middleware())
        self._transport.use_async(self.auth.async_middleware())
        self.connectivity = connection.Checker(
            client=self._transport.unary,
            client_version=_version("synnax"),
            clock_skew_threshold=clock_skew_threshold,
        )
        # Not __del__: interpreter finalization would run it after daemon threads
        # are frozen, deadlocking stop()'s join. finalize runs at atexit, before
        # that, and when an unclosed client is garbage collected.
        self._finalizer = weakref.finalize(self, self.connectivity.stop)

        ch_retriever = channel.Retriever(
            client=self._transport.unary, instrumentation=instrumentation
        )
        deleter = framer.Deleter(
            client=self._transport.unary, instrumentation=instrumentation
        )
        ch_creator = channel.Writer(
            client=self._transport.unary, instrumentation=instrumentation
        )
        super().__init__(
            stream_client=self._transport.stream,
            async_client=self._transport.stream_async,
            unary_client=self._transport.unary,
            file_transport=self._transport.file_transport,
            retriever=ch_retriever,
            deleter=deleter,
            registry=registry,
            instrumentation=instrumentation,
        )
        self.groups = group.Client(self._transport.unary)
        self.ontology = OntologyClient(client=self._transport.unary)
        self.channels = channel.Client(
            frame_client=self, retriever=ch_retriever, creator=ch_creator
        )
        range_retriever = RangeRetriever(
            client=self._transport.unary, instrumentation=instrumentation
        )
        range_creator = ranger.Writer(
            client=self._transport.unary, instrumentation=instrumentation
        )
        self.signals = SignalsRegistry(frame_client=self, channels=ch_retriever)
        self.racks = rack.Client(client=self._transport.unary)
        self.devices = device.Client(client=self._transport.unary)
        self.views = view.Client(client=self._transport.unary)
        self.imex = ImexClient(file_transport=self._transport.file_transport)
        self.projects = project.Client(
            client=self._transport.unary,
            file_transport=self._transport.file_transport,
        )
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
        self.control = ControlClient(
            framer=self, channels=ch_retriever, client=self._transport.unary
        )
        self.users = user.Client(self._transport.unary)
        self.statuses = status.Client(self._transport.unary)
        self.license = LicenseClient(self._transport.unary)
        self.arcs = arc.Client(self._transport.unary)
        self.access = AccessClient(self._transport.unary)

    def close(self) -> None:
        """Logs out, shuts down the client, and closes all connections. All open
        iterators or writers must be closed before calling this method. When the Core
        cannot be reached the session is left to idle out.
        """
        # The connectivity thread stops first: a check during the logout would log in
        # again and leave that session open.
        self._finalizer()
        try:
            self.auth.logout()
        except Unreachable:
            pass


def _configure_transport(
    *,
    opts: Options,
    open_timeout: TimeSpan,
    read_timeout: TimeSpan,
    keep_alive: TimeSpan,
    max_retries: int,
    registry: Registry,
    instrumentation: Instrumentation = NOOP,
) -> Transport:
    return Transport(
        instrumentation=instrumentation,
        registry=registry,
        url=URL(host=opts.host, port=opts.port),
        secure=opts.secure,
        ca_cert=opts.ca_cert,
        open_timeout=open_timeout,
        read_timeout=read_timeout,
        keep_alive=keep_alive,
        max_retries=max_retries,
    )
