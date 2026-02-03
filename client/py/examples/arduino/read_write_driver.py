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

ser = serial.Serial(PORT, BAUD_RATE)

client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
)

arduino_command = client.channels.create(
    name="arduino_command",
    data_type="uint8",
    virtual=True,
    retrieve_if_name_exists=True,
)

arduino_time = client.channels.create(
    name="arduino_time",
    is_index=True,
    data_type="timestamp",
    retrieve_if_name_exists=True,
)

arduino_state = client.channels.create(
    name="arduino_state",
    index=arduino_time.key,
    data_type="uint8",
    retrieve_if_name_exists=True,
)

arduino_value = client.channels.create(
    name="arduino_value",
    index=arduino_time.key,
    data_type="float32",
    retrieve_if_name_exists=True,
)

with client.open_streamer(["arduino_command"]) as streamer:
    with client.open_writer(
        start=sy.TimeStamp.now(),
        channels=["arduino_time", "arduino_state", "arduino_value"],
    ) as writer:
        while True:
            frame = streamer.read(timeout=0)
            if frame is not None:
                command = str(frame["arduino_command"][0])
                ser.write(command.encode("utf-8"))
            data = ser.readline().decode("utf-8").rstrip()
            if data:
                split = data.split(",")
                writer.write(
                    {
                        "arduino_time": sy.TimeStamp.now(),
                        "arduino_state": int(split[0]),
                        "arduino_value": float(split[1]),
                    }
                )
