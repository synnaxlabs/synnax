import uuid

import synnax as sy

client = sy.Synnax()

rack = client.hardware.create_rack([sy.Rack(name="gse")])

client.hardware.create_device(
    [
        sy.Device(
            key="130227d9-02aa-47e4-b370-0d590add1bc1",
            rack=rack[0].key,
            name="New PXI-6255",
            make="NI",
            model="PXI-6255",
            location="dev1",
            identifier="dev1",
        )
    ]
)

# client.hardware.create_task([
#     sy.Task(
#         key=rack[0].key,
#         name="Analog Read Task 1",
#         type="ni.analogRead",
#     )
# ])
