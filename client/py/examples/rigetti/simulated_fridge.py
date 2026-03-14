from typing import Literal

import synnax as sy

client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
)

# Declare pieces of hardware that we will use to create channels.
SENSORS: dict[str, float] = {
    "p1": 1013.25,
    "p2": 1013.25,
    "p6": 1013.25,
    "t4k": 4,
    "tstill": 2,
    "aux": 1013.25,
    "vent": 1013.25,
}

ACTUATORS = (
    "v14",
    "v15",
    "v16",
    "v18",
    "v19",
    "v20",
    "v21",
    "heater_wu",
    "heater_4k",
    "turbo_pump",
    "scroll_2_pump",
)


# Create channels for the sensors, valves, and pumps.

time_channel = client.channels.create(
    name="fridge_simulation_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)


sensor_channels = client.channels.create(
    [
        sy.Channel(name=sensor, data_type=sy.DataType.FLOAT32, index=time_channel.key)
        for sensor in SENSORS.keys()
    ],
    retrieve_if_name_exists=True,
)

sensor_override_channels = client.channels.create(
    [
        sy.Channel(
            name=f"{sensor}_override", data_type=sy.DataType.FLOAT32, virtual=True
        )
        for sensor in SENSORS.keys()
    ],
    retrieve_if_name_exists=True,
)


actuator_state_channels = client.channels.create(
    [
        sy.Channel(name=actuator, data_type=sy.DataType.UINT8, index=time_channel.key)
        for actuator in ACTUATORS
    ],
    retrieve_if_name_exists=True,
)

actuator_cmd_channels = client.channels.create(
    [
        sy.Channel(name=f"{actuator}_cmd", data_type=sy.DataType.UINT8, virtual=True)
        for actuator in ACTUATORS
    ],
    retrieve_if_name_exists=True,
)

# Now we have all of the channels, we will want to have a loop that reads from the
# command channels and updates the state and sensor values as appropriate.

loop = sy.Loop(sy.Rate.HZ * 10)


atmospheric_pressure_mbar = 1013.25
atmospheric_temp_k = 320


def update_state(
    state: dict[str, float | bool], frame: sy.Frame | None
) -> dict[str, float | bool]:
    state[time_channel.name] = sy.TimeStamp.now()
    if frame is not None:
        for channel in frame.channels:
            series = frame[channel]
            val = series[-1]
            if channel.endswith("_override"):
                sensor_name = channel.removesuffix("_override")
                print(f"Setting {sensor_name} to {float(val)}")
                state[sensor_name] = val
                continue
            actuator_name = channel.removesuffix("_cmd")
            is_open = bool(val)
            is_valve = channel.startswith("v")
            if is_valve:
                print(f"{'Opening' if is_open else 'Closing'} {actuator_name}")
            else:
                print(f"{'Activating' if is_open else 'De-activating'} {actuator_name}")
            state[actuator_name] = val

    next_state = state.copy()

    def decrease_sensor(sensor: str):
        next_state[sensor] -= 0.01 * state[sensor]

    def move_towards(sensor1: str, sensor2: str):
        delta = 0.01 * (next_state[sensor2] - next_state[sensor1])
        next_state[sensor1] += delta
        next_state[sensor2] -= delta

    def move_from(sensor1: str, sensor2: str):
        amount = 0.01 * state[sensor1]
        next_state[sensor1] -= amount
        next_state[sensor2] += amount

    def move_towards_target(sensor: str, target: float, amount: float = 0.01):
        delta = amount * (target - next_state[sensor])
        next_state[sensor] += delta

    ## Implement physics logic here

    if state["heater_4k"]:
        move_towards_target("t4k", atmospheric_temp_k)
        move_towards_target("tstill", atmospheric_temp_k)
    if state["heater_wu"]:
        move_towards_target("t4k", atmospheric_temp_k, 0.001)
        move_towards_target("tstill", atmospheric_temp_k, 0.001)

    if state["v14"] and state["v16"]:
        move_towards("p1", "p6")
    if state["v15"] and state["v16"]:
        move_towards("p2", "p6")
    if state["v14"] and state["v15"]:
        move_towards("p1", "p2")
    if state["v19"]:
        move_towards_target("p6", state["vent"])
    if state["v20"]:
        move_towards_target("p6", state["aux"])
    if state["v18"] and state["turbo_pump"]:
        move_from("p2", "p6")
    if state["v21"] and state["scroll_2_pump"]:
        decrease_sensor("p6")

    return next_state


streaming_channels = [ch.name for ch in actuator_cmd_channels] + [
    ch.name for ch in sensor_override_channels
]

writing_channels = (
    [ch.name for ch in actuator_state_channels]
    + [time_channel.name]
    + [ch.name for ch in sensor_channels]
)

with client.open_streamer(streaming_channels) as streamer:
    with client.open_writer(
        start=sy.TimeStamp.now(),
        channels=writing_channels,
        name="fridge_simulation",
    ) as writer:
        state: dict[str, float | bool | sy.TimeStamp] = {
            **SENSORS,
            **{actuator: False for actuator in ACTUATORS},
            **{time_channel.name: sy.TimeStamp.now()},
        }
        while loop.wait():
            frame = streamer.read(timeout=0)
            state = update_state(state, frame)
            writer.write(state)
