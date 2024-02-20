import synnax as sy
from uuid import uuid4
import json

client = sy.Synnax()

# Create a rack
rack = client.hardware.create_rack([
    sy.Rack(
        key=0,
        name="Rack 1"
    )
])

# Create a device
client.hardware.create_device([
    sy.Device(
        key=str(uuid4()),
        rack=rack[0].key,
        name="PXI-6255",
        description="a new device",
        make="ni",
        model="PXI-6255",
        properties="{}",
    )
])

with client.new_streamer(["sy_task_set"]) as streamer:
    for f in streamer:
        print(list(f["sy_task_set"]))
        t = client.hardware.retrieve_task(None, list(f["sy_task_set"]))
        f = open("task.json", "w")
        f.write(json.dumps(json.loads(t[0].config), indent=4))
