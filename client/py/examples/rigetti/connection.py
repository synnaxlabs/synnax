import synnax as sy
import requests

bluefors_endpoint = "https://localhost:49099"
synnax_client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="wseldon",
)

name = "v16"

api_key = "4"   

endpoint = bluefors_endpoint + "values/devices/v16"


r = requests.get(endpoint,params={"key": api_key, "fields": "name,value"})
print(r.json())