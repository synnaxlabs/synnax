import synnax as sy

client = sy.Synnax()

with client.control.acquire(["valve_1"], ["temp"]) as control:
    while control.receives_data():
        if control.temp > 35:
            control.valve_1.open()
        elif control.temp < 30:
            control.valve_1.close()
