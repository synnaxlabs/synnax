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


async def main():
    server = Server()
    await server.init()
    server.set_endpoint("opc.tcp://localhost:4841/freeopcua/server/")
    uri = "http://examples.freeopcua.github.io"
    idx = await server.register_namespace(uri)

    # Populating our address space
    myobj = await server.nodes.objects.add_object(idx, "MyObject")
    ARRAY_COUNT = 5
    arrays = list()
    for i in range(ARRAY_COUNT):
        arrays.append(
            await myobj.add_variable(
                idx, f"my_array_{i}", [1, 2, 3, 4, 5, 6, 7, 8], ua.VariantType.Float
            )
        )
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

    RATE = 500
    ARRAY_SIZE = 5
    await mytimearray.write_array_dimensions([ARRAY_SIZE])

    for i in range(5):
        # add 30 float variables t OPC
        my_float = await myobj.add_variable(
            idx, f"my_float_{i}", i, ua.VariantType.Float
        )
        await my_float.set_writable()
    for i in range(5):
        # add 30 float variables t OPC
        my_float = await myobj.add_variable(
            idx, f"my_bool_{i}", i, ua.VariantType.Boolean
        )
        await my_float.set_writable()

    i = 0
    start_ref = datetime.datetime.now(datetime.timezone.utc)
    async with server:
        while True:
            i += 1
            start = datetime.datetime.now(datetime.timezone.utc)
            timestamps = [
                start + datetime.timedelta(seconds=j * ((1 / RATE)))
                for j in range(ARRAY_SIZE)
            ]
            values = [
                math.sin((timestamps[j] - start_ref).total_seconds())
                for j in range(ARRAY_SIZE)
            ]
            for i, arr in enumerate(arrays):
                await arr.set_value(
                    [v + i for v in values], varianttype=ua.VariantType.Float
                )
            await mytimearray.set_value(timestamps, varianttype=ua.VariantType.DateTime)
            duration = (
                datetime.datetime.now(datetime.timezone.utc) - start
            ).total_seconds()
            await asyncio.sleep((1 / RATE) - duration)


if __name__ == "__main__":
    asyncio.run(main(), debug=True)
