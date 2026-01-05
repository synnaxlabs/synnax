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

command_channel = client.channels.create(
    name="arduino_command",
    data_type="uint8",
    virtual=True,
    retrieve_if_name_exists=True,
)

with client.open_streamer(["arduino_command"]) as streamer:
    for frame in streamer:
        command = str(frame["arduino_command"][0])
        print(f"Sending command: {command}")
        ser.write(command.encode("utf-8"))
