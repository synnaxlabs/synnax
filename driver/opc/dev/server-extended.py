import asyncio
import logging
import datetime
import math
import time

from asyncua import Server, ua
from asyncua.common.methods import uamethod


@uamethod
def multiply_by_two(parent, value):
    return value * 2


#
# async def setup_users(server):
#     # Define a user with a username and password
#     await server.user_manager.set_user_manager(user_manager)
#     server.user_manager.user_manager.add_user('username', 'password')
#
# def user_manager(isession, username, password):
#     # This is a simple user validation function; customize as needed
#     if username == 'username' and password == 'password':
#         return True
#     logging.warning(f"Unauthorized login attempt: {username}")
#     return False

async def main():
    # Setup our server
    server = Server()
    await server.init()
    server.set_endpoint("opc.tcp://localhost:4841/freeopcua/server/")
    # server.set_security_policy([
    #     ua.SecurityPolicyType.Basic256Sha256_SignAndEncrypt
    # ])

    # Register a namespace
    uri = "http://examples.freeopcua.github.io"
    idx = await server.register_namespace(uri)

    # await setup_users(server)

    # Populating our address space
    myobj = await server.nodes.objects.add_object(idx, "MyObject")

    # Add different types of variables
    myval = await myobj.add_variable(idx, "MyVariable", 6.7)
    myarray = await myobj.add_variable(idx, "MyArray", [1, 2, 3, 4, 5, 6, 7, 8], ua.VariantType.Float)
    mytimearray = await myobj.add_variable(idx, "MyTimeArray", [
        datetime.datetime.utcnow(),
        datetime.datetime.utcnow() + datetime.timedelta(milliseconds=1),
        datetime.datetime.utcnow() + datetime.timedelta(milliseconds=2),
        datetime.datetime.utcnow() + datetime.timedelta(milliseconds=3),
        datetime.datetime.utcnow() + datetime.timedelta(milliseconds=4),
        datetime.datetime.utcnow() + datetime.timedelta(milliseconds=5),
        datetime.datetime.utcnow() + datetime.timedelta(milliseconds=6),
        datetime.datetime.utcnow() + datetime.timedelta(milliseconds=7),
        datetime.datetime.utcnow() + datetime.timedelta(milliseconds=8),
    ], ua.VariantType.DateTime)

    myarray.set_writable()
    mytimearray.set_writable()

    RATE = 36*20*12
    ARRAY_SIZE = 40*8
    mytimearray.write_array_dimensions([ARRAY_SIZE])
    myarray.write_array_dimensions([ARRAY_SIZE])

    # Update values every second
    i = 0
    start_ref = datetime.datetime.utcnow()
    async with server:
        while True:
            i += 1
            start = datetime.datetime.utcnow()
            timestamps = [start + datetime.timedelta(seconds=j * ((1 / RATE))) for j in range(ARRAY_SIZE)]
            values = [math.sin((timestamps[j] - start_ref).total_seconds()) for j in range(ARRAY_SIZE)]
            await myarray.set_value(values, varianttype=ua.VariantType.Float)
            await mytimearray.set_value(timestamps, varianttype=ua.VariantType.DateTime)
            duration = (datetime.datetime.utcnow() - start).total_seconds()
            await asyncio.sleep((1/RATE) - duration)


if __name__ == "__main__":
    asyncio.run(main(), debug=True)
