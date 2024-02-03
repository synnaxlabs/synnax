import synnax as sy
from uuid import uuid4

client = sy.Synnax()

# Create a device
client.hardware.create_device([
    sy.Device(
        key=str(uuid4()),
        name="cDAQ-9188",
        description="a new device",
        make="ni",
        model="cDAQ-9188",
        properties="{}",
    )
])
