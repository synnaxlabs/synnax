import synnax as sy
import json

client = sy.Synnax()

dev = client.devices.retrieve(makes=["ethercat_slave"])

print([dev.model_dump() for dev in dev])