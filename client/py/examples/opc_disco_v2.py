import synnax as sy
import json

client = sy.Synnax()

DEV_KEY = "8b68fb06-94cf-45d1-966e-9daccfe71558"
TASK_ID = 281479271677956

# opc_server_dev = client.hardware.devices.retrieve(key=DEV_KEY)
# task = client.hardware.tasks.retrieve(key=TASK_ID)

# task_cfg = json.loads(task.config)
# dev_properties = json.loads(opc_server_dev.properties)

data = json.load(open("/Users/emilianobonilla/Downloads/opc_remap.json"))

dev_properties = data[DEV_KEY]["properties"]
task_cfg = data["tasks"][f"{TASK_ID}"]["properties"]

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

for c in task_cfg["channels"]:
    ch = client.channels.retrieve(c["channel"])
    c["name"] = c["node_name"]
    c["data_type"] = ch.data_type
    node_id = c["node_id"]
    for k, v in REPLACEMENTS.items():
        if k in node_id:
            node_id = node_id.replace(k, v)
    dev_properties["read"]["channels"][node_id] = c["channel"]

json.dump(task_cfg, open("opc_task_cfg.json", "w"))
json.dump(dev_properties, open("opc_dev_props.json", "w"))