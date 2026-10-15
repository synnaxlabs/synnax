#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Runs the OPC UA test server: ``python -m examples.opcua.server [--tls]``."""

import argparse
import asyncio

from examples.opcua import OPCUATLSSim, run_server

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="OPC UA test server")
    parser.add_argument(
        "--tls", action="store_true", help="Enable TLS encryption (port 4842)"
    )
    args = parser.parse_args()

    if args.tls:
        asyncio.run(run_server(endpoint=OPCUATLSSim.endpoint, encrypted=True))
    else:
        asyncio.run(run_server())
