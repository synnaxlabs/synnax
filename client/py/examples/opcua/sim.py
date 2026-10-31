#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import datetime
import math
import random
import socket
import ssl
from pathlib import Path

from asyncua import ua
from asyncua.client.client import Client
from asyncua.common.node import Node
from asyncua.crypto.cert_gen import setup_self_signed_certificate
from asyncua.crypto.permission_rules import User, UserRole
from asyncua.crypto.validator import CertificateValidator, CertificateValidatorOptions
from asyncua.server.internal_server import InternalServer
from asyncua.server.server import Server
from asyncua.server.user_managers import UserManager
from asyncua.ua.attribute_ids import AttributeIds
from cryptography.x509.oid import ExtendedKeyUsageOID

import synnax as sy
from examples.simulators.device_sim import DeviceSim
from synnax import opcua

ARRAY_COUNT = 5
DEFAULT_ARRAY_SIZE = 5
FLOAT_COUNT = 5
BOOL_COUNT = 5
COMMAND_COUNT = 3
DEFAULT_RATE = 50 * sy.Rate.HZ
BOOL_TRANSITION_OFFSET = 200 * sy.TimeSpan.MILLISECOND

ERROR_INJECTION_RATE = 0.1
ERROR_ARRAY_INDEX = 2
ERROR_FLOAT_INDEX = 2

CERT_DIR = Path(__file__).parent / "certificates"


def read_server_certificate() -> str:
    """Return the simulator's server certificate as PEM. The server writes it as DER
    when it starts with encryption."""
    der = (CERT_DIR / "server-certificate.der").read_bytes()
    return ssl.DER_cert_to_PEM_cert(der)


async def write_array_dimensions(node: Node, size: int) -> None:
    """Write a one-dimensional ArrayDimensions attribute. asyncua annotates
    Node.write_array_dimensions as int, but the attribute is an array."""
    dims = ua.Variant([size], ua.VariantType.UInt32)
    await node.write_attribute(AttributeIds.ArrayDimensions, ua.DataValue(dims))


async def create_array_variables(
    myobj: Node, idx: int, array_size: int = DEFAULT_ARRAY_SIZE
) -> list[Node]:
    """Create array variables with initial values."""
    arrays = []
    for i in range(ARRAY_COUNT):
        initial_values = [float(j + i) for j in range(array_size)]
        arr = await myobj.add_variable(
            idx, f"my_array_{i}", initial_values, ua.VariantType.Float
        )
        await write_array_dimensions(arr, array_size)
        arrays.append(arr)
    return arrays


async def create_time_array(
    myobj: Node, idx: int, array_size: int = DEFAULT_ARRAY_SIZE
) -> Node:
    """Create timestamp array variable."""
    now = datetime.datetime.now(datetime.UTC)
    initial_times = [
        now + datetime.timedelta(milliseconds=j) for j in range(array_size)
    ]
    mytimearray = await myobj.add_variable(
        idx, "my_time_array", initial_times, ua.VariantType.DateTime
    )
    await mytimearray.set_writable()
    await write_array_dimensions(mytimearray, array_size)
    return mytimearray


async def create_float_variables(myobj: Node, idx: int) -> list[Node]:
    """Create scalar float variables."""
    floats = []
    for i in range(FLOAT_COUNT):
        my_float = await myobj.add_variable(
            idx, f"my_float_{i}", 0.0, ua.VariantType.Float
        )
        await my_float.set_writable()
        floats.append(my_float)
    return floats


async def create_bool_variables(myobj: Node, idx: int) -> list[Node]:
    """Create scalar boolean variables."""
    bools = []
    for i in range(BOOL_COUNT):
        my_bool = await myobj.add_variable(
            idx, f"my_bool_{i}", False, ua.VariantType.Boolean
        )
        await my_bool.set_writable()
        bools.append(my_bool)
    return bools


async def create_command_variables(myobj: Node, idx: int) -> list[Node]:
    """Create writable command variables for testing write operations."""
    commands = []
    for i in range(COMMAND_COUNT):
        cmd = await myobj.add_variable(idx, f"command_{i}", 0.0, ua.VariantType.Float)
        await cmd.set_writable()
        commands.append(cmd)
    return commands


async def monitor_command_changes(
    commands: list[Node], command_values: list[object]
) -> None:
    """Periodically check command variables for changes and print them."""
    while True:
        for i, cmd in enumerate(commands):
            current_value = await cmd.read_value()
            if command_values[i] != current_value:
                timestamp = (
                    datetime.datetime.now().astimezone().strftime("%H:%M:%S.%f")[:-3]
                )
                print(f"[{timestamp}] COMMAND RECEIVED -> command_{i}: {current_value}")
                command_values[i] = current_value
        await asyncio.sleep(0.1)  # Check every 100ms


def generate_timestamps(
    start: datetime.datetime, rate: float, size: int
) -> list[datetime.datetime]:
    """Generate array of timestamps."""
    return [start + datetime.timedelta(seconds=j * (1 / rate)) for j in range(size)]


def generate_sinewave_values(
    timestamps: list[datetime.datetime], start_ref: datetime.datetime
) -> list[float]:
    """Generate sinewave values based on timestamps."""
    return [
        math.sin((timestamp - start_ref).total_seconds()) for timestamp in timestamps
    ]


def inject_error[T](values: list[T], array_size: int = DEFAULT_ARRAY_SIZE) -> list[T]:
    """
    Generate corrupted array data for error injection.
    """

    error_chance = random.random()

    if error_chance < 0.333:
        return []

    if error_chance < 0.667:
        size = random.randint(1, max(2, array_size - 2))
        return values[:size]

    extra = random.randint(1, 3)
    return values + [values[-1] for _ in range(extra)]


async def update_arrays(arrays: list[Node], values: list[float]) -> None:
    """Update array variables with generated values.

    Injects random errors into ERROR_ARRAY_INDEX at ERROR_INJECTION_RATE.
    """
    for i, arr in enumerate(arrays):
        offset_values = [v + i for v in values]

        if i == ERROR_ARRAY_INDEX and random.random() < ERROR_INJECTION_RATE:
            offset_values = inject_error(offset_values)

        await arr.set_value(offset_values, varianttype=ua.VariantType.Float)


async def update_floats(floats: list[Node], elapsed: float) -> None:
    """Update scalar float variables with sinewave values.

    Injects random errors into ERROR_FLOAT_INDEX at ERROR_INJECTION_RATE
    by skipping the write (using same pattern as arrays).
    """
    for idx, float_var in enumerate(floats):
        value = math.sin(elapsed) + idx
        written: float | list[float] = value
        if idx == ERROR_FLOAT_INDEX and random.random() < ERROR_INJECTION_RATE:
            written = inject_error([value])

        await float_var.set_value(written, varianttype=ua.VariantType.Float)


async def update_bools(bools: list[Node], elapsed: float) -> None:
    """Update scalar boolean variables with sequential square waves."""
    for idx, bool_var in enumerate(bools):
        offset_elapsed = elapsed + (idx * BOOL_TRANSITION_OFFSET.seconds)
        square_wave = int(offset_elapsed) % 2 == 0
        written: bool | list[bool] = square_wave
        if idx == ERROR_FLOAT_INDEX and random.random() < ERROR_INJECTION_RATE:
            written = inject_error([square_wave])

        await bool_var.set_value(written, varianttype=ua.VariantType.Boolean)


class SimpleUserManager(UserManager):
    """A user manager that validates credentials against a static user dict."""

    def __init__(self, users: dict[str, str]):
        self.users = users

    def get_user(
        self,
        iserver: InternalServer,
        username: str | None = None,
        password: str | None = None,
        certificate: object = None,
    ) -> User | None:
        if username in self.users and self.users[username] == password:
            return User(role=UserRole.User)
        return None


async def configure_encryption(server: Server) -> None:
    """Generate self-signed certificates and configure server encryption."""
    CERT_DIR.mkdir(exist_ok=True)
    server_cert = CERT_DIR / "server-certificate.der"
    server_key = CERT_DIR / "server-private-key.pem"
    client_cert = CERT_DIR / "client-certificate.der"
    client_key = CERT_DIR / "client-private-key.pem"

    host_name = socket.gethostname()
    server_uri = f"urn:synnax:opcua:server:{host_name}"
    client_uri = f"urn:synnax:opcua:client:{host_name}"
    subject = {
        "countryName": "US",
        "stateOrProvinceName": "Colorado",
        "localityName": "Evergreen",
        "organizationName": "Synnax Labs",
    }

    await setup_self_signed_certificate(
        server_key,
        server_cert,
        server_uri,
        host_name,
        [ExtendedKeyUsageOID.CLIENT_AUTH, ExtendedKeyUsageOID.SERVER_AUTH],
        subject,
    )
    await setup_self_signed_certificate(
        client_key,
        client_cert,
        client_uri,
        host_name,
        [ExtendedKeyUsageOID.CLIENT_AUTH],
        subject,
    )

    await server.set_application_uri(server_uri)
    server.set_security_policy(
        [
            ua.SecurityPolicyType.Basic256Sha256_SignAndEncrypt,
            ua.SecurityPolicyType.Basic256Sha256_Sign,
        ]
    )
    await server.load_certificate(str(server_cert))
    await server.load_private_key(str(server_key))

    validator = CertificateValidator(options=CertificateValidatorOptions.EXT_VALIDATION)
    server.set_certificate_validator(validator)


async def run_server(
    *,
    endpoint: str = "",
    rate: sy.Rate = DEFAULT_RATE,
    array_size: int = DEFAULT_ARRAY_SIZE,
    encrypted: bool = False,
    user_manager: UserManager | None = None,
    max_sessions: int | None = None,
) -> None:
    server = Server(user_manager=user_manager)
    await server.init()
    if not endpoint:
        endpoint = OPCUASim.endpoint
    server.set_endpoint(endpoint)

    if max_sessions is not None and max_sessions > 0:
        from asyncua.server.internal_session import InternalSession

        InternalSession.max_connections = max_sessions

    if encrypted:
        await configure_encryption(server)

    uri = "http://examples.freeopcua.github.io"
    idx = await server.register_namespace(uri)

    myobj = await server.nodes.objects.add_object(idx, "MyObject")
    arrays = await create_array_variables(myobj, idx, array_size)
    mytimearray = await create_time_array(myobj, idx, array_size)
    floats = await create_float_variables(myobj, idx)
    bools = await create_bool_variables(myobj, idx)
    commands = await create_command_variables(myobj, idx)

    print("\n" + "=" * 60)
    print("OPC UA Server Started - Monitoring Command Variables")
    print("=" * 60)
    print("\nCommand variables (writable, monitored for external writes):")
    command_values: list[object] = []
    for i, cmd in enumerate(commands):
        node_id = cmd.nodeid.to_string()
        print(f"  command_{i}: {node_id}")
        initial_value = await cmd.read_value()
        command_values.append(initial_value)

    print("\nRead-only variables (auto-updated by server):")
    print(f"  Arrays: my_array_0 to my_array_{ARRAY_COUNT - 1}")
    print(f"  Floats: my_float_0 to my_float_{FLOAT_COUNT - 1}")
    print(f"  Bools:  my_bool_0 to my_bool_{BOOL_COUNT - 1}")
    print("\nWaiting for commands...\n")

    start_ref = datetime.datetime.now(datetime.UTC)

    async with server, asyncio.TaskGroup() as tg:
        tg.create_task(monitor_command_changes(commands, command_values))
        while True:
            start = datetime.datetime.now(datetime.UTC)
            elapsed = (start - start_ref).total_seconds()

            timestamps = generate_timestamps(start, rate, array_size)
            sinewave_values = generate_sinewave_values(timestamps, start_ref)

            await update_arrays(arrays, sinewave_values)
            await mytimearray.set_value(timestamps, varianttype=ua.VariantType.DateTime)
            await update_floats(floats, elapsed)
            await update_bools(bools, elapsed)

            duration = (datetime.datetime.now(datetime.UTC) - start).total_seconds()
            await asyncio.sleep(max(0, (1 / rate) - duration))


class OPCUASim(DeviceSim):
    """OPC UA device simulator on port 4841."""

    description = "OPC UA simulator on port 4841"
    host = "127.0.0.1"
    port = 4841
    device_name = "OPC UA Server"
    endpoint = f"opc.tcp://{host}:{port}/freeopcua/server/"
    channel_names = (
        *(f"my_array_{i}" for i in range(ARRAY_COUNT)),
        "my_time_array",
        *(f"my_float_{i}" for i in range(FLOAT_COUNT)),
        *(f"my_bool_{i}" for i in range(BOOL_COUNT)),
        *(f"command_{i}" for i in range(COMMAND_COUNT)),
    )

    def __init__(
        self,
        *,
        array_size: int = DEFAULT_ARRAY_SIZE,
        rate: sy.Rate = 50 * sy.Rate.HZ,
        verbose: bool = False,
        encrypted: bool = False,
        user_manager: UserManager | None = None,
        max_sessions: int | None = None,
    ):
        super().__init__(rate=rate, verbose=verbose)
        self.array_size = array_size
        self.encrypted = encrypted
        self.user_manager = user_manager
        self.max_sessions = max_sessions

    def _wait_for_ready(self) -> None:
        """Poll until the server answers a GetEndpoints request. The socket accepts
        connections before the server is initialized, so the TCP check is too early."""
        super()._wait_for_ready()
        timer = sy.Timer()
        while True:
            try:
                client = Client(self.endpoint, timeout=1)
                asyncio.run(client.connect_and_get_server_endpoints())
                return
            except Exception as e:
                if timer.elapsed() > self.startup_timeout:
                    raise RuntimeError(
                        f"Server on {self.endpoint} not answering discovery after "
                        f"{self.startup_timeout}"
                    ) from e
                sy.sleep(0.1)

    async def _run_server(self) -> None:
        await run_server(
            endpoint=self.endpoint,
            rate=self.rate,
            array_size=self.array_size,
            encrypted=self.encrypted,
            user_manager=self.user_manager,
            max_sessions=self.max_sessions,
        )

    @staticmethod
    def create_device(rack_key: int) -> opcua.Device:
        return opcua.Device(
            endpoint=OPCUASim.endpoint,
            name=OPCUASim.device_name,
            location=OPCUASim.endpoint,
            rack=rack_key,
        )


class OPCUATLSSim(OPCUASim):
    """TLS-encrypted OPC UA device simulator on port 4842."""

    description = "TLS OPC UA simulator on port 4842"
    host = "127.0.0.1"
    port = 4842
    device_name = "OPC UA TLS Server"
    endpoint = f"opc.tcp://{host}:{port}/freeopcua/server/"

    def __init__(
        self,
        *,
        array_size: int = DEFAULT_ARRAY_SIZE,
        rate: sy.Rate = 50 * sy.Rate.HZ,
        verbose: bool = False,
        max_sessions: int | None = None,
    ):
        super().__init__(
            array_size=array_size,
            rate=rate,
            verbose=verbose,
            encrypted=True,
            max_sessions=max_sessions,
        )

    @staticmethod
    def create_device(rack_key: int) -> opcua.Device:
        return opcua.Device(
            endpoint=OPCUATLSSim.endpoint,
            name=OPCUATLSSim.device_name,
            location=OPCUATLSSim.endpoint,
            rack=rack_key,
            security_mode="SignAndEncrypt",
            security_policy="Basic256Sha256",
            server_cert=read_server_certificate(),
            client_cert=str(CERT_DIR / "client-certificate.der"),
            client_private_key=str(CERT_DIR / "client-private-key.pem"),
        )


class OPCUATLSDriverCertificateSim(OPCUATLSSim):
    """TLS-encrypted OPC UA simulator on port 4844 that the Driver reaches with its own
    certificate. The server checks the certificate's extensions and URI, but trusts any
    issuer. Its device trusts no server certificate, so the Driver pins the first one it
    sees."""

    description = "TLS OPC UA simulator for the Driver certificate on port 4844"
    host = "127.0.0.1"
    port = 4844
    device_name = "OPC UA TLS Driver Certificate Server"
    endpoint = f"opc.tcp://{host}:{port}/freeopcua/server/"

    @staticmethod
    def create_device(rack_key: int) -> opcua.Device:
        return opcua.Device(
            endpoint=OPCUATLSDriverCertificateSim.endpoint,
            name=OPCUATLSDriverCertificateSim.device_name,
            location=OPCUATLSDriverCertificateSim.endpoint,
            rack=rack_key,
            security_mode="SignAndEncrypt",
            security_policy="Basic256Sha256",
        )


TLS_AUTH_USERNAME = "testuser"
TLS_AUTH_PASSWORD = "testpass"


class OPCUATLSAuthSim(OPCUASim):
    """TLS-encrypted OPC UA simulator with username/password auth on port 4843."""

    description = "TLS OPC UA simulator with user/pass on port 4843"
    host = "127.0.0.1"
    port = 4843
    device_name = "OPC UA TLS Auth Server"
    endpoint = f"opc.tcp://{host}:{port}/freeopcua/server/"

    def __init__(
        self,
        *,
        array_size: int = DEFAULT_ARRAY_SIZE,
        rate: sy.Rate = 50 * sy.Rate.HZ,
        verbose: bool = False,
        max_sessions: int | None = None,
    ):
        super().__init__(
            array_size=array_size,
            rate=rate,
            verbose=verbose,
            encrypted=True,
            user_manager=SimpleUserManager({TLS_AUTH_USERNAME: TLS_AUTH_PASSWORD}),
            max_sessions=max_sessions,
        )

    @staticmethod
    def create_device(rack_key: int) -> opcua.Device:
        return opcua.Device(
            endpoint=OPCUATLSAuthSim.endpoint,
            name=OPCUATLSAuthSim.device_name,
            location=OPCUATLSAuthSim.endpoint,
            rack=rack_key,
            security_mode="SignAndEncrypt",
            security_policy="Basic256Sha256",
            server_cert=read_server_certificate(),
            client_cert=str(CERT_DIR / "client-certificate.der"),
            client_private_key=str(CERT_DIR / "client-private-key.pem"),
            username=TLS_AUTH_USERNAME,
            password=TLS_AUTH_PASSWORD,
        )
