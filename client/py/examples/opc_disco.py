import synnax as sy
import json

client = sy.Synnax()
devices = client.hardware.devices.retrieve(names=[])
output = dict()

output["devices"] = dict()

for d in devices:
    print(d)
    output[d.key] = {
        "name": d.name,
        "properties": json.loads(d.properties),
    }

tasks = client.hardware.tasks.retrieve(types=["opc_read"])
output["tasks"] = dict()
for t in tasks:
    output["tasks"][t.key] = {
        "name": t.name,
        "type": t.type,
        "properties": json.loads(t.config),
    }

channels = client.channels.retrieve(channel=["*"])
output["channels"] = list()
for c in channels:
    output["channels"].append(
        {
            "name": c.name,
            "data_type": c.data_type,
            "key": c.key,
            "is_index": c.is_index,
            "virtual": c.virtual,
        }
    )
json.dump(output, open("opc_remap.json", "w"))


