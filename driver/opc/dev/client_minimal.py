#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio

import synnax as sy
from asyncua import Client

url = "opc.tcp://localhost:4842/freeopcua/server/"
namespace = "http://examples.freeopcua.github.io"


async def main():
    print(f"Connecting to {url} ...")
    async with Client(url=url) as client:
        print(await client.nodes.root.get_children())
        # Find the namespace index
        nsidx = await client.get_namespace_index(namespace)
        print(f"Namespace Index for '{namespace}': {nsidx}")

        sy_client = sy.Synnax()

        ch = sy_client.channels.create(
            name="time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        data = sy_client.channels.create(
            name="data",
            data_type=sy.DataType.FLOAT64,
            index=ch.key,
            retrieve_if_name_exists=True,
        )

        # Get the variable node for read / write
        var = await client.nodes.root.get_child(
            f"0:Objects/{nsidx}:MyObject/{nsidx}:MyVariable"
        )

        with sy_client.new_writer(sy.TimeStamp.now(), [ch.key, data.key]) as w:
            while True:
                value = await var.read_value()
                print(f"Value of MyVariable ({var}): {value}")
                w.write({ch.key: sy.TimeStamp.now(), data.key: value})
                await asyncio.sleep(0.01)
        #
        # new_value = value - 50
        # print(f"Setting value of MyVariable to {new_value} ...")
        # await var.write_value(new_value)

        # # Calling a method
        # res = await client.nodes.objects.call_method(f"{nsidx}:ServerMethod", 5)
        # print(f"Calling ServerMethod returned {res}")


if __name__ == "__main__":
    asyncio.run(main())
