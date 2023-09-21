import time

import synnax as sy

client = sy.Synnax()

with client.control.acquire(
    name="Press Sequence",
    write_authorities=[sy.Authority.ABSOLUTE - 1],
    write=["press_en_cmd"],
    read=["pressure"],
) as auto:
    curr_target = 100
    while True:
        print("Enabling Press")
        auto['press_en_cmd'] = True
        print("Press Enabled")
        auto.wait_until(lambda c: c.pressure > curr_target or c.pressure < 1)
        auto['press_en_cmd'] = False
        curr_target += 100
        time.sleep(5)








