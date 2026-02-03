#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import serial

import synnax as sy

PORT = "/dev/cu.usbmodem21401"
BAUD_RATE = 9600

# Create the Synnax client
client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
)

# Create the index channel
index_channel = client.channels.create(
    name="arduino_time",
    is_index=True,
    data_type="timestamp",
    retrieve_if_name_exists=True,
)

# Create the data channel
data_channel = client.channels.create(
    name="arduino_value",
    index=index_channel.key,
    data_type="float32",
    retrieve_if_name_exists=True,
)

# Set up the serial connection
ser = serial.Serial(PORT, BAUD_RATE)
if ser.is_open:
    print("Serial connection established")
else:
    print("Failed to establish serial connection")

# Open a writer and continuously read from the Arduino
with client.open_writer(
    start=sy.TimeStamp.now(),
    channels=["arduino_time", "arduino_value"],
) as writer:
    while True:
        # Read from the serial connection
        value = float(ser.readline().decode("utf-8").rstrip())
        print(value)
        writer.write({"arduino_time": sy.TimeStamp.now(), "arduino_value": value})
