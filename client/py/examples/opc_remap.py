import synnax as sy
import json

client = sy.Synnax()

DEV_KEY = "8b68fb06-94cf-45d1-966e-9daccfe71558"
TASK_ID = 281479271677956

rack = client.hardware.racks.retrieve()
opc_server_dev = client.hardware.devices.retrieve(key=DEV_KEY)
task = client.hardware.tasks.retrieve(key=TASK_ID)

task_cfg = json.loads(task.config)
dev_properties = json.loads(opc_server_dev.properties)

REPLACEMENTS = {
    "fbFacilityGas": "fb_facility_gas",
    "fbFacility": "fb_facility",
    "fbAirPlant": "fb_air_plant",
    "fbEngine": "fb_engine",
    "fbPlasma": "fb_plasma",
    "fbLiquidCart": "fb_liquid_cart",
    "fbAirPlant": "fb_air_plant",
    "fbWaterSystem": "fb_water_system",
    "fbVitiator": "fb_vitiator",
    "fbDaq": "fb_daq",
    "fbGasFuel": "fb_gas_fuel",
    "plasmaChannelBuffers": "plasma_channel_buffers",
    "opcArr": "opc_arr",
    "AnalogInputSensors": "Analog_input_sensors",
    "fbTC": "fb_tC",
    "fbPT": "fb_pT",
    "fbMFR": "fb_mFR",
    "fbLC": "fb_lC",
    "fbGA": "fb_gA",
}

IDX_KEY = 1048581

for c in task_cfg["channels"]:
    ch = client.channels.retrieve(c["channel"])
    c["name"] = c["node_name"]
    c["data_type"] = ch.data_type
    node_id = c["node_id"]
    for k, v in REPLACEMENTS.items():
        if k in node_id:
            node_id = node_id.replace(k, v)
    dev_properties["read"]["channels"][node_id] = c["channel"]
    dev_properties["read"]["index"] = IDX_KEY
    dev_properties["read"]["indexes"] = [IDX_KEY]


client.hardware.devices.create(
    key=opc_server_dev.key,
    location=opc_server_dev.location,
    rack=opc_server_dev.rack,
    name=opc_server_dev.name,
    make=opc_server_dev.make,
    model=opc_server_dev.model,
    properties=json.dumps(dev_properties),
)

client.hardware.tasks.create(
    name=f"{task.name} (Remapped V2)",
    rack=rack.key,
    type=task.type,
    config=json.dumps(task_cfg),
)