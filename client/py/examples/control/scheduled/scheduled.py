import synnax as sy
from synnax.control import ScheduledCommand, Controller


client = sy.Synnax(
    host="localhost",
    port=9093,
    username="synnax",
    password="seldon",
    secure=False
)

OX_PRE_VLV_CMD = "ox_pre_vlv_cmd"
OX_MPV_CMD = "ox_mpv_cmd"
OX_MPV_STATE = "ox_mpv_state"
AUTO_LOGS = "auto_logs"
START_SCHEDULED_CMD = "start_scheduled_cmd"

client.channels.create(
    name=AUTO_LOGS,
    virtual=True,
    data_type=sy.DataType.STRING,
    retrieve_if_name_exists=True
)

client.channels.create(
    name=START_SCHEDULED_CMD,
    data_type=sy.DataType.UINT8,
    virtual=True,
    retrieve_if_name_exists=True
)

def log(aut: Controller, msg: str):
    aut.set(
        AUTO_LOGS,
        f"TPC  {sy.TimeStamp.now().datetime().strftime('%H:%M:%S.%f')}  {msg}",
    )

with client.control.acquire(
    name="Scheduled Sequence",
    read=[START_SCHEDULED_CMD, OX_MPV_STATE],
    write=[OX_PRE_VLV_CMD, OX_MPV_CMD, AUTO_LOGS],
    write_authorities=100,
) as c:
    start, ok = c.schedule(
        ScheduledCommand(
            channel=OX_PRE_VLV_CMD,
            value=1,
            delay=0,
        ),
        ScheduledCommand(
            channel=OX_MPV_CMD,
            value=1,
            delay=500 * sy.TimeSpan.MILLISECOND,
        ),
    )
    print(ok)
    if ok:
        log(c, "Sequence scheduled")
        c.wait_until_defined(START_SCHEDULED_CMD)

    log(c, "Sequence started")
    start_t = sy.TimeStamp.now()
    start()
    client.ranges.create(
        name="Test",
        time_range=sy.TimeRange(
            start=start_t,
            end=sy.TimeStamp.now(),
        ),
        color="#FF0000",
    )
    c.wait_until(lambda s: s[OX_MPV_STATE] == 1)
    log(c, "Sequence completed")


