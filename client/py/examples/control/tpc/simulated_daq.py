import time

import synnax as sy

client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
)

DAQ_TIME = "daq_time"
TPC_CMD = "tpc_vlv_cmd"
TPC_ACK = "tpc_vlv_ack"
MPV_CMD = "mpv_vlv_cmd"
MPV_ACK = "mpv_vlv_ack"
PRESS_ISO_CMD = "press_iso_cmd"
PRESS_ISO_ACK = "press_iso_ack"
VENT_CMD = "vent_cmd"
VENT_ACK = "vent_ack"
PRESS_TANK_PT = "press_tank_pt"
FUEL_TANK_PT = "fuel_tank_pt"

daq_time = client.channels.create(
    name=DAQ_TIME,
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

tpc_cmd_time = client.channels.create(
    name="tpc_vlv_cmd_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

tpc_cmd = client.channels.create(
    [
        sy.Channel(name=TPC_CMD, data_type=sy.DataType.UINT8, index=tpc_cmd_time.key),
        sy.Channel(name=TPC_ACK, data_type=sy.DataType.UINT8, index=daq_time.key),
    ],
    retrieve_if_name_exists=True,
)

mpv_cmd_time = client.channels.create(
    name="mpv_vlv_cmd_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

mpv_cmd = client.channels.create(
    [
        sy.Channel(name=MPV_CMD, data_type=sy.DataType.UINT8, index=mpv_cmd_time.key),
        sy.Channel(name=MPV_ACK, data_type=sy.DataType.UINT8, index=daq_time.key),
    ],
    retrieve_if_name_exists=True,
)

press_iso_cmd_time = client.channels.create(
    name="press_iso_cmd_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

press_iso_cmd = client.channels.create(
    [
        sy.Channel(
            name=PRESS_ISO_CMD,
            data_type=sy.DataType.UINT8,
            index=press_iso_cmd_time.key,
        ),
        sy.Channel(name=PRESS_ISO_ACK, data_type=sy.DataType.UINT8, index=daq_time.key),
    ],
    retrieve_if_name_exists=True,
)

vent_cmd_time = client.channels.create(
    name="vent_cmd_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

vent_cmd = client.channels.create(
    [
        sy.Channel(name=VENT_CMD, data_type=sy.DataType.UINT8, index=vent_cmd_time.key),
        sy.Channel(name=VENT_ACK, data_type=sy.DataType.UINT8, index=daq_time.key),
    ],
    retrieve_if_name_exists=True,
)

client.channels.create(
    name=PRESS_TANK_PT,
    data_type=sy.DataType.FLOAT32,
    index=daq_time.key,
    retrieve_if_name_exists=True,
)

client.channels.create(
    name=FUEL_TANK_PT,
    data_type=sy.DataType.FLOAT32,
    index=daq_time.key,
    retrieve_if_name_exists=True,
)

rate = (sy.Rate.HZ * 50).period.seconds

DAQ_STATE = {
    # Valves
    TPC_CMD: 0,
    MPV_CMD: 0,
    PRESS_ISO_CMD: 0,
    VENT_CMD: 0,
    # Pts
    PRESS_TANK_PT: 0,
    FUEL_TANK_PT: 0,
}

MPV_LAST_OPEN = None
scuba_pressure = 0
l_stand_pressure = 0

with client.open_streamer(
    [
        TPC_CMD,
        MPV_CMD,
        PRESS_ISO_CMD,
        VENT_CMD,
    ]
) as streamer:
    with client.open_writer(
        sy.TimeStamp.now(),
        channels=[
            DAQ_TIME,
            TPC_ACK,
            MPV_ACK,
            PRESS_ISO_ACK,
            VENT_ACK,
            FUEL_TANK_PT,
            PRESS_TANK_PT,
        ],
    ) as w:
        i = 0
        while True:
            try:
                time.sleep(rate)
                if streamer.received:
                    while streamer.received:
                        f = streamer.read()
                        for k in f.channels:
                            print(k, f[k])
                            DAQ_STATE[k] = f[k][0]

                mpv_open = DAQ_STATE[MPV_CMD] == 1
                tpc_open = DAQ_STATE[TPC_CMD] == 1
                press_iso_open = DAQ_STATE[PRESS_ISO_CMD] == 1
                vent_open = DAQ_STATE[VENT_CMD] == 1

                if mpv_open and MPV_LAST_OPEN is None:
                    MPV_LAST_OPEN = sy.TimeStamp.now()
                elif not mpv_open:
                    MPV_LAST_OPEN = None

                l_stand_delta = 0
                scuba_delta = 0

                if press_iso_open:
                    scuba_delta += 2.5

                if (
                    tpc_open
                    and scuba_pressure > 0
                    and not l_stand_pressure > scuba_pressure
                ):
                    scuba_delta -= 1
                    l_stand_delta += 1

                if not vent_open:
                    l_stand_delta -= 1.5

                if not vent_open and tpc_open:
                    scuba_delta -= 1

                if mpv_open:
                    l_stand_delta -= (
                        0.1 * sy.TimeSpan(sy.TimeStamp.now() - MPV_LAST_OPEN).seconds
                    )

                scuba_pressure += scuba_delta
                l_stand_pressure += l_stand_delta
                if scuba_pressure < 0:
                    scuba_pressure = 0
                if l_stand_pressure < 0:
                    l_stand_pressure = 0

                now = sy.TimeStamp.now()

                ok = w.write(
                    {
                        DAQ_TIME: now,
                        TPC_ACK: int(tpc_open),
                        MPV_ACK: int(mpv_open),
                        PRESS_ISO_ACK: int(press_iso_open),
                        VENT_ACK: int(vent_open),
                        PRESS_TANK_PT: scuba_pressure,
                        FUEL_TANK_PT: l_stand_pressure,
                    }
                )

                i += 1
                if (i % 40) == 0:
                    print(f"Committing {i} samples")
                    ok = w.commit()

            except Exception as e:
                print(e)
                raise e
