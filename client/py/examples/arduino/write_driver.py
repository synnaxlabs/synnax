import synnax as sy
import serial

PORT = "/dev/cu.usbmodem21401"
BAUD_RATE = 9600

ser = serial.Serial(PORT, BAUD_RATE)

client = sy.Synnax(host="localhost", port=9090, username="synnax", password="seldon")

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
