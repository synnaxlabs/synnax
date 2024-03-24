import uuid

import synnax as sy

client = sy.Synnax()

rack = client.hardware.create_rack([sy.Rack(name="gse")])

client.hardware.create_device(
    [
        sy.Device(
            key=str(uuid.uuid4()), rack=rack[0].key, name="Device 1",
            model="PXI-6255", location="dev1", identifier="dev1"
        )
    ]
)

client.hardware.create_task([
    sy.Task(
        key=rack[0].key,
        name="Analog Read Task 1",
        type="ni.analogRead",
    )
])
