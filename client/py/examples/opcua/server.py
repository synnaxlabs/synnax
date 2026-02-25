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
from pathlib import Path

from asyncua import Server, ua
from asyncua.crypto.cert_gen import setup_self_signed_certificate
from asyncua.crypto.validator import CertificateValidator, CertificateValidatorOptions
from cryptography.x509.oid import ExtendedKeyUsageOID

import synnax as sy
from examples.simulators.device_sim import DeviceSim
from synnax import opcua

# Configuration constants
ARRAY_COUNT = 5
DEFAULT_ARRAY_SIZE = 5
FLOAT_COUNT = 5
BOOL_COUNT = 5
DEFAULT_RATE = 50  # Hz
BOOL_OFFSET = 0.2  # seconds between each boolean transition

# Error injection configuration
ERROR_INJECTION_RATE = 0.1  # 10% error rate
ERROR_ARRAY_INDEX = 2  # Which array to inject errors into
ERROR_FLOAT_INDEX = 2  # Which float to inject errors into

# Encryption certificate directory
CERT_DIR = Path(__file__).parent / "certificates"


# Initialization Functions
async def create_array_variables(myobj, idx, array_size: int = DEFAULT_ARRAY_SIZE):
    """Create array variables with initial values."""
    arrays = []
    for i in range(ARRAY_COUNT):
        initial_values = [float(j + i) for j in range(array_size)]
        arr = await myobj.add_variable(
            idx, f"my_array_{i}", initial_values, ua.VariantType.Float
        )
        await arr.write_array_dimensions([array_size])
        arrays.append(arr)
    return arrays


async def create_time_array(myobj, idx, array_size: int = DEFAULT_ARRAY_SIZE):
    """Create timestamp array variable."""
    now = datetime.datetime.now(datetime.timezone.utc)
    initial_times = [
        now + datetime.timedelta(milliseconds=j) for j in range(array_size)
    ]
    mytimearray = await myobj.add_variable(
        idx, "my_time_array", initial_times, ua.VariantType.DateTime
    )
    await mytimearray.set_writable()
    await mytimearray.write_array_dimensions([array_size])
    return mytimearray


async def create_float_variables(myobj, idx):
    """Create scalar float variables."""
    floats = []
    for i in range(FLOAT_COUNT):
        my_float = await myobj.add_variable(
            idx, f"my_float_{i}", 0.0, ua.VariantType.Float
        )
        await my_float.set_writable()
        floats.append(my_float)
    return floats


async def create_bool_variables(myobj, idx):
    """Create scalar boolean variables."""
    bools = []
    for i in range(BOOL_COUNT):
        my_bool = await myobj.add_variable(
            idx, f"my_bool_{i}", False, ua.VariantType.Boolean
        )
        await my_bool.set_writable()
        bools.append(my_bool)
    return bools


async def create_command_variables(myobj, idx):
    """Create writable command variables for testing write operations."""
    commands = []
    for i in range(3):  # Create 3 command channels
        cmd = await myobj.add_variable(idx, f"command_{i}", 0.0, ua.VariantType.Float)
        await cmd.set_writable()
        commands.append(cmd)
    return commands


async def monitor_command_changes(commands, command_values):
    """Periodically check command variables for changes and print them."""
    while True:
        for i, cmd in enumerate(commands):
            current_value = await cmd.read_value()
            if command_values[i] != current_value:
                timestamp = datetime.datetime.now().strftime("%H:%M:%S.%f")[:-3]
                print(f"[{timestamp}] COMMAND RECEIVED -> command_{i}: {current_value}")
                command_values[i] = current_value
        await asyncio.sleep(0.1)  # Check every 100ms


# Data Generation Functions
def generate_timestamps(start, rate, size):
    """Generate array of timestamps."""
    return [start + datetime.timedelta(seconds=j * (1 / rate)) for j in range(size)]


def generate_sinewave_values(timestamps, start_ref):
    """Generate sinewave values based on timestamps."""
    return [
        math.sin((timestamp - start_ref).total_seconds()) for timestamp in timestamps
    ]


# Update Functions
def inject_error(values, array_size: int = DEFAULT_ARRAY_SIZE):
    """
    Generate corrupted array data for error injection.
    """

    error_chance = random.random()

    # Empty array
    if error_chance < 0.333:
        return []

    # Array smaller than expected
    elif error_chance < 0.667:
        size = random.randint(1, max(2, array_size - 2))
        return values[:size]

    # Array larger than expected
    else:
        extra = random.randint(1, 3)
        return values + [values[-1] for _ in range(extra)]


async def update_arrays(arrays, values):
    """Update array variables with generated values.

    Injects random errors into ERROR_ARRAY_INDEX at ERROR_INJECTION_RATE.
    """
    for i, arr in enumerate(arrays):
        offset_values = [v + i for v in values]

        if i == ERROR_ARRAY_INDEX and random.random() < ERROR_INJECTION_RATE:
            offset_values = inject_error(offset_values)

        await arr.set_value(offset_values, varianttype=ua.VariantType.Float)


async def update_floats(floats, elapsed):
    """Update scalar float variables with sinewave values.

    Injects random errors into ERROR_FLOAT_INDEX at ERROR_INJECTION_RATE
    by skipping the write (using same pattern as arrays).
    """
    for idx, float_var in enumerate(floats):

        value = math.sin(elapsed) + idx
        if idx == ERROR_FLOAT_INDEX and random.random() < ERROR_INJECTION_RATE:
            value = inject_error([value])

        await float_var.set_value(value, varianttype=ua.VariantType.Float)


async def update_bools(bools, elapsed):
    """Update scalar boolean variables with sequential square waves."""
    for idx, bool_var in enumerate(bools):
        offset_elapsed = elapsed + (idx * BOOL_OFFSET)
        square_wave = int(offset_elapsed) % 2 == 0
        if idx == ERROR_FLOAT_INDEX and random.random() < ERROR_INJECTION_RATE:
            square_wave = inject_error([square_wave])

        await bool_var.set_value(square_wave, varianttype=ua.VariantType.Boolean)


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

    validator = CertificateValidator(
        options=CertificateValidatorOptions.EXT_VALIDATION
    )
    server.set_certificate_validator(validator)


async def run_server(
    endpoint: str = "",
    rate: sy.Rate = DEFAULT_RATE * sy.Rate.HZ,
    array_size: int = DEFAULT_ARRAY_SIZE,
    encrypted: bool = False,
) -> None:
    # Initialize server
    server = Server()
    await server.init()
    if not endpoint:
        endpoint = OPCUASim.endpoint
    server.set_endpoint(endpoint)

    if encrypted:
        await configure_encryption(server)

    uri = "http://examples.freeopcua.github.io"
    idx = await server.register_namespace(uri)

    # Create OPC UA object and variables
    myobj = await server.nodes.objects.add_object(idx, "MyObject")
    arrays = await create_array_variables(myobj, idx, array_size)
    mytimearray = await create_time_array(myobj, idx, array_size)
    floats = await create_float_variables(myobj, idx)
    bools = await create_bool_variables(myobj, idx)
    commands = await create_command_variables(myobj, idx)

    # Set up monitoring for command variable writes
    print("\n" + "=" * 60)
    print("OPC UA Server Started - Monitoring Command Variables")
    print("=" * 60)
    print("\nCommand variables (writable, monitored for external writes):")
    command_values = []
    for i, cmd in enumerate(commands):
        node_id = cmd.nodeid.to_string()
        print(f"  command_{i}: {node_id}")
        initial_value = await cmd.read_value()
        command_values.append(initial_value)

    print("\nRead-only variables (auto-updated by server):")
    print(f"  Arrays: my_array_0 to my_array_{ARRAY_COUNT-1}")
    print(f"  Floats: my_float_0 to my_float_{FLOAT_COUNT-1}")
    print(f"  Bools:  my_bool_0 to my_bool_{BOOL_COUNT-1}")
    print("\nWaiting for commands...\n")

    # Start monitoring task
    asyncio.create_task(monitor_command_changes(commands, command_values))

    # Start server loop
    start_ref = datetime.datetime.now(datetime.timezone.utc)

    async with server:
        while True:
            start = datetime.datetime.now(datetime.timezone.utc)
            elapsed = (start - start_ref).total_seconds()

            # Generate data
            timestamps = generate_timestamps(start, rate, array_size)
            sinewave_values = generate_sinewave_values(timestamps, start_ref)

            # Update all variables
            await update_arrays(arrays, sinewave_values)
            await mytimearray.set_value(timestamps, varianttype=ua.VariantType.DateTime)
            await update_floats(floats, elapsed)
            await update_bools(bools, elapsed)

            # Sleep to maintain rate
            duration = (
                datetime.datetime.now(datetime.timezone.utc) - start
            ).total_seconds()
            await asyncio.sleep(max(0, (1 / rate) - duration))


class OPCUASim(DeviceSim):
    """OPC UA device simulator on port 4841."""

    description = "OPC UA simulator on port 4841"
    host = "127.0.0.1"
    port = 4841
    device_name = "OPC UA Server"
    endpoint = f"opc.tcp://{host}:{port}/freeopcua/server/"

    def __init__(
        self,
        array_size: int = DEFAULT_ARRAY_SIZE,
        rate: sy.Rate = 50 * sy.Rate.HZ,
        verbose: bool = False,
        encrypted: bool = False,
    ):
        super().__init__(rate=rate, verbose=verbose)
        self.array_size = array_size
        self.encrypted = encrypted

    async def _run_server(self) -> None:
        await run_server(self.endpoint, self.rate, self.array_size, self.encrypted)

    @staticmethod
    def create_device(rack_key: int) -> opcua.Device:
        return opcua.Device(
            endpoint=OPCUASim.endpoint,
            name=OPCUASim.device_name,
            location=OPCUASim.endpoint,
            rack=rack_key,
        )


class OPCUAEncryptedSim(OPCUASim):
    """Encrypted OPC UA device simulator on port 4842."""

    description = "Encrypted OPC UA simulator on port 4842"
    host = "127.0.0.1"
    port = 4842
    device_name = "OPC UA Encrypted Server"
    endpoint = f"opc.tcp://{host}:{port}/freeopcua/server/"

    def __init__(
        self,
        array_size: int = DEFAULT_ARRAY_SIZE,
        rate: sy.Rate = 50 * sy.Rate.HZ,
        verbose: bool = False,
    ):
        super().__init__(
            array_size=array_size,
            rate=rate,
            verbose=verbose,
            encrypted=True,
        )

    @staticmethod
    def create_device(rack_key: int) -> opcua.Device:
        return opcua.Device(
            endpoint=OPCUAEncryptedSim.endpoint,
            name=OPCUAEncryptedSim.device_name,
            location=OPCUAEncryptedSim.endpoint,
            rack=rack_key,
            security_mode="SignAndEncrypt",
            security_policy="Basic256Sha256",
            server_cert=str(CERT_DIR / "server-certificate.der"),
            client_cert=str(CERT_DIR / "client-certificate.der"),
            client_private_key=str(CERT_DIR / "client-private-key.pem"),
        )


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="OPC UA test server")
    parser.add_argument(
        "--encrypted", action="store_true", help="Enable encryption (port 4842)"
    )
    args = parser.parse_args()

    if args.encrypted:
        asyncio.run(run_server(endpoint=OPCUAEncryptedSim.endpoint, encrypted=True))
    else:
        asyncio.run(run_server())
