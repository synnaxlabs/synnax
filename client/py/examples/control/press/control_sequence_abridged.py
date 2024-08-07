import synnax as sy
import time

client = sy.Synnax()

PRESS_VALVE = "valve_command_0"
VENT_VALVE = "valve_command_1"
PRESSURE = "sensor_0"

with client.control.acquire(
    name="Press Sequence",
    write=[PRESS_VALVE, VENT_VALVE],
    read=[PRESSURE],
) as controller:
    start = sy.TimeStamp.now()
    controller[VENT_VALVE] = False
    curr_target = 20

    for i in range(2):
        controller[PRESS_VALVE] = True
        controller.wait_until(lambda c: c[PRESSURE] > curr_target)
        controller[PRESS_VALVE] = False
        time.sleep(2)
        curr_target += 20

    controller[VENT_VALVE] = True
    controller.wait_until(lambda c: c[PRESSURE] < 5)
    controller[VENT_VALVE] = False
    end = sy.TimeStamp.now()

    client.ranges.create(
        name=f"Auto Pressurization Sequence {end}",
        time_range=sy.TimeRange(
            start=start,
            end=sy.TimeStamp.now(),
        ),
    )
