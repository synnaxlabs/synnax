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
import random

from asyncua import Server, ua

# Configuration constants
ARRAY_COUNT = 5
ARRAY_SIZE = 5
FLOAT_COUNT = 5
BOOL_COUNT = 5
RATE = 100  # Hz
BOOL_OFFSET = 0.2  # seconds between each boolean transition

# Error injection configuration
ERROR_INJECTION_RATE = 0.1  # 10% error rate
ERROR_ARRAY_INDEX = 2  # Which array to inject errors into
ERROR_FLOAT_INDEX = 2  # Which float to inject errors into


# Initialization Functions
async def create_array_variables(myobj, idx):
    """Create array variables with initial values."""
    arrays = []
    for i in range(ARRAY_COUNT):
        initial_values = [float(j + i) for j in range(ARRAY_SIZE)]
        arr = await myobj.add_variable(
            idx, f"my_array_{i}", initial_values, ua.VariantType.Float
        )
        await arr.write_array_dimensions([ARRAY_SIZE])
        arrays.append(arr)
    return arrays

async def create_time_array(myobj, idx):
    """Create timestamp array variable."""
    now = datetime.datetime.now(datetime.timezone.utc)
    initial_times = [
        now + datetime.timedelta(milliseconds=j) for j in range(ARRAY_SIZE)
    ]
    mytimearray = await myobj.add_variable(
        idx, "my_time_array", initial_times, ua.VariantType.DateTime
    )
    await mytimearray.set_writable()
    await mytimearray.write_array_dimensions([ARRAY_SIZE])
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



# Data Generation Functions
def generate_timestamps(start, rate, size):
    """Generate array of timestamps."""
    return [
        start + datetime.timedelta(seconds=j * (1 / rate)) for j in range(size)
    ]

def generate_sinewave_values(timestamps, start_ref):
    """Generate sinewave values based on timestamps."""
    return [
        math.sin((timestamp - start_ref).total_seconds())
        for timestamp in timestamps
    ]



# Update Functions
def inject_error(values):
    """
    Generate corrupted array data for error injection.
    """

    error_chance = random.random()
    
    # Empty array
    if error_chance < 0.333:
        return []

    # Array smaller than expected
    elif error_chance < 0.667:
        size = random.randint(1, ARRAY_SIZE - 2)
        return values[:size]

    # Array larger than expected
    else: 
        extra = random.randint(1, 3)
        return values + [values[-1] for _ in range(extra)]

async def update_arrays(arrays, values, start_ref, cycle_count):
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



async def main():
    # Initialize server
    server = Server()
    await server.init()
    server.set_endpoint("opc.tcp://localhost:4841/freeopcua/server/")
    uri = "http://examples.freeopcua.github.io"
    idx = await server.register_namespace(uri)

    # Create OPC UA object and variables
    myobj = await server.nodes.objects.add_object(idx, "MyObject")
    arrays = await create_array_variables(myobj, idx)
    mytimearray = await create_time_array(myobj, idx)
    floats = await create_float_variables(myobj, idx)
    bools = await create_bool_variables(myobj, idx)

    # Start server loop
    start_ref = datetime.datetime.now(datetime.timezone.utc)
    cycle_count = 0

    async with server:
        while True:
            cycle_count += 1
            start = datetime.datetime.now(datetime.timezone.utc)
            elapsed = (start - start_ref).total_seconds()

            # Generate data
            timestamps = generate_timestamps(start, RATE, ARRAY_SIZE)
            sinewave_values = generate_sinewave_values(timestamps, start_ref)

            # Update all variables
            await update_arrays(arrays, sinewave_values, start_ref, cycle_count)
            await mytimearray.set_value(
                timestamps, varianttype=ua.VariantType.DateTime
            )
            await update_floats(floats, elapsed)
            await update_bools(bools, elapsed)

            # Sleep to maintain rate
            duration = (
                datetime.datetime.now(datetime.timezone.utc) - start
            ).total_seconds()
            await asyncio.sleep((1 / RATE) - duration)


if __name__ == "__main__":
    asyncio.run(main(), debug=True)
