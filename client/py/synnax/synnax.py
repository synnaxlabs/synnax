#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# On Python 3.15 and later, the submodules below load when a sub-client first uses
# them, not when synnax loads. Older versions ignore this list.
__lazy_modules__ = ["synnax"]

import os
import threading
import weakref
from collections.abc import Callable
from importlib.metadata import version as _version
from typing import Self, overload

from urllib3.exceptions import MaxRetryError

from alamos import NOOP, Instrumentation
from freighter import URL
from freighter.exceptions import Timeout, Unreachable
from synnax import (
    access as access_,
    arc,
    auth,
    certificate,
    channel,
    color,
    connection,
    control as control_,
    device,
    enumeration,
    exceptions,
    framer,
    group,
    icd,
    imex as imex_,
    label,
    license as license_,
    lineplot,
    log,
    panel,
    project,
    rack,
    ranger,
    resource,
    schematic,
    secret,
    signals as signals_,
    status,
    table,
    task,
    user,
    view,
)
from synnax.config import DESKTOP_NOT_RUNNING_MSG, load_options
from synnax.exceptions import ValidationError
from synnax.options import Options
from synnax.telem import CrudeTimeSpan, TimeSpan
from synnax.transport import Transport
from x.exceptions import Registry
from x.params import deprecate_positional


class _Lazy[T]:
    """Builds a sub-client on its first access. The build runs once per client, even
    when threads race for it.
    """

    def __init__(self, build: Callable[["Synnax"], T]) -> None:
        self._build = build
        self._name = build.__name__
        # Keyed by each client's build lock, and weak on the values, so this map never
        # keeps a client or its sub-clients alive.
        self._built: weakref.WeakValueDictionary[object, T] = (
            weakref.WeakValueDictionary()
        )

    @overload
    def __get__(self, obj: None, owner: type) -> Self: ...

    @overload
    def __get__(self, obj: "Synnax", owner: type) -> T: ...

    def __get__(self, obj: "Synnax | None", owner: type) -> "T | Self":
        if obj is None:
            return self
        # After the first build, the value in the instance dict hides this
        # descriptor, so later reads take no lock.
        with obj._build_lock:
            value = self._built.get(obj._build_lock)
            if value is None:
                value = self._build(obj)
                self._built[obj._build_lock] = value
                obj.__dict__[self._name] = value
            return value


def _connect_failed(e: BaseException | None) -> bool:
    """Reports whether a failure to connect caused e. Only that means Synnax Desktop is
    not running: a read timeout or a reset comes from a Core that is running.
    """
    while e is not None:
        if isinstance(e, MaxRetryError):
            return True
        e = e.__cause__
    return False


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

    connectivity: connection.Checker

    _transport: Transport

    @deprecate_positional(
        "host",
        "port",
        "username",
        "password",
        "secure",
        "open_timeout",
        "read_timeout",
        "keep_alive",
        "max_retries",
        "instrumentation",
        "clock_skew_threshold",
        "ca_cert",
    )
    def __init__(
        self,
        *,
        host: str = "",
        port: int = 0,
        username: str = "",
        password: str = "",
        secure: bool | None = None,
        open_timeout: CrudeTimeSpan = TimeSpan.SECOND * 5,
        read_timeout: CrudeTimeSpan = TimeSpan.SECOND * 5,
        keep_alive: CrudeTimeSpan = TimeSpan.SECOND * 30,
        max_retries: int = 3,
        instrumentation: Instrumentation = NOOP,
        clock_skew_threshold: TimeSpan = TimeSpan.SECOND,
        ca_cert: str | None = None,
        api_key: str = "",
    ):
        """Creates a new client. Each option comes from the argument, then from its
        SYNNAX_* environment variable, then from the default. When neither gives a host,
        the Core that `sy login` saved, or else the Core of Synnax Desktop on this
        computer, comes before the default. The credentials and the TLS options (secure,
        ca_cert) each come whole from the first source that sets one of them.

        :param host: Hostname of a Core in the Synnax cluster.
        :param port: Port of the Core.
        :param username: Username to authenticate with.
        :param password: Password to authenticate with.
        :param secure: Whether to use TLS when connecting to the Core. None means True
            with a ca_cert, else the choice of the next source. The client verifies the
            Core's certificate against ca_cert, or the system trust roots when ca_cert
            is None.
        :param open_timeout: The time to wait for a connection. A bare number is
            seconds.
        :param read_timeout: The time to wait for a response. A bare number is seconds.
        :param keep_alive: The time between keep alive pings. A bare number is seconds.
        :param clock_skew_threshold: The maximum allowable clock skew between this
            host and Synnax Core before a warning is issued. Defaults to 1 second.
        :param api_key: API key to authenticate with, in place of a username and
            password.
        :param ca_cert: The path to a PEM file of CA certificates that sign the Core's
            certificate, such as the ca.crt that the Core generates.
        """
        if api_key and (username or password):
            raise ValidationError(
                "give either an API key or a username and password, not both"
            )
        opts, embedded = load_options(
            env=os.environ,
            host=host,
            port=port,
            username=username,
            password=password,
            secure=secure,
            ca_cert=ca_cert,
            api_key=api_key,
        )
        registry = exceptions.create_registry()
        self._registry = registry
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
        try:
            self.auth.authenticate()
        except Unreachable as e:
            if embedded and _connect_failed(e):
                raise Unreachable(
                    target=e.target, message=DESKTOP_NOT_RUNNING_MSG
                ) from e
            raise
        self._transport.use(self.auth.middleware())
        self._transport.use_async(self.auth.async_middleware())
        self.connectivity = connection.Checker(
            client=self._transport.unary,
            client_version=_version("synnax"),
            clock_skew_threshold=clock_skew_threshold,
            embedded=embedded,
        )
        # Not __del__: interpreter finalization would run it after daemon threads
        # are frozen, deadlocking stop()'s join. finalize runs at atexit, before
        # that, and when an unclosed client is garbage collected.
        self._finalizer = weakref.finalize(self, self.connectivity.stop)

        self._build_lock = threading.RLock()
        self._instrumentation = instrumentation
        self._channel_retriever = channel.Retriever(
            client=self._transport.unary, instrumentation=instrumentation
        )
        super().__init__(
            stream_client=self._transport.stream,
            async_client=self._transport.stream_async,
            unary_client=self._transport.unary,
            file_transport=self._transport.file_transport,
            retriever=self._channel_retriever,
            deleter=framer.Deleter(
                client=self._transport.unary, instrumentation=instrumentation
            ),
            registry=registry,
            instrumentation=instrumentation,
        )

    @_Lazy
    def channels(self) -> channel.Client:
        return channel.Client(
            client=self._transport.unary,
            frame_client=self,
            retriever=self._channel_retriever,
            creator=channel.Writer(
                client=self._transport.unary, instrumentation=self._instrumentation
            ),
        )

    @_Lazy
    def access(self) -> access_.Client:
        return access_.Client(self._transport.unary)

    @_Lazy
    def users(self) -> user.Client:
        return user.Client(self._transport.unary)

    @_Lazy
    def ranges(self) -> ranger.Client:
        return ranger.Client(
            unary_client=self._transport.unary,
            frame_client=self,
            channel_retriever=self._channel_retriever,
            writer=ranger.Writer(
                client=self._transport.unary, instrumentation=self._instrumentation
            ),
            retriever=ranger.Retriever(
                client=self._transport.unary, instrumentation=self._instrumentation
            ),
            signals=self.signals,
            labels=self.labels,
        )

    @_Lazy
    def control(self) -> control_.Client:
        return control_.Client(
            framer=self,
            channels=self._channel_retriever,
            client=self._transport.unary,
        )

    @_Lazy
    def signals(self) -> signals_.Registry:
        return signals_.Registry(frame_client=self, channels=self._channel_retriever)

    @_Lazy
    def racks(self) -> rack.Client:
        return rack.Client(client=self._transport.unary)

    @_Lazy
    def devices(self) -> device.Client:
        return device.Client(client=self._transport.unary)

    @_Lazy
    def tasks(self) -> task.Client:
        return task.Client(
            client=self._transport.unary,
            frame_client=self,
            rack_client=self.racks,
            device_client=self.devices,
        )

    @_Lazy
    def resources(self) -> resource.Client:
        return resource.Client(self._transport.unary)

    @_Lazy
    def license(self) -> license_.Client:
        return license_.Client(self._transport.unary)

    @_Lazy
    def statuses(self) -> status.Client:
        return status.Client(self._transport.unary)

    @_Lazy
    def arcs(self) -> arc.Client:
        return arc.Client(self._transport.unary, tasks=self.tasks)

    @_Lazy
    def groups(self) -> group.Client:
        return group.Client(self._transport.unary)

    @_Lazy
    def labels(self) -> label.Client:
        return label.Client(self._transport.unary)

    @_Lazy
    def colors(self) -> color.Client:
        return color.Client(self._transport.unary)

    @_Lazy
    def enumerations(self) -> enumeration.Client:
        return enumeration.Client(self._transport.unary)

    @_Lazy
    def views(self) -> view.Client:
        return view.Client(client=self._transport.unary)

    @_Lazy
    def secrets(self) -> secret.Client:
        return secret.Client(client=self._transport.unary)

    @_Lazy
    def certificates(self) -> certificate.Client:
        return certificate.Client(client=self._transport.unary)

    @_Lazy
    def icds(self) -> icd.Client:
        return icd.Client(client=self._transport.unary, registry=self._registry)

    @_Lazy
    def imex(self) -> imex_.Client:
        return imex_.Client(file_transport=self._transport.file_transport)

    @_Lazy
    def projects(self) -> project.Client:
        return project.Client(
            client=self._transport.unary,
            file_transport=self._transport.file_transport,
        )

    @_Lazy
    def tables(self) -> table.Client:
        return table.Client(client=self._transport.unary)

    @_Lazy
    def line_plots(self) -> lineplot.Client:
        return lineplot.Client(client=self._transport.unary)

    @_Lazy
    def panels(self) -> panel.Client:
        return panel.Client(client=self._transport.unary)

    @_Lazy
    def logs(self) -> log.Client:
        return log.Client(self._transport.unary)

    @_Lazy
    def schematics(self) -> schematic.Client:
        return schematic.Client(client=self._transport.unary)

    def close(self) -> None:
        """Logs out, shuts down the client, and closes all connections. All open
        iterators or writers must be closed before calling this method. When the Core
        cannot be reached or does not respond in time, the session is left to idle
        out.
        """
        # The connectivity thread stops first: a check during the logout would log in
        # again and leave that session open.
        self._finalizer()
        try:
            self.auth.logout()
        except (Unreachable, Timeout):
            pass


def _configure_transport(
    *,
    opts: Options,
    open_timeout: CrudeTimeSpan,
    read_timeout: CrudeTimeSpan,
    keep_alive: CrudeTimeSpan,
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
