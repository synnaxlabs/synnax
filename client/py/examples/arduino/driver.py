import synnax as sy
import serial

ARDUINO_PORT = "/dev/cu.usbmodem21401"  # Replace with your Arduino's port (e.g., COM3,
ARDUINO_BAUD_RATE = 9600
client = sy.Synnax()
index_channel = client.channels.create(
    name="arduino_time",
    is_index=True,
    retrieve_if_name_exists=True
)
data_channel = client.channels.create(
    name="arduino_data",
    data_type="float32",
    index=index_channel.key,
    retrieve_if_name_exists=True
)
with client.open_writer(sy.TimeStamp.now(), ["arduino_time", "arduino_data"]) as w:
    ser = serial.Serial(ARDUINO_PORT, ARDUINO_BAUD_RATE)
    print(ser.is_open)
    while True:
        line = ser.readline().decode("utf-8").strip()
        w.write({"arduino_time": sy.TimeStamp.now(), "arduino_data": float(line)})
