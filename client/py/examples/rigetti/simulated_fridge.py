import synnax as sy
from typing import TypedDict


class Sensors(TypedDict):
    p1: float
    p6: float


class Valves(TypedDict):
    v14: bool
    v16: bool
    v19: bool
    v20: bool
    v21: bool
    v22: bool
    v23: bool
    turbo_pump: bool
    scroll_2_pump_state: bool


# Create channels for the sensors, valves, and pumps.


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
)

time_channel = client.channels.create(
    name="fridge_simulation_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

sensor_channels = [
    sy.Channel(
        name=sensor,
        data_type=sy.DataType.FLOAT32,
        index=time_channel.key,
    )
    for sensor in Sensors.keys()
]

valve_state_channels = [
    sy.Channel(
        name=valve,
        data_type=sy.DataType.UINT8,
        index=time_channel.key,
    )
    for valve in Valves.keys()
]

client.channels.create(
    [*sensor_channels, *valve_state_channels], retrieve_if_name_exists=True
)

valve_state_channels = client.channels.create(
    valve_state_channels, retrieve_if_name_exists=True
)

## Create a cmd_time and cmd channel for each valve and pump.
valve_cmd_time_channels = [
    sy.Channel(
        name=f"{valve}_cmd_time",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
    )
    for valve in Valves.keys()
]

valve_cmd_time_channels = client.channels.create(
    valve_cmd_time_channels, retrieve_if_name_exists=True
)


client.channels.create(
    [
        sy.Channel(
            name=f"{valve}_cmd",
            data_type=sy.DataType.UINT8,
            index=valve_cmd_time_channels[i].key,
        )
        for i, valve in enumerate(Valves.keys())
    ],
    retrieve_if_name_exists=True,
)


class DaqState:
    sensors: Sensors
    valves: Valves

    def to_frame(self) -> sy.Frame:
        return sy.Frame(
            {
                time_channel.name: sy.TimeStamp.now(),
                **self.sensors,
                **{f"{valve}_state": value for valve, value in self.valves.items()},
            }
        )


# Now we have all of the channels, we will want to have a loop that reads from the
# command channels and updates the state and sensor values as appropriate.

loop = sy.Loop(sy.Rate.HZ * 100)


atmospheric_pressure = 1013.25  # mBar


def update_state(
    daq_state: dict[str, float | bool], frame: sy.Frame | None
) -> dict[str, float | bool]:
    ### Implement physics logic in this function.
    daq_state[time_channel.name] = sy.TimeStamp.now()
    if frame is not None:
        for channel in frame.channels:
            series = frame[channel]
            val = series[-1]
            print(f"Received command: {channel}={val}")
            state_channel_name = str(channel).replace("_cmd", "_state")
            daq_state[state_channel_name] = val
    ## determine p1 and p2 pressure based on v14, v16, and v19 states
    v14_and_v16_open = bool(daq_state["V14_state"]) and bool(daq_state["V16_state"])
    v19_open = bool(daq_state["V19_state"])
    if v14_and_v16_open and v19_open:
        daq_state["p1"] += (atmospheric_pressure - daq_state["p1"]) * 0.01
        daq_state["p6"] += (atmospheric_pressure - daq_state["p6"]) * 0.01
    elif v14_and_v16_open:
        # make p1 and p6 move towards each other
        daq_state["p1"] += (daq_state["p6"] - daq_state["p1"]) * 0.01
        daq_state["p6"] += (daq_state["p1"] - daq_state["p6"]) * 0.01
    elif v19_open:
        # make p6 move towards atmospheric pressure
        daq_state["p6"] += (atmospheric_pressure - daq_state["p6"]) * 0.01
    else:
        # do nothing, everything is closed
        pass
    return daq_state


streaming_channels = [ch.name for ch in valve_cmd_channels] + [
    ch.name for ch in pump_cmd_channels
]

writing_channels: list[str] = [*Sensors.keys(), *Valves.keys()]


with client.open_streamer(streaming_channels) as streamer:
    with client.open_writer(
        start=sy.TimeStamp.now(),
        channels=writing_channels,
        name="fridge_simulation",
    ) as writer:
        daq_state: dict[str, float | bool] = {
            "p1": 3,
            "p6": 0.4,
            **{f"{valve}_state": False for valve in VALVES},
            **{f"{pump}_state": False for pump in PUMPS},
            **{time_channel.name: sy.TimeStamp.now()},
        }
        while loop.wait():
            frame = streamer.read(timeout=0)
            daq_state = update_state(daq_state, frame)
            writer.write(daq_state)
