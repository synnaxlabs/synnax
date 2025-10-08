#  Copyright 2025 Synnax Labs, Inc.
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

from asyncua import Server, ua

# Configuration constants
ARRAY_COUNT = 5
ARRAY_SIZE = 5
FLOAT_COUNT = 5
BOOL_COUNT = 5
RATE = 100  # Hz
BOOL_OFFSET = 0.2  # seconds between each boolean transition


async def main():
    server = Server()
    await server.init()
    server.set_endpoint("opc.tcp://localhost:4841/freeopcua/server/")
    uri = "http://examples.freeopcua.github.io"
    idx = await server.register_namespace(uri)

    # Populating our address space
    myobj = await server.nodes.objects.add_object(idx, "MyObject")
    arrays = list()
    for i in range(ARRAY_COUNT):
        initial_values = [float(j + i) for j in range(ARRAY_SIZE)]
        arrays.append(
            await myobj.add_variable(
                idx, f"my_array_{i}", initial_values, ua.VariantType.Float
            )
        )
        await arrays[i].write_array_dimensions([ARRAY_SIZE])
    mytimearray = await myobj.add_variable(
        idx,
        "my_time_array",
        [
            datetime.datetime.now(datetime.timezone.utc),
            datetime.datetime.now(datetime.timezone.utc)
            + datetime.timedelta(milliseconds=1),
            datetime.datetime.now(datetime.timezone.utc)
            + datetime.timedelta(milliseconds=2),
            datetime.datetime.now(datetime.timezone.utc)
            + datetime.timedelta(milliseconds=3),
            datetime.datetime.now(datetime.timezone.utc)
            + datetime.timedelta(milliseconds=4),
            datetime.datetime.now(datetime.timezone.utc)
            + datetime.timedelta(milliseconds=5),
            datetime.datetime.now(datetime.timezone.utc)
            + datetime.timedelta(milliseconds=6),
            datetime.datetime.now(datetime.timezone.utc)
            + datetime.timedelta(milliseconds=7),
            datetime.datetime.now(datetime.timezone.utc)
            + datetime.timedelta(milliseconds=8),
        ],
        ua.VariantType.DateTime,
    )

    await mytimearray.set_writable()
    await mytimearray.write_array_dimensions([ARRAY_SIZE])

    floats = []
    for i in range(FLOAT_COUNT):
        my_float = await myobj.add_variable(
            idx, f"my_float_{i}", 0.0, ua.VariantType.Float
        )
        await my_float.set_writable()
        floats.append(my_float)

    bools = []
    for i in range(BOOL_COUNT):
        my_bool = await myobj.add_variable(
            idx, f"my_bool_{i}", False, ua.VariantType.Boolean
        )
        await my_bool.set_writable()
        bools.append(my_bool)

    i = 0
    start_ref = datetime.datetime.now(datetime.timezone.utc)
    async with server:
        while True:
            i += 1
            start = datetime.datetime.now(datetime.timezone.utc)
            elapsed = (start - start_ref).total_seconds()
            timestamps = [
                start + datetime.timedelta(seconds=j * ((1 / RATE)))
                for j in range(ARRAY_SIZE)
            ]
            values = [
                math.sin((timestamps[j] - start_ref).total_seconds())
                for j in range(ARRAY_SIZE)
            ]
            for i, arr in enumerate(arrays):
                if i == 2:
                    # This simulates the PLC buffer being empty but with Good status.
                    cycle_count = int((datetime.datetime.now(datetime.timezone.utc) - start_ref).total_seconds() * RATE)
                    if cycle_count % 10 == 0:  # Every 10th cycle, return empty array with Good status
                        await arr.set_value([], varianttype=ua.VariantType.Float)
                    else:
                        await arr.set_value([v + i for v in values], varianttype=ua.VariantType.Float)
                else:
                    # Other arrays always return full arrays
                    await arr.set_value([v + i for v in values], varianttype=ua.VariantType.Float)

            await mytimearray.set_value(timestamps, varianttype=ua.VariantType.DateTime)

            # Update floats with sinewaves (offset like arrays)
            for idx, float_var in enumerate(floats):
                await float_var.set_value(math.sin(elapsed) + idx, varianttype=ua.VariantType.Float)

            # Update booleans with 1Hz square waves (sequential pattern)
            for idx, bool_var in enumerate(bools):
                offset_elapsed = elapsed + (idx * BOOL_OFFSET)
                square_wave = int(offset_elapsed) % 2 == 0
                await bool_var.set_value(square_wave, varianttype=ua.VariantType.Boolean)

            duration = (
                datetime.datetime.now(datetime.timezone.utc) - start
            ).total_seconds()
            await asyncio.sleep((1 / RATE) - duration)


if __name__ == "__main__":
    asyncio.run(main(), debug=True)
