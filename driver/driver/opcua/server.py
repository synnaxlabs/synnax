import asyncio
import logging

from asyncua import Server, ua
from asyncua.common.methods import uamethod


@uamethod
def func(parent, value):
    return value * 2


async def main():
    _logger = logging.getLogger(__name__)
    # setup our server
    server = Server()
    await server.init()
    server.set_endpoint("opc.tcp://0.0.0.0:4840/freeopcua/server/")

    # set up our own namespace, not really necessary but should as spec
    uri = "http://examples.freeopcua.github.io"
    idx = await server.register_namespace(uri)
    myobj = await server.nodes.objects.add_object(idx, "MyObject")

    myvars = []
    for i in range(10):
        # populating our address space
        # server.nodes, contains links to very common nodes like objects and root
        myvar = await myobj.add_variable(
            idx,
            f"MyVariable{i}",
            6.7 + i,
            datatype=ua.NodeId(ua.ObjectIds.Double),
        )
        myvars.append(myvar)
    # Set MyVariable to be writable by clients
    async with server:
        while True:
            await asyncio.sleep(1)
            for myvar in myvars:
                new_val = await myvar.get_value() + 0.1
                _logger.info("Set value of %s to %.1f", myvar, new_val)
                await myvar.write_value(new_val)


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)
    asyncio.run(main(), debug=True)