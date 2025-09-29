#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import logging
import socket
import sys
from pathlib import Path

sys.path.insert(0, "..")
from asyncua import Server, ua
from asyncua.crypto.truststore import TrustStore
from asyncua.crypto.validator import CertificateValidator, CertificateValidatorOptions

logging.basicConfig(level=logging.INFO)

USE_TRUST_STORE = False


async def main():
    server_cert = Path("server.der")
    server_private_key = Path("server.key.der")

    host_name = socket.gethostname()
    server_app_uri = f"myselfsignedserver@{host_name}"

    server = Server()
    await server.init()
    await server.set_application_uri(server_app_uri)
    server.set_endpoint("opc.tcp://0.0.0.0:4840/freeopcua/server/")
    server.set_security_policy([ua.SecurityPolicyType.Basic256Sha256_SignAndEncrypt])
    # load server certificate and private key. This enables endpoints
    # with signing and encryption.
    await server.load_certificate(str(server_cert))
    await server.load_private_key(str(server_private_key))

    if USE_TRUST_STORE:
        trust_store = TrustStore(
            [Path("examples") / "certificates" / "trusted" / "certs"], []
        )
        await trust_store.load()
        validator = CertificateValidator(
            options=CertificateValidatorOptions.TRUSTED_VALIDATION
            | CertificateValidatorOptions.PEER_CLIENT,
            trust_store=trust_store,
        )
    else:
        validator = CertificateValidator(
            options=CertificateValidatorOptions.EXT_VALIDATION
            | CertificateValidatorOptions.PEER_CLIENT
        )
    server.set_certificate_validator(validator)

    idx = 0

    # populating our address space
    myobj = await server.nodes.objects.add_object(idx, "MyObject")
    myvar = await myobj.add_variable(idx, "MyVariable", 0.0)
    await myvar.set_writable()  # Set MyVariable to be writable by clients

    # starting!

    async with server:
        while True:
            await asyncio.sleep(1)
            current_val = await myvar.get_value()
            count = current_val + 0.1
            await myvar.write_value(count)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    asyncio.run(main())
