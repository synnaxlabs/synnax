import time

import synnax as sy

client = sy.Synnax()

with client.control.acquire(
    name="Press Sequence",
    write_authorities=[sy.Authority.ABSOLUTE - 1],
    write=["press_en_cmd", "vent_en_cmd"],
    read=["pressure"],
) as auto:
    curr_target = 100
    auto["vent_en_cmd"] = False
    while True:
        auto["press_en_cmd"] = True
        if auto.wait_until(
            lambda c: c.pressure > curr_target or c.pressure < 1,
            timeout=10 * sy.TimeSpan.SECOND,
        ):
            curr_target += 100
        auto["press_en_cmd"] = False
        auto["vent_en_cmd"] = False
        time.sleep(5)
