import json

import requests

bluefors_url = "https://localhost:49099"
params = {"key": ""}  # TODO: put your api key here


values_payload = {"data": {"mapper.bf.valves.v21": {"content": {"value": "1"}}}}
values_url = bluefors_url + "/values/devices/mapper/bf/valves/v21"

driver_payload = {"data": {"driver.vc.ch.c44": {"content": {"value": "1"}}}}
driver_url = bluefors_url + "/values/devices/driver/vc/ch/c44"


r = requests.post(driver_url, params=params, json=driver_payload, verify=False)
with open("response.json", "w") as f:
    json.dump(r.json(), f)

# Other things to try if this doesn't work:
# 1. Try changing "value": "1" (string) to True (boolean) or 1 (number)
# 2. Use different URL endpoints: "/values", "/values/devices", "values/devices/driver",
#    "values/devices/driver/vc", "values/devices/driver/vc/ch"
